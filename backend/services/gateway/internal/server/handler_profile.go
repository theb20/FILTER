package server

import (
	"context"

	"github.com/google/uuid"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Profil, adresse e-mail, entreprise et consentements du compte CONNECTÉ. L'identifiant du compte vient TOUJOURS du
// jeton vérifié (jamais du corps ni de l'URL) : personne ne peut lire ni modifier le compte d'un autre.

func optStr(o api.OptString) *string {
	if !o.IsSet() {
		return nil
	}
	v := o.Value
	return &v
}

func (h *handler) UpdateMe(ctx context.Context, req *api.MeUpdate) (api.UpdateMeRes, error) {
	in := &identityv1.UpdateUserRequest{Id: claimsFrom(ctx).Subject, FirstName: optStr(req.FirstName), LastName: optStr(req.LastName), PreferredCurrency: optStr(req.PreferredCurrency)}
	if req.Phone.IsSet() {
		v := string(req.Phone.Value)
		in.PhoneE164 = &v
	}
	if req.Locale.IsSet() {
		v := string(req.Locale.Value)
		in.Locale = &v
	}
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.UpdateUser(ctx2, in)
	if err != nil {
		return nil, err
	}
	me := toMe(res.GetUser())
	return &me, nil
}

func (h *handler) RequestEmailChange(ctx context.Context, req *api.EmailChangeRequest) (api.RequestEmailChangeRes, error) {
	in := &identityv1.RequestEmailChangeRequest{UserId: claimsFrom(ctx).Subject, NewEmail: req.NewEmail}
	if req.Password.IsSet() && !req.Password.Null {
		in.Password = &req.Password.Value
	}
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.RequestEmailChange(ctx2, in); err != nil {
		return nil, err
	}
	return &api.Accepted{Status: api.AcceptedStatusAccepted}, nil
}

func (h *handler) ConfirmEmailChange(ctx context.Context, req *api.VerifyEmailRequest) (api.ConfirmEmailChangeRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.ConfirmEmailChange(ctx2, &identityv1.ConfirmEmailChangeRequest{Token: req.Token}); err != nil {
		return nil, err
	}
	return &api.ConfirmEmailChangeNoContent{}, nil
}

// ── entreprise ──

func (h *handler) GetMyCompany(ctx context.Context) (api.GetMyCompanyRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.GetCompany(ctx2, &identityv1.GetCompanyRequest{UserId: claimsFrom(ctx).Subject})
	if err != nil {
		return nil, err
	}
	out := toCompany(res.GetCompany())
	return &out, nil
}

func (h *handler) PutMyCompany(ctx context.Context, req *api.CompanyInput) (api.PutMyCompanyRes, error) {
	in := &identityv1.CompanyInput{LegalName: req.LegalName, TaxId: req.TaxId, Country: string(req.Country)}
	if req.JobTitle.IsSet() && !req.JobTitle.Null {
		in.JobTitle = &req.JobTitle.Value
	}
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.PutCompany(ctx2, &identityv1.PutCompanyRequest{UserId: claimsFrom(ctx).Subject, Company: in})
	if err != nil {
		return nil, err
	}
	out := toCompany(res.GetCompany())
	return &out, nil
}

// ── consentements ──

func fromPurposes(p api.ConsentInputPurposes) *identityv1.ConsentPurposes {
	return &identityv1.ConsentPurposes{Necessary: p.Necessary, Preferences: p.Preferences.Or(false), Functional: p.Functional.Or(false), Analytics: p.Analytics.Or(false), Marketing: p.Marketing.Or(false)}
}

func toConsent(r *identityv1.ConsentRecord) api.ConsentRecord {
	id, _ := uuid.Parse(r.GetId())
	p := r.GetPurposes()
	return api.ConsentRecord{
		ID: id, PolicyVersion: r.GetPolicyVersion(), RecordedAt: r.GetRecordedAt().AsTime(),
		Purposes: api.ConsentRecordPurposes{Necessary: p.GetNecessary(), Preferences: api.NewOptBool(p.GetPreferences()), Functional: api.NewOptBool(p.GetFunctional()), Analytics: api.NewOptBool(p.GetAnalytics()), Marketing: api.NewOptBool(p.GetMarketing())},
	}
}

func (h *handler) recordConsent(ctx context.Context, userID string, req *api.ConsentInput) (api.ConsentRecord, error) {
	info := infoFrom(ctx)
	in := &identityv1.RecordConsentRequest{Purposes: fromPurposes(req.Purposes), PolicyVersion: req.PolicyVersion, Locale: string(req.Locale.Or(api.Locale(info.Locale))), ClientIp: info.IP}
	if userID != "" {
		in.UserId = &userID
	}
	if req.AnonymousId.IsSet() && !req.AnonymousId.Null {
		in.AnonymousId = &req.AnonymousId.Value
	}
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.RecordConsent(ctx2, in)
	if err != nil {
		return api.ConsentRecord{}, err
	}
	return toConsent(res.GetRecord()), nil
}

func (h *handler) RecordAnonymousConsent(ctx context.Context, req *api.ConsentInput) (api.RecordAnonymousConsentRes, error) {
	out, err := h.recordConsent(ctx, "", req) // visiteur : jamais rattaché à un compte, même si un jeton traînait
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (h *handler) PutMyConsents(ctx context.Context, req *api.ConsentInput) (api.PutMyConsentsRes, error) {
	out, err := h.recordConsent(ctx, claimsFrom(ctx).Subject, req)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (h *handler) GetMyConsents(ctx context.Context) (api.GetMyConsentsRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.GetConsents(ctx2, &identityv1.GetConsentsRequest{UserId: claimsFrom(ctx).Subject})
	if err != nil {
		return nil, err
	}
	out := toConsent(res.GetRecord())
	return &out, nil
}
