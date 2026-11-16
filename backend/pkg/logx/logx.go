// Package logx fournit un logger JSON structuré qui EXPURGE les secrets (règle G15) : jamais de mot de passe,
// jeton, code de réinitialisation ni en-tête Authorization dans les logs.
package logx

import (
	"io"
	"log/slog"
	"strings"
)

var sensitive = []string{"password", "passwd", "token", "secret", "authorization", "cookie", "code", "key", "otp", "recovery"}

// New crée un logger JSON. `service` est ajouté à chaque ligne.
func New(w io.Writer, service string, level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level, ReplaceAttr: redact})
	return slog.New(h).With("service", service)
}

func redact(_ []string, a slog.Attr) slog.Attr {
	k := strings.ToLower(a.Key)
	for _, s := range sensitive {
		if strings.Contains(k, s) {
			return slog.String(a.Key, "[REDACTED]")
		}
	}
	return a
}
