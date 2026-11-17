package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
	"google.golang.org/grpc/codes"
)

// ── tiers ───────────────────────────────────────────────────────────────────

func (s *Service) CreateThirdParty(ctx context.Context, a Actor, in domain.ThirdParty) (*domain.ThirdParty, error) {
	in.ID, in.Active = s.NewID(), true
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateThirdParty(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.third_party.create", "finance_third_party", in.ID, nil, map[string]any{"kind": in.Kind, "name": in.DisplayName, "ref": strOrNil(in.ExternalRef)})
	})
	if err != nil {
		return nil, err
	}
	return s.Store.ThirdPartyByID(ctx, in.ID)
}

func (s *Service) UpdateThirdParty(ctx context.Context, a Actor, id string, in domain.ThirdParty, active bool) (*domain.ThirdParty, error) {
	cur, err := s.Store.ThirdPartyByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	in.ID, in.Active, in.Kind = id, active, cur.Kind // la nature (client/fournisseur) est figée
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.UpdateThirdParty(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.third_party.update", "finance_third_party", id, map[string]any{"name": cur.DisplayName, "active": cur.Active}, map[string]any{"name": in.DisplayName, "active": in.Active})
	})
	if err != nil {
		return nil, err
	}
	return s.Store.ThirdPartyByID(ctx, id)
}

func (s *Service) ListThirdParties(ctx context.Context, query string, kind *string, page, size int) (*Page[*domain.ThirdParty], error) {
	page, size = paging(page, size)
	items, total, err := s.Store.ListThirdParties(ctx, strings.TrimSpace(query), kind, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

// ── catégories ──────────────────────────────────────────────────────────────

func (s *Service) checkCategoryAccount(ctx context.Context, c *domain.OpCategory) error {
	acc, err := s.Store.AccountByID(ctx, c.AccountID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return invalid("/accountId", "compte introuvable")
		}
		return fail(err)
	}
	if !acc.Active {
		return invalid("/accountId", "le compte est inactif")
	}
	return nil
}

func (s *Service) CreateOpCategory(ctx context.Context, a Actor, in domain.OpCategory) (*domain.OpCategory, error) {
	in.ID, in.Active = s.NewID(), true
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := s.checkCategoryAccount(ctx, &in); err != nil {
		return nil, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateOpCategory(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.category.create", "finance_category", in.ID, nil, map[string]any{"kind": in.Kind, "label": in.Label, "accountId": in.AccountID})
	})
	if err != nil {
		return nil, err
	}
	return s.Store.OpCategoryByID(ctx, in.ID)
}

// UpdateOpCategory : le type (dépense/revenu) est figé ; le compte peut être corrigé (il n'affecte que les NOUVELLES écritures).
func (s *Service) UpdateOpCategory(ctx context.Context, a Actor, id string, in domain.OpCategory, active bool) (*domain.OpCategory, error) {
	cur, err := s.Store.OpCategoryByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	in.ID, in.Kind, in.Active = id, cur.Kind, active
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if in.AccountID != cur.AccountID {
		if err := s.checkCategoryAccount(ctx, &in); err != nil {
			return nil, err
		}
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.UpdateOpCategory(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.category.update", "finance_category", id, map[string]any{"label": cur.Label, "accountId": cur.AccountID, "active": cur.Active}, map[string]any{"label": in.Label, "accountId": in.AccountID, "active": in.Active})
	})
	if err != nil {
		return nil, err
	}
	return s.Store.OpCategoryByID(ctx, id)
}

func (s *Service) ListOpCategories(ctx context.Context, kind *string, active *bool) ([]*domain.OpCategory, error) {
	v, err := s.Store.ListOpCategories(ctx, kind, active)
	return v, fail(err)
}

// ── opérations ──────────────────────────────────────────────────────────────

func opSnap(o *domain.Operation) map[string]any {
	return map[string]any{"kind": o.Kind, "date": o.OperationDate.Format(domain.DateLayout), "label": o.Label, "reference": strOrNil(o.Reference), "net": o.AmountNet, "tax": o.TaxAmount,
		"category": o.CategoryID, "counterparty": o.CounterpartyAccountID, "mode": o.SettlementMode, "thirdParty": strOrNil(o.ThirdPartyID)}
}

// prepareOperation valide la forme puis vérifie catégorie, comptes et tiers.
func (s *Service) prepareOperation(ctx context.Context, o *domain.Operation) error {
	if err := o.Validate(); err != nil {
		return err
	}
	st, err := s.Store.GetSettings(ctx)
	if err != nil {
		return fail(err)
	}
	o.Currency = st.BaseCurrency
	cat, err := s.Store.OpCategoryByID(ctx, o.CategoryID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return invalid("/categoryId", "catégorie introuvable")
		}
		return fail(err)
	}
	if !cat.Active {
		return invalid("/categoryId", "la catégorie est inactive")
	}
	if cat.Kind != o.Kind {
		return invalid("/categoryId", "cette catégorie ne correspond pas au type de l'opération")
	}
	o.CategoryAccountID = cat.AccountID
	if o.Nature == nil && o.Kind == domain.KindExpense {
		o.Nature = cat.Nature
	}
	ids := []string{o.CounterpartyAccountID}
	if o.TaxAccountID != nil {
		ids = append(ids, *o.TaxAccountID)
	}
	accs, err := s.Store.AccountsByIDs(ctx, ids)
	if err != nil {
		return fail(err)
	}
	if a, ok := accs[o.CounterpartyAccountID]; !ok || !a.Active {
		return invalid("/counterpartyAccountId", "compte de contrepartie introuvable ou inactif")
	}
	if o.TaxAccountID != nil {
		if a, ok := accs[*o.TaxAccountID]; !ok || !a.Active {
			return invalid("/taxAccountId", "compte de taxe introuvable ou inactif")
		}
	}
	if o.ThirdPartyID != nil {
		tp, err := s.Store.ThirdPartyByID(ctx, *o.ThirdPartyID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return invalid("/thirdPartyId", "tiers introuvable")
			}
			return fail(err)
		}
		if !tp.Active {
			return invalid("/thirdPartyId", "le tiers est inactif")
		}
		want := "supplier"
		if o.Kind == domain.KindIncome {
			want = "customer"
		}
		if tp.Kind != want {
			return invalid("/thirdPartyId", "ce tiers n'est pas du bon type (fournisseur pour une dépense, client pour un revenu)")
		}
	}
	return nil
}

// checkDuplicate : même tiers, même référence de pièce, mêmes montants. Un soupçon n'interdit rien : il demande une confirmation explicite, tracée.
func (s *Service) checkDuplicate(ctx context.Context, o *domain.Operation, confirmed bool) error {
	if o.Reference == nil || o.ThirdPartyID == nil {
		return nil
	}
	dup, err := s.Store.DuplicateOperationExists(ctx, o)
	if err != nil {
		return fail(err)
	}
	if dup && !confirmed {
		return apperr.New(codes.AlreadyExists, apperr.CodeConflict, "doublon probable : une opération du même tiers, de même référence et de mêmes montants existe déjà. Si ce n'est pas un doublon, confirmez-le explicitement.").
			WithFields(apperr.Field{Pointer: "/duplicateConfirmed", Code: "DUPLICATE_SUSPECTED", Message: "doublon probable : confirmation requise"})
	}
	o.DuplicateConfirmed = dup && confirmed
	return nil
}

func (s *Service) CreateOperation(ctx context.Context, a Actor, in domain.Operation, confirmed bool) (*domain.Operation, error) {
	if a.ID == "" {
		return nil, apperr.Unauthenticated(apperr.CodeUnauthenticated, "auteur inconnu")
	}
	in.ID, in.CreatedBy, in.CreatedAt, in.EntryID, in.RecurringTemplateID = s.NewID(), a.ID, s.Now(), nil, nil
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.prepareOperation(ctx, &in); err != nil {
			return err
		}
		if err := s.checkDuplicate(ctx, &in, confirmed); err != nil {
			return err
		}
		if err := s.Store.CreateOperation(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.operation.create", "finance_operation", in.ID, nil, opSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.GetOperation(ctx, in.ID)
}

func (s *Service) UpdateOperation(ctx context.Context, a Actor, id string, in domain.Operation, confirmed bool) (*domain.Operation, error) {
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.Store.OperationByID(ctx, id, true)
		if err != nil {
			return err
		}
		if cur.EntryID != nil {
			return ErrFrozenOperation
		}
		in.ID, in.CreatedBy, in.CreatedAt, in.RecurringTemplateID = id, cur.CreatedBy, cur.CreatedAt, cur.RecurringTemplateID
		if err := s.prepareOperation(ctx, &in); err != nil {
			return err
		}
		if err := s.checkDuplicate(ctx, &in, confirmed); err != nil {
			return err
		}
		if err := s.Store.UpdateOperation(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.operation.update", "finance_operation", id, opSnap(cur), opSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.GetOperation(ctx, id)
}

func (s *Service) GetOperation(ctx context.Context, id string) (*domain.Operation, error) {
	o, err := s.Store.OperationByID(ctx, id, false)
	return o, fail(err)
}

func (s *Service) ListOperations(ctx context.Context, f OperationFilter, page, size int) (*Page[*domain.Operation], OpTotals, string, error) {
	page, size = paging(page, size)
	f.Query = strings.TrimSpace(f.Query)
	items, tot, total, err := s.Store.ListOperations(ctx, f, size, (page-1)*size)
	if err != nil {
		return nil, OpTotals{}, "", fail(err)
	}
	st, err := s.Store.GetSettings(ctx)
	if err != nil {
		return nil, OpTotals{}, "", fail(err)
	}
	return pageOf(items, total, page, size), tot, st.BaseCurrency, nil
}

// DeleteOperation : seule une opération sans écriture se supprime.
func (s *Service) DeleteOperation(ctx context.Context, a Actor, id string) error {
	return s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.Store.OperationByID(ctx, id, true)
		if err != nil {
			return err
		}
		if cur.EntryID != nil {
			return ErrFrozenOperation
		}
		if err := s.Store.DeleteOperation(ctx, id); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.operation.delete", "finance_operation", id, opSnap(cur), nil)
	})
}

// GenerateOperationEntry crée l'écriture de constatation en BROUILLON. Une opération n'en produit qu'une (verrou de ligne + clé d'unicité de source).
func (s *Service) GenerateOperationEntry(ctx context.Context, a Actor, id, journalID string) (*domain.Operation, error) {
	err := s.tx(ctx, func(ctx context.Context) error {
		o, err := s.Store.OperationByID(ctx, id, true)
		if err != nil {
			return err
		}
		if o.EntryID != nil {
			return conflict("l'écriture de cette opération existe déjà")
		}
		e := domain.Entry{JournalID: journalID, AccountingDate: o.OperationDate, Label: o.Label, Reference: o.Reference, Origin: domain.OriginOperation,
			Source: &domain.EntrySource{Type: "operation", ID: o.ID, Kind: "recognition"}, Lines: o.EntryLines(o.CategoryAccountID)}
		d, err := s.createDraft(ctx, a, e)
		if err != nil {
			return err
		}
		if err := s.Store.SetOperationEntry(ctx, id, &d.ID); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.operation.entry", "finance_operation", id, nil, map[string]any{"entry": d.ID, "gross": o.Gross()})
	})
	if err != nil {
		return nil, err
	}
	return s.GetOperation(ctx, id)
}

// CancelOperationEntry supprime l'écriture si elle est encore en brouillon ; une écriture validée ne se retire pas (contre-passation).
func (s *Service) CancelOperationEntry(ctx context.Context, a Actor, id string) (*domain.Operation, error) {
	err := s.tx(ctx, func(ctx context.Context) error {
		o, err := s.Store.OperationByID(ctx, id, true)
		if err != nil {
			return err
		}
		if o.EntryID == nil {
			return precondition("cette opération n'a pas d'écriture")
		}
		if o.Status != domain.OpEntryDraft {
			return precondition("l'écriture est validée : contre-passez-la depuis les écritures")
		}
		if len(o.Payments) > 0 {
			return precondition("des règlements sont enregistrés : supprimez-les d'abord")
		}
		if err := s.Store.DeleteEntry(ctx, *o.EntryID); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.operation.entry_cancel", "finance_operation", id, map[string]any{"entry": *o.EntryID}, nil)
	})
	if err != nil {
		return nil, err
	}
	return s.GetOperation(ctx, id)
}

// RecordOperationPayment enregistre un règlement (écriture en brouillon) d'une opération « en compte » dont l'écriture est VALIDÉE.
func (s *Service) RecordOperationPayment(ctx context.Context, a Actor, id string, date time.Time, amount int64, accountID, journalID string) (*domain.Operation, error) {
	if amount <= 0 {
		return nil, invalid("/amount", "le montant doit être strictement positif")
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		o, err := s.Store.OperationByID(ctx, id, true)
		if err != nil {
			return err
		}
		if o.SettlementMode != domain.SettleOnAccount {
			return precondition("cette opération est réglée immédiatement : aucun règlement à enregistrer")
		}
		if o.Status != domain.OpRecorded {
			return precondition("l'écriture de l'opération doit d'abord être validée")
		}
		if amount > o.Remaining() {
			return invalid("/amount", "le montant dépasse le reste à régler")
		}
		acc, err := s.Store.AccountByID(ctx, accountID)
		if err != nil || !acc.Active {
			return invalid("/accountId", "compte de règlement introuvable ou inactif")
		}
		payID := s.NewID()
		e := domain.Entry{JournalID: journalID, AccountingDate: date, Label: "Règlement : " + o.Label, Reference: o.Reference, Origin: domain.OriginOperation,
			Source: &domain.EntrySource{Type: "operation_payment", ID: payID, Kind: "settlement"}, Lines: o.PaymentLines(accountID, amount)}
		d, err := s.createDraft(ctx, a, e)
		if err != nil {
			return err
		}
		if err := s.Store.AddOperationPayment(ctx, id, domain.OperationPayment{ID: payID, PaymentDate: date, Amount: amount, AccountID: accountID, EntryID: d.ID}, a.ID); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.operation.payment", "finance_operation", id, nil, map[string]any{"amount": amount, "entry": d.ID, "date": date.Format(domain.DateLayout)})
	})
	if err != nil {
		return nil, err
	}
	return s.GetOperation(ctx, id)
}

// ── récurrences ─────────────────────────────────────────────────────────────

func (s *Service) CreateRecurring(ctx context.Context, a Actor, in domain.Recurring) (*domain.Recurring, error) {
	in.ID, in.CreatedBy, in.Active, in.LastGeneratedMonth = s.NewID(), a.ID, true, nil
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.checkRecurringRefs(ctx, &in); err != nil {
			return err
		}
		if err := s.Store.CreateRecurring(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.recurring.create", "finance_recurring", in.ID, nil, map[string]any{"label": in.Label, "net": in.AmountNet, "day": in.DayOfMonth})
	})
	if err != nil {
		return nil, err
	}
	return s.Store.RecurringByID(ctx, in.ID)
}

func (s *Service) checkRecurringRefs(ctx context.Context, r *domain.Recurring) error {
	probe := domain.Operation{Kind: r.Kind, OperationDate: r.Start, Label: r.Label, AmountNet: r.AmountNet, TaxAmount: r.TaxAmount, TaxAccountID: r.TaxAccountID, CategoryID: r.CategoryID,
		CounterpartyAccountID: r.CounterpartyAccountID, SettlementMode: r.SettlementMode, Nature: r.Nature, ThirdPartyID: r.ThirdPartyID}
	if err := s.prepareOperation(ctx, &probe); err != nil {
		return err
	}
	if j, err := s.Store.JournalByID(ctx, r.JournalID); err != nil || !j.Active {
		return invalid("/journalId", "journal introuvable ou inactif")
	}
	return nil
}

func (s *Service) UpdateRecurring(ctx context.Context, a Actor, id string, in domain.Recurring, active bool) (*domain.Recurring, error) {
	cur, err := s.Store.RecurringByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	in.ID, in.Kind, in.CreatedBy, in.Active, in.LastGeneratedMonth, in.Start = id, cur.Kind, cur.CreatedBy, active, cur.LastGeneratedMonth, cur.Start
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		if err := s.checkRecurringRefs(ctx, &in); err != nil {
			return err
		}
		if err := s.Store.UpdateRecurring(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.recurring.update", "finance_recurring", id, map[string]any{"net": cur.AmountNet, "active": cur.Active}, map[string]any{"net": in.AmountNet, "active": in.Active})
	})
	if err != nil {
		return nil, err
	}
	return s.Store.RecurringByID(ctx, id)
}

func (s *Service) ListRecurring(ctx context.Context) ([]*domain.Recurring, error) {
	v, err := s.Store.ListRecurring(ctx)
	return v, fail(err)
}

// GenerateRecurring crée les opérations (brouillons, sans écriture) des mois échus. Idempotent : un mois déjà généré est compté « ignoré ».
func (s *Service) GenerateRecurring(ctx context.Context, a Actor, upTo time.Time) (created, skipped int, err error) {
	tpls, err := s.Store.ListRecurring(ctx)
	if err != nil {
		return 0, 0, fail(err)
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		for _, t := range tpls {
			if !t.Active {
				continue
			}
			var last *time.Time
			for _, m := range t.DueMonths(upTo) {
				o := domain.Operation{ID: s.NewID(), Kind: t.Kind, OperationDate: time.Date(m.Year(), m.Month(), t.DayOfMonth, 0, 0, 0, 0, time.UTC), ThirdPartyID: t.ThirdPartyID, CategoryID: t.CategoryID,
					Label: t.Label, AmountNet: t.AmountNet, TaxAmount: t.TaxAmount, TaxAccountID: t.TaxAccountID, CounterpartyAccountID: t.CounterpartyAccountID, SettlementMode: t.SettlementMode,
					Nature: t.Nature, CreatedBy: a.ID, CreatedAt: s.Now(), RecurringTemplateID: &t.ID}
				if err := s.prepareOperation(ctx, &o); err != nil {
					return err
				}
				ok, err := s.Store.CreateOperationIfAbsent(ctx, &o, t.ID, m)
				if err != nil {
					return err
				}
				if ok {
					created++
				} else {
					skipped++
				}
				mm := m
				last = &mm
			}
			if last != nil {
				if err := s.Store.SetRecurringLastMonth(ctx, t.ID, *last); err != nil {
					return err
				}
			}
		}
		if created+skipped == 0 {
			return nil
		}
		return s.audit(ctx, a, "finance.recurring.generate", "finance_recurring", "generate", nil, map[string]any{"upTo": upTo.Format(domain.DateLayout), "created": created, "skipped": skipped})
	})
	return created, skipped, err
}

// ── pièces justificatives ───────────────────────────────────────────────────

func (s *Service) UploadAttachment(ctx context.Context, a Actor, opID, filename string, data []byte) (*domain.Attachment, error) {
	if len(data) == 0 {
		return nil, invalid("/data", "fichier vide")
	}
	if len(data) > domain.MaxAttachmentSize {
		return nil, invalid("/data", "fichier trop volumineux (5 Mo au maximum)")
	}
	ct := domain.DetectContentType(data)
	if ct == "" {
		return nil, invalid("/data", "format non accepté : PDF, PNG ou JPEG uniquement")
	}
	name := domain.CleanFilename(filename)
	if name == "" {
		return nil, invalid("/filename", "nom de fichier obligatoire")
	}
	sum := sha256.Sum256(data)
	att := domain.Attachment{ID: s.NewID(), OperationID: opID, Filename: name, ContentType: ct, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data)), CreatedAt: s.Now()}
	err := s.tx(ctx, func(ctx context.Context) error {
		if _, err := s.Store.OperationByID(ctx, opID, false); err != nil {
			return err
		}
		if err := s.Store.AddAttachment(ctx, &att, data, a.ID); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.attachment.add", "finance_operation", opID, nil, map[string]any{"file": name, "size": att.Size, "sha256": att.SHA256})
	})
	if err != nil {
		return nil, err
	}
	return &att, nil
}

func (s *Service) ListAttachments(ctx context.Context, opID string) ([]*domain.Attachment, error) {
	if _, err := s.Store.OperationByID(ctx, opID, false); err != nil {
		return nil, fail(err)
	}
	v, err := s.Store.ListAttachments(ctx, opID)
	return v, fail(err)
}

func (s *Service) GetAttachment(ctx context.Context, id string) (*domain.Attachment, []byte, error) {
	a, d, err := s.Store.AttachmentByID(ctx, id)
	return a, d, fail(err)
}

// DeleteAttachment : refusée dès que l'écriture de l'opération est validée (pièce justificative conservée).
func (s *Service) DeleteAttachment(ctx context.Context, a Actor, id string) error {
	return s.tx(ctx, func(ctx context.Context) error {
		att, _, err := s.Store.AttachmentByID(ctx, id)
		if err != nil {
			return err
		}
		o, err := s.Store.OperationByID(ctx, att.OperationID, true)
		if err != nil {
			return err
		}
		if o.Status == domain.OpRecorded || o.Status == domain.OpReversed {
			return precondition("l'écriture est validée : la pièce justificative est conservée")
		}
		if err := s.Store.DeleteAttachment(ctx, id); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.attachment.delete", "finance_operation", o.ID, map[string]any{"file": att.Filename, "sha256": att.SHA256}, nil)
	})
}
