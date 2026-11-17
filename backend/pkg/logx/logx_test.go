package logx

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestExpurgeLesSecrets(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "test", slog.LevelInfo)
	l.Info("connexion", "email", "a@b.ci", "password", "hunter2", "refresh_token", "abc", "Authorization", "Bearer xyz", "reset_code", "123456")
	out := buf.String()
	for _, secret := range []string{"hunter2", "abc", "Bearer xyz", "123456"} {
		if strings.Contains(out, secret) {
			t.Errorf("le secret %q fuit dans les logs : %s", secret, out)
		}
	}
	if !strings.Contains(out, "a@b.ci") {
		t.Errorf("les champs non sensibles doivent rester : %s", out)
	}
}
