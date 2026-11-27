package mcp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
)

type jwtClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// UserFromRequest 从 Authorization Bearer JWT 解析用户；静态 auth_token 不可用（无 user_id）。
func UserFromRequest(cfg *config.Config, r *http.Request) (userID, username string, err error) {
	raw := bearerToken(r.Header.Get("Authorization"))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if raw == "" {
		return "", "", errors.New("login required: Authorization Bearer JWT")
	}
	static := strings.TrimSpace(cfg.Server.AuthToken)
	if static != "" && raw == static {
		return "", "", errors.New("login required: MCP needs a user JWT, not static auth_token")
	}
	claims, err := parseJWT(cfg.Server.JWTSecret, raw)
	if err != nil {
		return "", "", errors.New("unauthorized")
	}
	return claims.Subject, claims.Username, nil
}

func parseJWT(secret, raw string) (*jwtClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
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

func bearerToken(h string) string {
	const p = "Bearer "
	if strings.HasPrefix(h, p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}
