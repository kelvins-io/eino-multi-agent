package a2a

import (
	"context"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

func bearerMiddleware(token string) app.HandlerFunc {
	token = strings.TrimSpace(token)
	return func(c context.Context, ctx *app.RequestContext) {
		if token == "" {
			ctx.Next(c)
			return
		}
		auth := strings.TrimSpace(string(ctx.GetHeader("Authorization")))
		got := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if got == "" {
			got = auth
		}
		if got != token {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		ctx.Next(c)
	}
}

type bearerRoundTripper struct {
	base  http.RoundTripper
	token string
}

func (b bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := b.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	if b.token != "" {
		clone.Header.Set("Authorization", "Bearer "+b.token)
	}
	return base.RoundTrip(clone)
}

func httpClientWithBearer(token string) *http.Client {
	return &http.Client{
		Transport: bearerRoundTripper{token: strings.TrimSpace(token)},
	}
}
