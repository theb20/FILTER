package server

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	pb "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/protobuf/types/known/structpb"
)

// Brouillon de demande de devis du client connecté : un seul par compte, conservé côté serveur (il le retrouve sur tous ses
// appareils, dans « Mon compte »). Stocké par le service content (collection quote_draft, clé = identifiant du compte).

const draftKind = "quote_draft"

func (h *handler) draftOwner(ctx context.Context) (string, error) {
	c := claimsFrom(ctx)
	if c == nil || c.Subject == "" {
		return "", apperr.Unauthenticated(apperr.CodeUnauthenticated, "authentification requise")
	}
	return c.Subject, nil
}

func (h *handler) draftOf(ctx context.Context, c pb.ContentAdminServiceClient, owner string) (*pb.Document, error) {
	res, err := c.GetDocument(ctx, &pb.GetDocumentRequest{Kind: draftKind, IdOrKey: owner})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return res.GetDocument(), nil
}

func toQuoteDraft(d *pb.Document) (*api.QuoteDraft, error) {
	m := d.GetData().AsMap()
	delete(m, "userId")
	m["updatedAt"] = d.GetUpdatedAt().AsTime().Format("2006-01-02T15:04:05.999Z07:00")
	b, _ := json.Marshal(m)
	var out api.QuoteDraft
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, apperr.Internal(err)
	}
	return &out, nil
}

func (h *handler) GetMyQuoteDraft(ctx context.Context) (api.GetMyQuoteDraftRes, error) {
	owner, err := h.draftOwner(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	d, err := h.draftOf(ctx, c, owner)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, apperr.NotFound("aucun brouillon")
	}
	return toQuoteDraft(d)
}

func (h *handler) PutMyQuoteDraft(ctx context.Context, req *api.QuoteDraft) (api.PutMyQuoteDraftRes, error) {
	owner, err := h.draftOwner(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	data, err := inputStruct(req)
	if err != nil {
		return nil, err
	}
	delete(data.Fields, "updatedAt") // lecture seule
	data.Fields["userId"] = structpb.NewStringValue(owner)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	cur, err := h.draftOf(ctx, c, owner)
	if err != nil {
		return nil, err
	}
	var d *pb.Document
	if cur == nil {
		res, err := c.CreateDocument(ctx, &pb.CreateDocumentRequest{Kind: draftKind, Data: data})
		if err != nil {
			return nil, err
		}
		d = res.GetDocument()
	} else {
		res, err := c.UpdateDocument(ctx, &pb.UpdateDocumentRequest{Kind: draftKind, Id: cur.GetId(), Data: data, Replace: true})
		if err != nil {
			return nil, err
		}
		d = res.GetDocument()
	}
	return toQuoteDraft(d)
}

func (h *handler) DeleteMyQuoteDraft(ctx context.Context) (api.DeleteMyQuoteDraftRes, error) {
	owner, err := h.draftOwner(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	d, err := h.draftOf(ctx, c, owner)
	if err != nil {
		return nil, err
	}
	if d != nil {
		if _, err := c.DeleteDocument(ctx, &pb.DeleteDocumentRequest{Kind: draftKind, Id: d.GetId()}); err != nil {
			return nil, err
		}
	}
	return &api.DeleteMyQuoteDraftNoContent{}, nil
}

// ListMyQuoteRequests : suivi des demandes de devis du client. Elles se retrouvent par l'adresse e-mail du compte ; seule une
// adresse VÉRIFIÉE y donne accès (sinon, créer un compte avec l'adresse d'un tiers suffirait à lire ses demandes).
func (h *handler) ListMyQuoteRequests(ctx context.Context) (api.ListMyQuoteRequestsRes, error) {
	owner, err := h.draftOwner(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	u, err := h.id.GetUser(ctx, &identityv1.GetUserRequest{Id: owner})
	if err != nil {
		return nil, err
	}
	if !u.GetUser().GetEmailVerified() {
		return nil, apperr.PermissionDenied("EMAIL_NOT_VERIFIED", "confirmez votre adresse e-mail pour suivre vos demandes")
	}
	out := &api.MyQuoteRequestList{Items: []api.MyQuoteRequest{}}
	for p := int32(1); p < 20; p++ {
		res, err := c.ListDocuments(ctx, &pb.ListDocumentsRequest{Kind: "quote_request", Page: &commonv1.PageRequest{Page: p, PageSize: 100}, Where: map[string]string{"email": strings.ToLower(u.GetUser().GetEmail())}})
		if err != nil {
			return nil, err
		}
		for _, d := range res.GetItems() {
			m := d.GetData().AsMap()
			it := api.MyQuoteRequest{ID: uuidOf(d.GetId()), Status: api.MyQuoteRequestStatus(str(m, "status")), CreatedAt: d.GetCreatedAt().AsTime(), UpdatedAt: d.GetUpdatedAt().AsTime()}
			if l, ok := m["items"].([]any); ok {
				it.ItemCount = int64(len(l))
			}
			if v := str(m, "subject"); v != "" {
				it.Subject = api.NewOptString(v)
			}
			if v := str(m, "reference"); v != "" {
				it.Reference = api.NewOptString(v)
			}
			if v := str(m, "company"); v != "" {
				it.Company = api.NewOptString(v)
			}
			// Le devis chiffré n'est visible du client qu'une fois ENVOYÉ (les brouillons et les notes internes ne sortent jamais).
			if st := str(m, "status"); st == "quoted" || st == "closed" {
				if r, err := c.GetDocument(ctx, &pb.GetDocumentRequest{Kind: "quote_response", IdOrKey: d.GetId()}); err == nil {
					if q := publicQuote(r.GetDocument()); q != nil {
						it.Quote = api.NewOptMyQuote(*q)
					}
				}
			}
			out.Items = append(out.Items, it)
		}
		if p >= res.GetPage().GetTotalPages() {
			break
		}
	}
	return out, nil
}

func fnum(m map[string]any, k string) float64 { v, _ := m[k].(float64); return v }

// publicQuote : la vue client d'une réponse envoyée. Construite champ par champ : une nouvelle donnée interne n'y entre jamais par accident.
func publicQuote(d *pb.Document) *api.MyQuote {
	m := d.GetData().AsMap()
	if str(m, "status") != "sent" {
		return nil
	}
	q := &api.MyQuote{Lines: []api.MyQuoteLine{}, Total: int64(fnum(m, "total"))}
	opt := func(k string, set func(string)) {
		if v := str(m, k); v != "" {
			set(v)
		}
	}
	opt("number", func(v string) { q.Number = api.NewOptString(v) })
	opt("intro", func(v string) { q.Intro = api.NewOptString(v) })
	opt("deliveryPlace", func(v string) { q.DeliveryPlace = api.NewOptString(v) })
	opt("deliveryDelay", func(v string) { q.DeliveryDelay = api.NewOptString(v) })
	opt("paymentTerms", func(v string) { q.PaymentTerms = api.NewOptString(v) })
	opt("version", func(v string) { q.Version = api.NewOptString(v) })
	opt("incoterm", func(v string) { q.Incoterm = api.NewOptString(v) })
	opt("correspondent", func(v string) { q.Correspondent = api.NewOptString(v) })
	q.Other = api.NewOptInt64(int64(fnum(m, "other")))
	q.Acompte = api.NewOptInt64(int64(fnum(m, "acompte")))
	q.Remaining = api.NewOptInt64(int64(fnum(m, "total") - fnum(m, "acompte")))
	if imgs, ok := m["annexes"].([]any); ok {
		for _, x := range imgs {
			im, _ := x.(map[string]any)
			a := api.MyQuoteImage{URL: str(im, "url")}
			if c := str(im, "caption"); c != "" {
				a.Caption = api.NewOptString(c)
			}
			q.Annexes = append(q.Annexes, a)
		}
	}
	if t, err := time.Parse(time.RFC3339, str(m, "sentAt")); err == nil {
		q.SentAt = api.NewOptDateTime(t)
	}
	if t, err := time.Parse(time.RFC3339, str(m, "validUntil")); err == nil {
		q.ValidUntil = api.NewOptDateTime(t)
	}
	q.Subtotal = api.NewOptInt64(int64(fnum(m, "subtotal")))
	q.DiscountPct = api.NewOptFloat64(fnum(m, "discountPct"))
	q.DiscountAmount = api.NewOptInt64(int64(fnum(m, "discountAmount")))
	q.Shipping = api.NewOptInt64(int64(fnum(m, "shipping")))
	q.VatRate = api.NewOptFloat64(fnum(m, "vatRate"))
	q.VatAmount = api.NewOptInt64(int64(fnum(m, "vatAmount")))
	lines, _ := m["lines"].([]any)
	for _, x := range lines {
		l, _ := x.(map[string]any)
		out := api.MyQuoteLine{Reference: str(l, "reference"), Quantity: int64(fnum(l, "quantity"))}
		if v := str(l, "name"); v != "" {
			out.Name = api.NewOptString(v)
		}
		if v := str(l, "leadTime"); v != "" {
			out.LeadTime = api.NewOptString(v)
		}
		if v := str(l, "image"); v != "" {
			out.Image = api.NewOptString(v)
		}
		if v := str(l, "note"); v != "" { // note destinée au client ; « internal » n'est jamais lue
			out.Note = api.NewOptString(v)
		}
		if p, ok := l["unitPrice"].(float64); ok {
			out.UnitPrice = api.NewOptInt64(int64(p))
			out.DiscountPct = api.NewOptFloat64(fnum(l, "discountPct"))
			out.LineTotal = api.NewOptInt64(int64(math.Round(float64(out.Quantity) * p * (1 - fnum(l, "discountPct")/100))))
		}
		if o, _ := l["optional"].(bool); o {
			out.Optional = api.NewOptBool(true)
		}
		q.Lines = append(q.Lines, out)
	}
	return q
}
