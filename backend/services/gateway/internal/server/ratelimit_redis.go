package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindow : fenêtre glissante atomique dans Redis (ensemble trié). L'horloge est celle de Redis (TIME) et non
// celle des instances du gateway : un décalage d'horloge entre instances ne fausse donc pas la limite.
// Retourne {autorisé(0/1), restant, attente_ms}.
var slidingWindow = redis.NewScript(`
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - window)
local n = redis.call('ZCARD', KEYS[1])
if n >= limit then
  local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
  local wait = window
  if oldest[2] then wait = tonumber(oldest[2]) + window - now end
  if wait < 1 then wait = 1 end
  return {0, 0, wait}
end
redis.call('ZADD', KEYS[1], now, now .. '-' .. ARGV[3])
redis.call('PEXPIRE', KEYS[1], window + 1000)
return {1, limit - n - 1, 0}
`)

// RedisLimiter : limitation de débit PARTAGÉE entre toutes les instances du gateway.
//
// Panne de Redis : on bascule sur une limitation locale (par instance) et on journalise, au lieu de refuser tout le
// trafic (fail-closed) ou de supprimer toute limite (fail-open). La protection est dégradée (limite × nb d'instances)
// mais jamais absente ; /readyz signale la panne via la sonde « redis ».
type RedisLimiter struct {
	rdb      redis.UniversalClient
	prefix   string
	timeout  time.Duration
	fallback *MemoryLimiter
	log      *slog.Logger
	id       string // identité de l'instance : rend les membres uniques entre instances
	seq      atomic.Uint64
	degraded atomic.Bool
}

// NewRedisLimiter construit le limiteur à partir d'un client Redis déjà configuré.
func NewRedisLimiter(rdb redis.UniversalClient, log *slog.Logger) *RedisLimiter {
	if log == nil {
		log = slog.Default()
	}
	var b [6]byte
	_, _ = rand.Read(b[:])
	return &RedisLimiter{id: hex.EncodeToString(b[:]), rdb: rdb, prefix: "ff:rl:", timeout: 150 * time.Millisecond, fallback: NewMemoryLimiter(), log: log}
}

// Ping vérifie la connexion (démarrage et sonde de disponibilité).
func (l *RedisLimiter) Ping(ctx context.Context) error { return l.rdb.Ping(ctx).Err() }

func (l *RedisLimiter) Allow(key string, limit int, window time.Duration, now time.Time) (bool, int, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), l.timeout)
	defer cancel()
	member := l.id + "-" + strconv.FormatUint(l.seq.Add(1), 36)
	res, err := slidingWindow.Run(ctx, l.rdb, []string{l.prefix + key}, limit, window.Milliseconds(), member).Int64Slice()
	if err != nil || len(res) != 3 {
		if l.degraded.CompareAndSwap(false, true) {
			l.log.Error("Redis indisponible : limitation de débit dégradée (par instance)", "err", err)
		}
		return l.fallback.Allow(key, limit, window, now)
	}
	if l.degraded.CompareAndSwap(true, false) {
		l.log.Info("Redis rétabli : limitation de débit partagée")
	}
	return res[0] == 1, int(res[1]), time.Duration(res[2]) * time.Millisecond
}

// Client expose le client Redis (partagé avec l'idempotence).
func (l *RedisLimiter) Client() redis.UniversalClient { return l.rdb }
