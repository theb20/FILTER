package server

import (
	"context"
	"errors"
	"net/http"

	ht "github.com/ogen-go/ogen/http"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/ogen-go/ogen/validate"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/problem"
	"google.golang.org/grpc/codes"
)

// errorHandler convertit TOUTE erreur (décodage, sécurité, gRPC, non implémentée) en RFC 9457. Aucun détail interne
// n'est exposé.
func (s *Server) errorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	trace := infoFrom(r.Context()).TraceID
	var p problem.Problem
	var (
		secErr *ogenerrors.SecurityError
		decReq *ogenerrors.DecodeRequestError
		decPar *ogenerrors.DecodeParamsError
		valErr *validate.Error
	)
	switch {
	case errors.As(err, &secErr), errors.Is(err, ogenerrors.ErrSecurityRequirementIsNotSatisfied):
		// la cause porte déjà le bon code (jeton invalide) ; sinon : authentification requise
		var ae *apperr.Error
		if secErr != nil && errors.As(secErr.Err, &ae) {
			p = problem.From(ae, trace)
		} else {
			p = problem.From(apperr.Unauthenticated(apperr.CodeUnauthenticated, "authentification requise"), trace)
		}
	case errors.As(err, &valErr):
		p = problem.From(apperr.Validation("données invalides"), trace)
		p.Errors = append(p.Errors, problem.FieldError{Code: apperr.CodeValidation, Message: "une valeur ne respecte pas le contrat de l'API"})
	case errors.As(err, &decReq), errors.As(err, &decPar):
		// requête malformée : JSON invalide, champ inconnu, type erroné, en-tête requis absent
		p = problem.New(http.StatusBadRequest, apperr.CodeValidation, "Requête invalide : elle ne respecte pas le contrat de l'API.", trace)
	case errors.Is(err, ht.ErrNotImplemented):
		p = problem.From(apperr.New(codes.Unimplemented, apperr.CodeNotImplemented, "non implémenté"), trace)
	default:
		p = problem.From(err, trace)
	}
	if p.Status >= 500 {
		s.log.Error("erreur serveur", "trace_id", trace, "path", r.URL.Path, "code", p.Code)
	}
	problem.Write(w, p)
}
