package problem

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"google.golang.org/grpc/codes"
)

func TestCorrespondanceG8(t *testing.T) {
	cases := map[codes.Code]int{
		codes.InvalidArgument: 422, codes.FailedPrecondition: 422, codes.NotFound: 404, codes.AlreadyExists: 409, codes.Aborted: 409,
		codes.PermissionDenied: 403, codes.Unauthenticated: 401, codes.ResourceExhausted: 429, codes.Unavailable: 503,
		codes.DeadlineExceeded: 504, codes.Internal: 500, codes.Unimplemented: 501,
	}
	for c, want := range cases {
		if got := HTTPStatus(c); got != want {
			t.Errorf("%v -> %d, attendu %d", c, got, want)
		}
	}
}

func TestAllerRetourGRPCEtRetryAfter(t *testing.T) {
	src := apperr.RateLimited(90 * time.Second)
	p := From(apperr.ToGRPC(src), "trace-1")
	if p.Status != 429 || p.Code != apperr.CodeRateLimited || p.TraceID != "trace-1" {
		t.Fatalf("problème inattendu : %+v", p)
	}
	rec := httptest.NewRecorder()
	Write(rec, p)
	if rec.Header().Get("Retry-After") != "90" {
		t.Fatalf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
	if ct := rec.Header().Get("Content-Type"); ct != ContentType {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestChampsEtCodeMetierConserves(t *testing.T) {
	src := apperr.Validation("données invalides", apperr.Field{Pointer: "/password", Code: apperr.CodePasswordCompromised, Message: "mot de passe compromis"})
	p := From(apperr.ToGRPC(src), "")
	if p.Status != http.StatusUnprocessableEntity || len(p.Errors) != 1 || p.Errors[0].Field != "/password" || p.Errors[0].Code != apperr.CodePasswordCompromised {
		t.Fatalf("champs perdus : %+v", p)
	}
}

func TestNExposeJamaisLeDetailInterne(t *testing.T) {
	p := From(apperr.ToGRPC(errString("connexion postgres://user:secret@db/ff refusée")), "t")
	if p.Status != 500 {
		t.Fatalf("statut = %d", p.Status)
	}
	rec := httptest.NewRecorder()
	Write(rec, p)
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "postgres") {
		t.Fatalf("détail interne exposé : %s", rec.Body.String())
	}
}

type errString string

func (e errString) Error() string { return string(e) }
