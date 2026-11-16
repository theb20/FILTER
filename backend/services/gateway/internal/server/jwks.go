package server

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/theb20/FILTER/backend/pkg/jwtx"
)

// JWKSSource fournit les clés publiques d'une audience (identity.GetJwks).
type JWKSSource interface {
	Fetch(ctx context.Context, audience string) ([]byte, error)
}

// KeyCache vérifie les jetons localement. Le JWKS est rechargé sur `kid` inconnu (au plus une fois par
// `minRefresh`) : une rotation de clé est prise en compte sans redémarrage.
type KeyCache struct {
	src        JWKSSource
	leeway     time.Duration
	minRefresh time.Duration
	now        func() time.Time

	mu       sync.Mutex
	verifier map[string]*jwtx.Verifier
	last     map[string]time.Time
}

func NewKeyCache(src JWKSSource, leeway, minRefresh time.Duration, now func() time.Time) *KeyCache {
	if now == nil {
		now = time.Now
	}
	return &KeyCache{src: src, leeway: leeway, minRefresh: minRefresh, now: now, verifier: map[string]*jwtx.Verifier{}, last: map[string]time.Time{}}
}

func (c *KeyCache) load(ctx context.Context, aud string, force bool) (*jwtx.Verifier, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.verifier[aud]; ok && !force {
		return v, nil
	}
	if force && c.now().Sub(c.last[aud]) < c.minRefresh {
		return c.verifier[aud], nil // trop tôt pour recharger : on garde l'existant
	}
	raw, err := c.src.Fetch(ctx, aud)
	if err != nil {
		return c.verifier[aud], err
	}
	v, err := jwtx.NewVerifierFromJWKS(raw, aud, c.leeway)
	if err != nil {
		return c.verifier[aud], err
	}
	c.verifier[aud], c.last[aud] = v, c.now()
	return v, nil
}

// Verify contrôle un jeton de l'audience donnée.
func (c *KeyCache) Verify(ctx context.Context, aud, token string) (*jwtx.Claims, error) {
	v, err := c.load(ctx, aud, false)
	if v == nil {
		return nil, err
	}
	claims, verr := v.Verify(token, c.now())
	if errors.Is(verr, jwtx.ErrUnknownKey) {
		if v2, _ := c.load(ctx, aud, true); v2 != nil {
			return v2.Verify(token, c.now())
		}
	}
	return claims, verr
}
