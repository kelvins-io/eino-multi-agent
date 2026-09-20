package api

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	ctxUserID   = "auth_user_id"
	ctxUsername = "auth_username"
)

type jwtClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type authRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func (s *Server) register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	username, password, err := normalizeAuth(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	displayName, err := normalizeDisplayName(req.DisplayName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password failed"})
		return
	}
	user := &store.User{
		ID:           uuid.NewString(),
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: string(hash),
	}
	if err := s.store.CreateUser(c.Request.Context(), user); err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	token, expiresAt, err := s.issueToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Set(ctxUserID, user.ID)
	c.Set(ctxUsername, user.Username)
	s.audit(c, "auth.register", "user", user.ID, user.Username)
	c.JSON(http.StatusCreated, gin.H{
		"token":      token,
		"expires_at": expiresAt,
		"user":       publicUser(user),
	})
}

func (s *Server) login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	username, password, err := normalizeAuth(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := s.store.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		if store.IsNotFound(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	token, expiresAt, err := s.issueToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Set(ctxUserID, user.ID)
	c.Set(ctxUsername, user.Username)
	s.audit(c, "auth.login", "user", user.ID, user.Username)
	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
		"user":       publicUser(user),
	})
}

func (s *Server) me(c *gin.Context) {
	userID, _ := c.Get(ctxUserID)
	id, _ := userID.(string)
	if id == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	user, err := s.store.GetUserByID(c.Request.Context(), id)
	if err != nil {
		if store.IsNotFound(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, publicUser(user))
}

func (s *Server) issueToken(user *store.User) (string, time.Time, error) {
	expire := s.cfg.Server.JWTExpire
	if expire <= 0 {
		expire = 168 * time.Hour
	}
	expiresAt := time.Now().Add(expire)
	claims := jwtClaims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.Server.JWTSecret))
	return signed, expiresAt, err
}

func (s *Server) parseToken(raw string) (*jwtClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.Server.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Subject == "" {
		return nil, errors.New("missing subject")
	}
	return claims, nil
}

func normalizeAuth(username, password string) (string, string, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return "", "", errors.New("username and password are required")
	}
	if utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 64 {
		return "", "", errors.New("username length must be 3-64")
	}
	if utf8.RuneCountInString(password) < 6 || utf8.RuneCountInString(password) > 128 {
		return "", "", errors.New("password length must be 6-128")
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", "", errors.New("username may only contain letters, digits, _ and -")
	}
	return username, password, nil
}

func normalizeDisplayName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("display_name is required")
	}
	n := utf8.RuneCountInString(name)
	if n < 1 || n > 64 {
		return "", errors.New("display_name length must be 1-64")
	}
	return name, nil
}

func publicUser(u *store.User) gin.H {
	display := u.DisplayName
	if display == "" {
		display = u.Username
	}
	return gin.H{
		"id":           u.ID,
		"username":     u.Username,
		"display_name": display,
		"created_at":   u.CreatedAt,
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}

func (s *Server) requireAuth() gin.HandlerFunc {
	staticToken := strings.TrimSpace(s.cfg.Server.AuthToken)
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api/") || isPublicAPI(path) {
			c.Next()
			return
		}
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			raw = c.Query("token")
		}
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if staticToken != "" && raw == staticToken {
			c.Set(ctxUsername, "static-token")
			c.Next()
			return
		}
		claims, err := s.parseToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set(ctxUserID, claims.Subject)
		c.Set(ctxUsername, claims.Username)
		c.Next()
	}
}

func isPublicAPI(path string) bool {
	switch path {
	case "/api/v1/health",
		"/api/v1/auth/register",
		"/api/v1/auth/login",
		"/api/v1/hooks/echo":
		return true
	default:
		return false
	}
}

func bearerToken(h string) string {
	const p = "Bearer "
	if strings.HasPrefix(h, p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

func operatorOf(c *gin.Context) string {
	if v, ok := c.Get(ctxUsername); ok {
		if s, _ := v.(string); strings.TrimSpace(s) != "" {
			return s
		}
	}
	if v := strings.TrimSpace(c.GetHeader("X-Operator")); v != "" {
		return v
	}
	return "anonymous"
}

func (s *Server) requireUser(c *gin.Context) (string, bool) {
	userID := strings.TrimSpace(c.GetString(ctxUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return "", false
	}
	return userID, true
}

func notFoundOrErr(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	if store.IsNotFound(err) {
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

