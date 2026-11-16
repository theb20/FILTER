// Package problem écrit les erreurs HTTP au format RFC 9457 (application/problem+json) et applique
// la table de correspondance gRPC → HTTP du gateway (règle G8).
package problem

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"google.golang.org/grpc/codes"
)

// ContentType est le type de média des erreurs.
const ContentType = "application/problem+json"

// FieldError est un élément de `errors[]`.
type FieldError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// Problem est le corps d'erreur du contrat (`components.schemas.Problem`).
type Problem struct {
	Type    string       `json:"type,omitempty"`
	Title   string       `json:"title"`
	Status  int          `json:"status"`
	Detail  string       `json:"detail,omitempty"`
	Code    string       `json:"code"`
	TraceID string       `json:"traceId,omitempty"`
	Errors  []FieldError `json:"errors,omitempty"`

	retryAfterSeconds int
}

// HTTPStatus applique la règle G8 : famille gRPC → statut HTTP.
func HTTPStatus(c codes.Code) int {
	switch c {
	case codes.InvalidArgument:
		return http.StatusUnprocessableEntity // 422 : règle de validation métier ; 400 est réservé aux requêtes malformées (rejetées avant)
	case codes.FailedPrecondition:
		return http.StatusUnprocessableEntity
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// From construit un Problem depuis n'importe quelle erreur. Le détail interne n'est JAMAIS exposé :
// une erreur inconnue donne un message générique.
func From(err error, traceID string) Problem {
	ae := apperr.FromGRPC(err)
	st := HTTPStatus(ae.Kind)
	p := Problem{Title: http.StatusText(st), Status: st, Code: ae.Code, TraceID: traceID}
	if st >= 500 {
		p.Detail = "Une erreur est survenue. Communiquez l'identifiant de trace au support."
		if ae.Kind == codes.Unimplemented {
			p.Detail = "Cette opération n'est pas encore disponible."
		}
	} else {
		p.Detail = ae.Message
	}
	for _, f := range ae.Fields {
		p.Errors = append(p.Errors, FieldError{Code: f.Code, Message: f.Message, Field: f.Pointer})
	}
	if ae.RetryAfter > 0 {
		p.retryAfterSeconds = int((ae.RetryAfter + 999_999_999) / 1_000_000_000) // arrondi supérieur
	}
	return p
}

// New construit un Problem explicite (hors erreurs gRPC).
func New(status int, code, detail, traceID string) Problem {
	return Problem{Title: http.StatusText(status), Status: status, Code: code, Detail: detail, TraceID: traceID}
}

// WithRetryAfter fixe l'en-tête Retry-After (secondes).
func (p Problem) WithRetryAfter(seconds int) Problem { p.retryAfterSeconds = seconds; return p }

// Write écrit la réponse.
func Write(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", ContentType)
	w.Header().Set("Cache-Control", "no-store")
	if p.retryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(p.retryAfterSeconds))
	}
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}
