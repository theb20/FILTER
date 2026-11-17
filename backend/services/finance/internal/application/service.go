package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/events"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

type Config struct {
	Store  Store
	Box    Sealer           // chiffrement des numéros de compte bancaire ; nil = numéros refusés
	Events events.Publisher // audit : `audit.recorded.v1` écrit dans la MÊME transaction que l'action (outbox)
	Now    func() time.Time
	NewID  func() string
}

type Service struct{ Config }

func New(c Config) *Service {
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.NewID == nil {
		c.NewID = func() string { return uuid.Must(uuid.NewV7()).String() }
	}
	return &Service{c}
}

// Actor : qui agit (transmis par le gateway, identifiant tiré d'un jeton déjà vérifié). Vide = système.
type Actor struct{ ID, IP, TraceID string }

type Page[T any] struct {
	Items                        []T
	Page, PageSize, Total, Pages int
}

func paging(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}

func pageOf[T any](items []T, total, page, size int) *Page[T] {
	return &Page[T]{Items: items, Page: page, PageSize: size, Total: total, Pages: (total + size - 1) / size}
}

func invalid(ptr, msg string) error {
	return apperr.Validation(msg, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
}
func conflict(msg string) error { return apperr.New(codes.AlreadyExists, apperr.CodeConflict, msg) }
func precondition(msg string) error {
	return apperr.New(codes.FailedPrecondition, apperr.CodeConflict, msg)
}

// selfAction : la double validation s'applique à tous, super-administrateur compris.
func selfAction(msg string) error {
	return apperr.PermissionDenied(apperr.CodeSelfActionForbidden, msg)
}

// fail traduit les erreurs du store en erreurs de l'API.
func fail(err error) error {
	var ae *apperr.Error
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ae):
		return err
	case errors.Is(err, ErrNotFound):
		return apperr.NotFound("élément introuvable")
	case errors.Is(err, ErrCodeTaken):
		return conflict("ce code est déjà utilisé")
	case errors.Is(err, ErrLabelTaken):
		return conflict("ce libellé est déjà utilisé")
	case errors.Is(err, ErrOverlap):
		return conflict("cet exercice chevauche un exercice existant")
	case errors.Is(err, ErrNoPeriod):
		return invalid("/accountingDate", "aucun exercice ne couvre cette date : ouvrez l'exercice correspondant")
	case errors.Is(err, ErrPeriodClosed):
		return precondition("la période comptable est clôturée")
	case errors.Is(err, ErrHasDrafts):
		return precondition("des écritures en brouillon sont datées dans cette période : validez-les ou supprimez-les avant de clôturer")
	case errors.Is(err, ErrNotApproved):
		return precondition("la réouverture n'a pas été approuvée par une autre personne")
	case errors.Is(err, ErrUnbalanced):
		return invalid("/lines", "l'écriture n'est pas équilibrée")
	case errors.Is(err, ErrFourEyes):
		return selfAction("une autre personne doit valider cette écriture")
	case errors.Is(err, ErrImmutable):
		return precondition("une écriture validée ne peut plus être modifiée ni supprimée : contre-passez-la")
	case errors.Is(err, ErrAccountInactive):
		return invalid("/lines", "un compte de l'écriture est inactif")
	case errors.Is(err, ErrJournalInactive):
		return invalid("/journalId", "le journal est inactif")
	case errors.Is(err, ErrDuplicateSource):
		return conflict("cette opération a déjà été comptabilisée")
	case errors.Is(err, ErrAlreadyReversed):
		return conflict("cette écriture a déjà été contre-passée")
	case errors.Is(err, ErrPendingExists):
		return conflict("une demande est déjà en attente")
	case errors.Is(err, ErrInUse):
		return precondition("élément utilisé : action impossible")
	case errors.Is(err, ErrFrozenOperation):
		return precondition("cette opération est liée à une écriture : elle ne peut plus être modifiée")
	case errors.Is(err, ErrNameTaken):
		return conflict("ce nom est déjà utilisé")
	case errors.Is(err, ErrRecurringTaken):
		return conflict("cette occurrence a déjà été générée")
	case errors.Is(err, ErrDuplicateFile):
		return conflict("cette pièce est déjà jointe à l'opération")
	case errors.Is(err, ErrFileImported):
		return conflict("ce fichier a déjà été importé pour ce compte")
	case errors.Is(err, ErrReconOver):
		return invalid("/amount", "le montant dépasse ce qui reste à rapprocher d'un des deux côtés")
	case errors.Is(err, ErrReconDirection):
		return invalid("/entryLineId", "sens incompatible : une entrée du relevé se rapproche d'un débit du compte banque, une sortie d'un crédit")
	case errors.Is(err, ErrReconAccount):
		return invalid("/entryLineId", "cette ligne d'écriture n'appartient pas au compte comptable de ce compte bancaire")
	case errors.Is(err, ErrReconNotPosted):
		return precondition("seules les écritures validées se rapprochent")
	case errors.Is(err, ErrBankLinked):
		return conflict("ce compte comptable est déjà lié à un compte bancaire")
	case errors.Is(err, ErrAccountMissing):
		return invalid("/lines", "un compte de l'écriture est introuvable")
	case errors.Is(err, ErrJournalMissing):
		return invalid("/journalId", "journal introuvable")
	}
	return apperr.Internal(err)
}

func (s *Service) tx(ctx context.Context, f func(ctx context.Context) error) error {
	return fail(s.Store.Do(ctx, f))
}

// ── audit ───────────────────────────────────────────────────────────────────

func strOrNil(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

// audit publie `audit.recorded.v1` dans la transaction en cours ; identity l'inscrit dans son journal (append-only).
func (s *Service) audit(ctx context.Context, a Actor, action, entity, entityID string, before, after map[string]any) error {
	if s.Events == nil {
		return nil
	}
	b, err1 := structpb.NewStruct(before)
	af, err2 := structpb.NewStruct(after)
	if err1 != nil || err2 != nil {
		return apperr.Internal(errors.Join(err1, err2))
	}
	actorType := "staff"
	if a.ID == "" {
		actorType = "system"
	}
	msg := &eventsv1.AuditRecorded{ActorType: actorType, ActorId: a.ID, Action: action, Entity: entity, EntityId: entityID, Before: b, After: af, Ip: a.IP}
	if before == nil {
		msg.Before = nil
	}
	if after == nil {
		msg.After = nil
	}
	if err := s.Events.Publish(ctx, events.Event{Type: "audit.recorded.v1", AggregateID: entityID, ActorID: a.ID, ActorType: actorType, TraceID: a.TraceID, Payload: msg}); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func accountSnap(a *domain.Account) map[string]any {
	return map[string]any{"code": a.Code, "label": a.Label, "kind": a.Kind, "active": a.Active}
}
func journalSnap(j *domain.Journal) map[string]any {
	return map[string]any{"code": j.Code, "label": j.Label, "kind": j.Kind, "active": j.Active}
}
func entrySnap(e *domain.Entry) map[string]any {
	lines := make([]any, 0, len(e.Lines))
	for _, l := range e.Lines {
		lines = append(lines, map[string]any{"account": l.AccountCode, "debit": l.Debit, "credit": l.Credit, "memo": strOrNil(l.Memo)})
	}
	return map[string]any{"journal": e.JournalCode, "date": e.AccountingDate.Format(domain.DateLayout), "label": e.Label, "reference": strOrNil(e.Reference), "status": e.Status,
		"totalDebit": e.TotalDebit, "totalCredit": e.TotalCredit, "lines": lines}
}

// ── réglages ────────────────────────────────────────────────────────────────

func (s *Service) GetSettings(ctx context.Context) (*domain.Settings, error) {
	v, err := s.Store.GetSettings(ctx)
	return v, fail(err)
}

// PutSettings : la devise de base ne change plus dès qu'une écriture existe ; le seuil de double validation est audité.
func (s *Service) PutSettings(ctx context.Context, a Actor, in domain.Settings) (*domain.Settings, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.Store.GetSettings(ctx)
		if err != nil {
			return err
		}
		if in.BaseCurrency != cur.BaseCurrency {
			if used, err := s.Store.AnyEntryExists(ctx); err != nil {
				return err
			} else if used {
				return precondition("la devise de base ne peut plus changer : des écritures existent")
			}
		}
		if err := s.Store.PutSettings(ctx, &in, a.ID); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.settings.update", "finance_settings", "settings",
			map[string]any{"baseCurrency": cur.BaseCurrency, "fourEyesMinAmount": cur.FourEyesMinAmount},
			map[string]any{"baseCurrency": in.BaseCurrency, "fourEyesMinAmount": in.FourEyesMinAmount})
	})
	if err != nil {
		return nil, err
	}
	return s.GetSettings(ctx)
}

// ── plan comptable ──────────────────────────────────────────────────────────

func (s *Service) CreateAccount(ctx context.Context, a Actor, in domain.Account) (*domain.Account, error) {
	in.ID, in.Active = s.NewID(), true
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateAccount(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.account.create", "finance_account", in.ID, nil, accountSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.GetAccount(ctx, in.ID)
}

func (s *Service) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	v, err := s.Store.AccountByID(ctx, id)
	return v, fail(err)
}

// UpdateAccount : le code est figé ; la nature ne change plus dès que le compte porte des écritures (les états en dépendent).
func (s *Service) UpdateAccount(ctx context.Context, a Actor, id string, in domain.Account, active bool) (*domain.Account, error) {
	cur, err := s.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	in.ID, in.Active = id, active
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if !strings.EqualFold(in.Code, cur.Code) {
		return nil, invalid("/code", "le code d'un compte ne peut pas changer")
	}
	in.Code = cur.Code
	if in.Kind != cur.Kind {
		if used, err := s.Store.AccountUsed(ctx, id); err != nil {
			return nil, fail(err)
		} else if used {
			return nil, invalid("/kind", "la nature ne peut plus changer : le compte porte des écritures")
		}
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.UpdateAccount(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.account.update", "finance_account", id, accountSnap(cur), accountSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.GetAccount(ctx, id)
}

func (s *Service) ListAccounts(ctx context.Context, f AccountFilter, page, size int) (*Page[*domain.Account], error) {
	page, size = paging(page, size)
	f.Query = strings.TrimSpace(f.Query)
	items, total, err := s.Store.ListAccounts(ctx, f, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

type ImportError struct {
	Line    int
	Message string
}

type ImportResult struct {
	Simulated               bool
	Total, Created, Skipped int
	Errors                  []ImportError
}

const maxImportLines = 5000

// splitCSVLine : séparateur « ; » (Excel en français) ou « , » ; les guillemets simples autour d'un champ sont retirés.
func splitCSVLine(l string) []string {
	sep := ","
	if strings.Contains(l, ";") {
		sep = ";"
	} else if strings.Contains(l, "\t") {
		sep = "\t"
	}
	parts := strings.Split(l, sep)
	for i := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(parts[i]), `"`)
	}
	return parts
}

// ImportAccounts : « code;libellé;nature ». Chaque ligne est jugée séparément ; un code existant est ignoré, jamais écrasé.
func (s *Service) ImportAccounts(ctx context.Context, a Actor, csv string, simulate bool) (*ImportResult, error) {
	lines := strings.Split(strings.ReplaceAll(csv, "\r\n", "\n"), "\n")
	if len(lines) > maxImportLines+1 {
		return nil, invalid("/csv", "5000 lignes au maximum")
	}
	res := &ImportResult{Simulated: simulate}
	type row struct {
		line int
		acc  domain.Account
	}
	var rows []row
	seen := map[string]bool{}
	for i, raw := range lines {
		raw = strings.TrimPrefix(strings.TrimSpace(raw), "\ufeff")
		if raw == "" {
			continue
		}
		f := splitCSVLine(raw)
		if i == 0 && len(f) >= 1 && strings.EqualFold(f[0], "code") {
			continue // en-tête
		}
		res.Total++
		if len(f) < 3 {
			res.Errors = append(res.Errors, ImportError{i + 1, "trois colonnes attendues : code ; libellé ; nature"})
			continue
		}
		acc := domain.Account{Code: f[0], Label: f[1], Kind: strings.ToLower(f[2]), Active: true}
		if err := acc.Validate(); err != nil {
			msg := "ligne invalide"
			var ae *apperr.Error
			if errors.As(err, &ae) && len(ae.Fields) > 0 {
				msg = ae.Fields[0].Message
			}
			res.Errors = append(res.Errors, ImportError{i + 1, msg})
			continue
		}
		key := strings.ToLower(acc.Code)
		if seen[key] {
			res.Errors = append(res.Errors, ImportError{i + 1, "code en double dans le fichier"})
			continue
		}
		seen[key] = true
		rows = append(rows, row{i + 1, acc})
	}
	codes := make([]string, len(rows))
	for i, r := range rows {
		codes[i] = strings.ToLower(r.acc.Code)
	}
	exist, err := s.Store.ExistingAccountCodes(ctx, codes)
	if err != nil {
		return nil, fail(err)
	}
	var toCreate []domain.Account
	for _, r := range rows {
		if exist[strings.ToLower(r.acc.Code)] {
			res.Skipped++
			continue
		}
		r.acc.ID = s.NewID()
		toCreate = append(toCreate, r.acc)
	}
	res.Created = len(toCreate)
	if simulate || len(toCreate) == 0 {
		return res, nil
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		for i := range toCreate {
			if err := s.Store.CreateAccount(ctx, &toCreate[i]); err != nil {
				return err
			}
		}
		return s.audit(ctx, a, "finance.account.import", "finance_account", "import", nil, map[string]any{"created": len(toCreate), "skipped": res.Skipped, "errors": len(res.Errors)})
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ── journaux ────────────────────────────────────────────────────────────────

func (s *Service) CreateJournal(ctx context.Context, a Actor, in domain.Journal) (*domain.Journal, error) {
	in.ID, in.Active = s.NewID(), true
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateJournal(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.journal.create", "finance_journal", in.ID, nil, journalSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.Store.JournalByID(ctx, in.ID)
}

func (s *Service) UpdateJournal(ctx context.Context, a Actor, id string, in domain.Journal, active bool) (*domain.Journal, error) {
	cur, err := s.Store.JournalByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	in.ID, in.Active = id, active
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if in.Code != cur.Code {
		return nil, invalid("/code", "le code d'un journal ne peut pas changer (il figure dans les numéros d'écriture)")
	}
	if in.Kind != cur.Kind {
		if used, err := s.Store.JournalUsed(ctx, id); err != nil {
			return nil, fail(err)
		} else if used {
			return nil, invalid("/kind", "la nature ne peut plus changer : le journal porte des écritures")
		}
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.UpdateJournal(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.journal.update", "finance_journal", id, journalSnap(cur), journalSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.Store.JournalByID(ctx, id)
}

func (s *Service) ListJournals(ctx context.Context, active *bool) ([]*domain.Journal, error) {
	v, err := s.Store.ListJournals(ctx, active)
	return v, fail(err)
}

// ── exercices et périodes ───────────────────────────────────────────────────

func (s *Service) CreateFiscalYear(ctx context.Context, a Actor, in domain.FiscalYear) (*domain.FiscalYear, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	in.ID = s.NewID()
	in.Periods = domain.BuildPeriods(in.Start, in.End)
	for i := range in.Periods {
		in.Periods[i].ID, in.Periods[i].FiscalYearID = s.NewID(), in.ID
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateFiscalYear(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.fiscal_year.create", "finance_fiscal_year", in.ID, nil,
			map[string]any{"label": in.Label, "start": in.Start.Format(domain.DateLayout), "end": in.End.Format(domain.DateLayout), "periods": len(in.Periods)})
	})
	if err != nil {
		return nil, err
	}
	return s.GetFiscalYear(ctx, in.ID)
}

func (s *Service) GetFiscalYear(ctx context.Context, id string) (*domain.FiscalYear, error) {
	v, err := s.Store.FiscalYearByID(ctx, id)
	return v, fail(err)
}

func (s *Service) ListFiscalYears(ctx context.Context) ([]*domain.FiscalYear, error) {
	v, err := s.Store.ListFiscalYears(ctx)
	return v, fail(err)
}

func periodLabel(p *domain.FiscalPeriod) string {
	return p.Start.Format(domain.DateLayout) + " → " + p.End.Format(domain.DateLayout)
}

// ClosePeriod : refusée s'il reste des brouillons (contrôle aussi en base).
func (s *Service) ClosePeriod(ctx context.Context, a Actor, id string) (*domain.FiscalPeriod, error) {
	err := s.tx(ctx, func(ctx context.Context) error {
		p, err := s.Store.PeriodByID(ctx, id, true)
		if err != nil {
			return err
		}
		if p.Status == domain.PeriodClosed {
			return precondition("la période est déjà clôturée")
		}
		if n, err := s.Store.CountDrafts(ctx, id); err != nil {
			return err
		} else if n > 0 {
			return precondition("il reste " + itoa(n) + " écriture(s) en brouillon datée(s) dans cette période : validez-les ou supprimez-les avant de clôturer")
		}
		if err := s.Store.ClosePeriod(ctx, id, a.ID, s.Now()); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.period.close", "finance_period", id, map[string]any{"status": "open", "period": periodLabel(p)}, map[string]any{"status": "closed"})
	})
	if err != nil {
		return nil, err
	}
	p, err := s.Store.PeriodByID(ctx, id, false)
	return p, fail(err)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// RequestPeriodReopen : la réouverture est une opération sensible, approuvée par une AUTRE personne.
func (s *Service) RequestPeriodReopen(ctx context.Context, a Actor, id, reason string) (*domain.Approval, error) {
	reason, err := domain.ValidateReason(reason, "/reason")
	if err != nil {
		return nil, err
	}
	ap := domain.Approval{ID: s.NewID(), Type: domain.ApprovalReopenPeriod, SubjectID: id, Reason: reason, Status: domain.ApprovalPending, RequestedBy: a.ID, RequestedAt: s.Now()}
	err = s.tx(ctx, func(ctx context.Context) error {
		p, err := s.Store.PeriodByID(ctx, id, true)
		if err != nil {
			return err
		}
		if p.Status != domain.PeriodClosed {
			return precondition("la période n'est pas clôturée")
		}
		ap.SubjectLabel = periodLabel(p)
		if err := s.Store.CreateApproval(ctx, &ap); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.approval.request", "finance_approval", ap.ID, nil, map[string]any{"type": ap.Type, "subject": ap.SubjectLabel, "reason": reason})
	})
	if err != nil {
		return nil, err
	}
	return s.Store.ApprovalByID(ctx, ap.ID, false)
}

func (s *Service) ListApprovals(ctx context.Context, status *string, page, size int) (*Page[*domain.Approval], error) {
	page, size = paging(page, size)
	items, total, err := s.Store.ListApprovals(ctx, status, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

// DecideApproval : l'approbateur est différent du demandeur, y compris pour le super-administrateur.
func (s *Service) DecideApproval(ctx context.Context, a Actor, id string, approve bool, note *string) (*domain.Approval, error) {
	if note != nil {
		n := strings.TrimSpace(*note)
		if len(n) > 500 {
			return nil, invalid("/note", "500 caractères maximum")
		}
		if n == "" {
			note = nil
		} else {
			note = &n
		}
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		ap, err := s.Store.ApprovalByID(ctx, id, true)
		if err != nil {
			return err
		}
		if ap.Status != domain.ApprovalPending {
			return precondition("cette demande a déjà été traitée")
		}
		status := domain.ApprovalRejected
		if approve {
			if ap.RequestedBy == a.ID {
				return selfAction("une autre personne doit approuver cette demande")
			}
			status = domain.ApprovalApproved
		}
		if err := s.Store.DecideApproval(ctx, id, status, a.ID, note, s.Now()); err != nil {
			return err
		}
		if approve && ap.Type == domain.ApprovalReopenPeriod {
			if err := s.Store.ReopenPeriod(ctx, ap.SubjectID); err != nil {
				return err
			}
			if err := s.audit(ctx, a, "finance.period.reopen", "finance_period", ap.SubjectID, map[string]any{"status": "closed"}, map[string]any{"status": "open", "approval": id, "reason": ap.Reason}); err != nil {
				return err
			}
		}
		return s.audit(ctx, a, "finance.approval."+status, "finance_approval", id, map[string]any{"status": ap.Status}, map[string]any{"status": status, "note": strOrNil(note)})
	})
	if err != nil {
		return nil, err
	}
	return s.Store.ApprovalByID(ctx, id, false)
}

// ── écritures ───────────────────────────────────────────────────────────────

// prepare valide la forme, rattache l'écriture à sa période (ouverte) et vérifie journal et comptes.
func (s *Service) prepare(ctx context.Context, e *domain.Entry) error {
	if err := e.ValidateDraft(); err != nil {
		return err
	}
	if err := e.Totals(); err != nil {
		return err
	}
	st, err := s.Store.GetSettings(ctx)
	if err != nil {
		return fail(err)
	}
	e.Currency = st.BaseCurrency
	p, err := s.Store.PeriodForDate(ctx, e.AccountingDate)
	if err != nil {
		return fail(err)
	}
	if p.Status != domain.PeriodOpen {
		return invalid("/accountingDate", "la période comptable de cette date est clôturée")
	}
	e.FiscalPeriodID, e.FiscalYearID = p.ID, p.FiscalYearID
	j, err := s.Store.JournalByID(ctx, e.JournalID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return invalid("/journalId", "journal introuvable")
		}
		return fail(err)
	}
	if !j.Active {
		return invalid("/journalId", "le journal est inactif")
	}
	e.JournalCode = j.Code
	ids := make([]string, len(e.Lines))
	for i, l := range e.Lines {
		ids[i] = l.AccountID
	}
	accs, err := s.Store.AccountsByIDs(ctx, ids)
	if err != nil {
		return fail(err)
	}
	for i := range e.Lines {
		acc, ok := accs[e.Lines[i].AccountID]
		if !ok {
			return invalid("/lines/"+itoa(i)+"/accountId", "compte introuvable")
		}
		if !acc.Active {
			return invalid("/lines/"+itoa(i)+"/accountId", "le compte « "+acc.Code+" » est inactif")
		}
		e.Lines[i].AccountCode, e.Lines[i].AccountLabel = acc.Code, acc.Label
		e.Lines[i].ID = s.NewID()
	}
	return nil
}

// CreateEntry : une saisie manuelle est toujours un BROUILLON, sans numéro.
func (s *Service) CreateEntry(ctx context.Context, a Actor, in domain.Entry) (*domain.Entry, error) {
	in.Origin, in.Source = domain.OriginManual, nil
	e, err := s.createDraft(ctx, a, in)
	if err != nil {
		return nil, err
	}
	return s.GetEntry(ctx, e.ID)
}

// createDraft enregistre un brouillon (manuel ou issu d'une opération : Origin et Source sont conservés tels que fournis). Jamais de validation ici.
func (s *Service) createDraft(ctx context.Context, a Actor, in domain.Entry) (*domain.Entry, error) {
	in.ID, in.Status, in.EntryNumber = s.NewID(), domain.StatusDraft, nil
	in.PostedBy, in.PostedAt = nil, nil
	in.CreatedBy, in.CreatedAt = a.ID, s.Now()
	if a.ID == "" {
		return nil, apperr.Unauthenticated(apperr.CodeUnauthenticated, "auteur inconnu")
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.prepare(ctx, &in); err != nil {
			return err
		}
		if err := s.Store.CreateEntry(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.entry.create", "finance_entry", in.ID, nil, entrySnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return &in, nil
}

func (s *Service) UpdateEntry(ctx context.Context, a Actor, id string, in domain.Entry) (*domain.Entry, error) {
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.Store.EntryByID(ctx, id, true)
		if err != nil {
			return err
		}
		if cur.Status != domain.StatusDraft {
			return ErrImmutable
		}
		before := entrySnap(cur)
		in.ID, in.Status, in.Origin, in.Source, in.CreatedBy, in.CreatedAt = id, cur.Status, cur.Origin, cur.Source, cur.CreatedBy, cur.CreatedAt
		in.ReversesEntryID, in.ReversalReason = cur.ReversesEntryID, cur.ReversalReason
		if err := s.prepare(ctx, &in); err != nil {
			return err
		}
		if err := s.Store.ReplaceEntry(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.entry.update", "finance_entry", id, before, entrySnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntry(ctx, id)
}

func (s *Service) DeleteEntry(ctx context.Context, a Actor, id string) error {
	return s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.Store.EntryByID(ctx, id, true)
		if err != nil {
			return err
		}
		if cur.Status != domain.StatusDraft {
			return ErrImmutable
		}
		if err := s.Store.DeleteEntry(ctx, id); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.entry.delete", "finance_entry", id, entrySnap(cur), nil)
	})
}

func (s *Service) GetEntry(ctx context.Context, id string) (*domain.Entry, error) {
	e, err := s.Store.EntryByID(ctx, id, false)
	return e, fail(err)
}

func (s *Service) ListEntries(ctx context.Context, f EntryFilter, page, size int) (*Page[*domain.Entry], error) {
	page, size = paging(page, size)
	f.Query = strings.TrimSpace(f.Query)
	items, total, err := s.Store.ListEntries(ctx, f, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

// PostEntry valide une écriture : équilibre, période ouverte, comptes et journal actifs, numéro attribué dans la même transaction.
// Le validateur doit différer de l'auteur dès que le total atteint le seuil (0 = toujours), y compris pour le super-administrateur.
func (s *Service) PostEntry(ctx context.Context, a Actor, id string) (*domain.Entry, error) {
	if a.ID == "" {
		return nil, apperr.Unauthenticated(apperr.CodeUnauthenticated, "validateur inconnu")
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		e, err := s.Store.EntryByID(ctx, id, true)
		if err != nil {
			return err
		}
		if e.Status != domain.StatusDraft {
			return precondition("cette écriture est déjà validée")
		}
		if err := e.ValidateForPosting(); err != nil {
			return err
		}
		st, err := s.Store.GetSettings(ctx)
		if err != nil {
			return err
		}
		if e.CreatedBy == a.ID && e.TotalDebit >= st.FourEyesMinAmount {
			return selfAction("une autre personne doit valider cette écriture (vous en êtes l'auteur)")
		}
		p, err := s.Store.PeriodByID(ctx, e.FiscalPeriodID, false)
		if err != nil {
			return err
		}
		if p.Status != domain.PeriodOpen {
			return ErrPeriodClosed
		}
		n, err := s.Store.PostEntry(ctx, e, a.ID, s.Now())
		if err != nil {
			return err
		}
		e.EntryNumber, e.Status = &n, domain.StatusPosted
		return s.audit(ctx, a, "finance.entry.post", "finance_entry", id, map[string]any{"status": "draft", "author": e.CreatedBy}, map[string]any{"status": "posted", "number": n, "total": e.TotalDebit})
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntry(ctx, id)
}

// ReverseEntry crée un BROUILLON de contre-passation (lignes inversées), à valider à son tour par une autre personne.
func (s *Service) ReverseEntry(ctx context.Context, a Actor, id, reason string, date *time.Time) (*domain.Entry, error) {
	reason, err := domain.ValidateReason(reason, "/reason")
	if err != nil {
		return nil, err
	}
	if a.ID == "" {
		return nil, apperr.Unauthenticated(apperr.CodeUnauthenticated, "auteur inconnu")
	}
	var newID string
	err = s.tx(ctx, func(ctx context.Context) error {
		orig, err := s.Store.EntryByID(ctx, id, true)
		if err != nil {
			return err
		}
		if orig.Status != domain.StatusPosted {
			return precondition("seule une écriture validée se contre-passe (un brouillon se modifie ou se supprime)")
		}
		if orig.Origin == domain.OriginReversal {
			return precondition("une contre-passation ne se contre-passe pas : saisissez une nouvelle écriture")
		}
		if orig.ReversedByEntryID != nil {
			return ErrAlreadyReversed
		}
		d := s.Now().Truncate(24 * time.Hour)
		if date != nil {
			d = *date
		}
		num := ""
		if n := orig.Number(); n != nil {
			num = *n
		}
		rev := domain.Entry{ID: s.NewID(), JournalID: orig.JournalID, AccountingDate: d, Status: domain.StatusDraft, Label: "Contre-passation de " + num + " : " + orig.Label,
			Reference: orig.Reference, Origin: domain.OriginReversal, ReversesEntryID: &id, ReversalReason: &reason, Lines: orig.ReversalLines(), CreatedBy: a.ID, CreatedAt: s.Now()}
		if len([]rune(rev.Label)) > 300 {
			rev.Label = string([]rune(rev.Label)[:300])
		}
		if err := s.prepare(ctx, &rev); err != nil {
			return err
		}
		if err := s.Store.CreateEntry(ctx, &rev); err != nil {
			return err
		}
		newID = rev.ID
		return s.audit(ctx, a, "finance.entry.reverse", "finance_entry", id, nil, map[string]any{"reversal": rev.ID, "reason": reason, "original": num})
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntry(ctx, newID)
}
