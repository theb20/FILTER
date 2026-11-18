package server_test

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
)

func newRedisLimiter(t *testing.T) (*server.RedisLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return server.NewRedisLimiter(rdb, slog.New(slog.NewTextHandler(io.Discard, nil))), mr
}

func TestRedisLimiter_FenetreEtRestant(t *testing.T) {
	l, mr := newRedisLimiter(t)
	for i := 0; i < 3; i++ {
		ok, rem, _ := l.Allow("k", 3, time.Minute, time.Now())
		if !ok || rem != 2-i {
			t.Fatalf("appel %d : ok=%v restant=%d", i, ok, rem)
		}
	}
	ok, rem, retry := l.Allow("k", 3, time.Minute, time.Now())
	if ok || rem != 0 || retry <= 0 || retry > time.Minute {
		t.Fatalf("4e appel doit être refusé : ok=%v restant=%d retry=%v", ok, rem, retry)
	}
	if ok, _, _ := l.Allow("autre", 3, time.Minute, time.Now()); !ok {
		t.Fatal("une autre clé ne doit pas être touchée")
	}
	mr.SetTime(time.Now().Add(2 * time.Minute)) // la fenêtre glisse
	mr.FastForward(2 * time.Minute)
	if ok, _, _ := l.Allow("k", 3, time.Minute, time.Now()); !ok {
		t.Fatal("après la fenêtre, l'accès doit être rétabli")
	}
}

// Deux instances du gateway (deux limiteurs) partagent le même compteur : c'est tout l'intérêt de Redis.
func TestRedisLimiter_PartageEntreInstances(t *testing.T) {
	a, mr := newRedisLimiter(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	b := server.NewRedisLimiter(rdb, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for i := 0; i < 5; i++ {
		l := a
		if i%2 == 1 {
			l = b
		}
		if ok, _, _ := l.Allow("login:1.2.3.4", 5, time.Minute, time.Now()); !ok {
			t.Fatalf("appel %d refusé trop tôt", i)
		}
	}
	if ok, _, _ := a.Allow("login:1.2.3.4", 5, time.Minute, time.Now()); ok {
		t.Fatal("a : la limite doit être partagée")
	}
	if ok, _, _ := b.Allow("login:1.2.3.4", 5, time.Minute, time.Now()); ok {
		t.Fatal("b : la limite doit être partagée")
	}
}

func TestRedisLimiter_ConcurrenceNeDepassePasLaLimite(t *testing.T) {
	l, _ := newRedisLimiter(t)
	const limit, workers = 10, 60
	res := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		go func() { ok, _, _ := l.Allow("c", limit, time.Minute, time.Now()); res <- ok }()
	}
	n := 0
	for i := 0; i < workers; i++ {
		if <-res {
			n++
		}
	}
	if n != limit {
		t.Fatalf("%d autorisées pour une limite de %d", n, limit)
	}
}

// Redis tombe : repli par instance (limite toujours appliquée), puis retour au partage.
func TestRedisLimiter_PanneRepliLocal(t *testing.T) {
	l, mr := newRedisLimiter(t)
	mr.Close()
	for i := 0; i < 2; i++ {
		if ok, _, _ := l.Allow("p", 2, time.Minute, time.Now()); !ok {
			t.Fatalf("repli : appel %d refusé trop tôt", i)
		}
	}
	if ok, _, _ := l.Allow("p", 2, time.Minute, time.Now()); ok {
		t.Fatal("repli : la limite locale doit s'appliquer pendant la panne")
	}
}
