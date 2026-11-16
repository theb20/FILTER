package server

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/jwtx"
)

type ctxKey int

const (
	keyInfo ctxKey = iota
	keyClaims
	keyRefreshCookie
	keyBearer
)

// reqInfo est propagé aux handlers (l'API ogen ne leur donne pas *http.Request).
type reqInfo struct {
	TraceID string
	IP      string
	UA      string
	Locale  string
}

func withInfo(ctx context.Context, i reqInfo) context.Context {
	return context.WithValue(ctx, keyInfo, i)
}
func infoFrom(ctx context.Context) reqInfo {
	i, _ := ctx.Value(keyInfo).(reqInfo)
	return i
}

func withClaims(ctx context.Context, c *jwtx.Claims) context.Context {
	return context.WithValue(ctx, keyClaims, c)
}
func claimsFrom(ctx context.Context) *jwtx.Claims {
	c, _ := ctx.Value(keyClaims).(*jwtx.Claims)
	return c
}

func withRefreshCookie(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, keyRefreshCookie, v)
}
func refreshCookieFrom(ctx context.Context) string {
	v, _ := ctx.Value(keyRefreshCookie).(string)
	return v
}

// withBearer conserve le jeton de défi 2FA (déjà vérifié localement) pour le transmettre à identity, qui le consomme.
func withBearer(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, keyBearer, v)
}
func bearerFrom(ctx context.Context) string {
	v, _ := ctx.Value(keyBearer).(string)
	return v
}
