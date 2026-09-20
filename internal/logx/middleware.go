package logx

import (
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GinLogger writes an access log line for each request.
func GinLogger() gin.HandlerFunc {
	log := Named("http")
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("size", c.Writer.Size()),
		}
		if query != "" {
			fields = append(fields, zap.String("query", query))
		}
		if uid := c.GetString("auth_user_id"); uid != "" {
			fields = append(fields, zap.String("user_id", uid))
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()))
		}

		switch {
		case path == "/api/v1/health":
			log.Debug("request", fields...)
		case status >= 500:
			log.Error("request", fields...)
		case status >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

// GinRecovery recovers panics and logs the stack with zap.
func GinRecovery() gin.HandlerFunc {
	log := Named("http")
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						msg := strings.ToLower(se.Error())
						brokenPipe = strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
					}
				}
				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					log.Error("broken pipe",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					_ = c.Error(err.(error))
					c.Abort()
					return
				}
				log.Error("panic recovered",
					zap.Any("error", err),
					zap.String("request", string(httpRequest)),
					zap.ByteString("stack", debug.Stack()),
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
