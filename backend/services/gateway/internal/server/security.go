package server

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// security implémente api.SecurityHandler : vérification LOCALE des jetons (clés d'identity.GetJwks).
type security struct{ keys *KeyCache }

var _ api.SecurityHandler = (*security)(nil)

func (s *security) verify(ctx context.Context, aud, token string) (context.Context, error) {
	c, err := s.keys.Verify(ctx, aud, token)
	if err != nil {
		return ctx, apperr.Unauthenticated(apperr.CodeUnauthenticated, "jeton invalide ou expiré")
	}
	return withClaims(ctx, c), nil
}

func (s *security) HandleCustomerAuth(ctx context.Context, _ api.OperationName, t api.CustomerAuth) (context.Context, error) {
	return s.verify(ctx, jwtx.AudCustomer, t.Token)
}

// HandleStaffAuth vérifie le jeton PUIS la permission de l'opération (x-required-permission du contrat, table générée).
// Une opération absente de la table est REFUSÉE (fail closed). Les permissions viennent du jeton (rôles au moment de la
// connexion/du renouvellement, 15 min au plus) ; identity les relit en base pour toute action qui accorde des droits.
func (s *security) HandleStaffAuth(ctx context.Context, op api.OperationName, t api.StaffAuth) (context.Context, error) {
	ctx, err := s.verify(ctx, jwtx.AudStaff, t.Token)
	if err != nil {
		return ctx, err
	}
	required, known := api.RequiredPermission[op]
	if !known || (required != "" && !claimsFrom(ctx).HasPermission(required)) {
		return ctx, apperr.PermissionDenied(apperr.CodeForbidden, "permission insuffisante")
	}
	return ctx, nil
}

func (s *security) HandleStaffChallenge(ctx context.Context, _ api.OperationName, t api.StaffChallenge) (context.Context, error) {
	ctx, err := s.verify(ctx, jwtx.AudStaffChallenge, t.Token)
	return withBearer(ctx, t.Token), err
}

// HandleRefreshCookie ne vérifie pas le jeton opaque (c'est identity qui le fait, avec rotation) : il le transmet
// au handler, qui l'envoie à identity. Il n'est jamais journalisé.
func (s *security) HandleRefreshCookie(ctx context.Context, _ api.OperationName, t api.RefreshCookie) (context.Context, error) {
	return withRefreshCookie(ctx, t.APIKey), nil
}
