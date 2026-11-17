// Package application : cas d'usage du service finance. Ne dépend que des ports ci-dessous.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
)

var (
	ErrNotFound        = errors.New("introuvable")
	ErrCodeTaken       = errors.New("code déjà utilisé")
	ErrOverlap         = errors.New("exercice chevauchant")
	ErrLabelTaken      = errors.New("libellé déjà utilisé")
	ErrNoPeriod        = errors.New("aucune période pour cette date")
	ErrPeriodClosed    = errors.New("période clôturée")
	ErrHasDrafts       = errors.New("brouillons restants")
	ErrNotApproved     = errors.New("réouverture non approuvée")
	ErrUnbalanced      = errors.New("écriture déséquilibrée")
	ErrFourEyes        = errors.New("validation par une autre personne requise")
	ErrImmutable       = errors.New("écriture validée immuable")
	ErrAccountInactive = errors.New("compte inactif")
	ErrJournalInactive = errors.New("journal inactif")
	ErrDuplicateSource = errors.New("source déjà comptabilisée")
	ErrAlreadyReversed = errors.New("déjà contre-passée")
	ErrPendingExists   = errors.New("demande déjà en attente")
	ErrInUse           = errors.New("élément utilisé")
	ErrAccountMissing  = errors.New("compte introuvable")
	ErrJournalMissing  = errors.New("journal introuvable")
	ErrFrozenOperation = errors.New("opération figée")
	ErrNameTaken       = errors.New("nom déjà utilisé")
	ErrRecurringTaken  = errors.New("occurrence déjà générée")
	ErrDuplicateFile   = errors.New("pièce déjà jointe")
	ErrFileImported    = errors.New("fichier déjà importé")
	ErrReconOver       = errors.New("montant supérieur au reste à rapprocher")
	ErrReconDirection  = errors.New("sens incompatible")
	ErrReconAccount    = errors.New("ligne d'un autre compte")
	ErrReconNotPosted  = errors.New("écriture non validée")
	ErrBankLinked      = errors.New("compte comptable déjà lié à un compte bancaire")
)

// Store : persistance. Do exécute f dans UNE transaction (écritures + événement d'audit validés ensemble) ; un appel imbriqué la rejoint.
type Store interface {
	Do(ctx context.Context, f func(ctx context.Context) error) error

	GetSettings(ctx context.Context) (*domain.Settings, error)
	PutSettings(ctx context.Context, s *domain.Settings, by string) error
	AnyEntryExists(ctx context.Context) (bool, error)

	CreateAccount(ctx context.Context, a *domain.Account) error // ErrCodeTaken
	UpdateAccount(ctx context.Context, a *domain.Account) error // ErrNotFound
	AccountByID(ctx context.Context, id string) (*domain.Account, error)
	AccountsByIDs(ctx context.Context, ids []string) (map[string]*domain.Account, error)
	ListAccounts(ctx context.Context, f AccountFilter, limit, offset int) ([]*domain.Account, int, error)
	ExistingAccountCodes(ctx context.Context, codes []string) (map[string]bool, error)
	AccountUsed(ctx context.Context, id string) (bool, error)

	CreateJournal(ctx context.Context, j *domain.Journal) error // ErrCodeTaken
	UpdateJournal(ctx context.Context, j *domain.Journal) error // ErrNotFound
	JournalByID(ctx context.Context, id string) (*domain.Journal, error)
	ListJournals(ctx context.Context, active *bool) ([]*domain.Journal, error)
	ExistingJournalCodes(ctx context.Context, codes []string) (map[string]bool, error)
	JournalUsed(ctx context.Context, id string) (bool, error)

	CreateFiscalYear(ctx context.Context, y *domain.FiscalYear) error // ErrOverlap, ErrLabelTaken
	FiscalYearByID(ctx context.Context, id string) (*domain.FiscalYear, error)
	ListFiscalYears(ctx context.Context) ([]*domain.FiscalYear, error)
	PeriodByID(ctx context.Context, id string, lock bool) (*domain.FiscalPeriod, error)
	PeriodForDate(ctx context.Context, d time.Time) (*domain.FiscalPeriod, error) // ErrNoPeriod
	ClosePeriod(ctx context.Context, id, by string, at time.Time) error           // ErrHasDrafts
	ReopenPeriod(ctx context.Context, id string) error                            // ErrNotApproved
	CountDrafts(ctx context.Context, periodID string) (int, error)

	CreateApproval(ctx context.Context, a *domain.Approval) error // ErrPendingExists
	ApprovalByID(ctx context.Context, id string, lock bool) (*domain.Approval, error)
	ListApprovals(ctx context.Context, status *string, limit, offset int) ([]*domain.Approval, int, error)
	DecideApproval(ctx context.Context, id, status, by string, note *string, at time.Time) error

	CreateEntry(ctx context.Context, e *domain.Entry) error  // ErrDuplicateSource, ErrAlreadyReversed
	ReplaceEntry(ctx context.Context, e *domain.Entry) error // brouillon seulement
	DeleteEntry(ctx context.Context, id string) error
	EntryByID(ctx context.Context, id string, lock bool) (*domain.Entry, error)
	ListEntries(ctx context.Context, f EntryFilter, limit, offset int) ([]*domain.Entry, int, error)
	// PostEntry attribue le numéro suivant (compteur verrouillé dans la transaction) et passe l'écriture à « validée ».
	PostEntry(ctx context.Context, e *domain.Entry, by string, at time.Time) (int64, error) // ErrUnbalanced, ErrFourEyes, ErrPeriodClosed, ErrAccountInactive, ErrJournalInactive

	// ── F2 : opérations ──
	CreateThirdParty(ctx context.Context, t *domain.ThirdParty) error // ErrNameTaken, ErrCodeTaken
	UpdateThirdParty(ctx context.Context, t *domain.ThirdParty) error
	ThirdPartyByID(ctx context.Context, id string) (*domain.ThirdParty, error)
	ListThirdParties(ctx context.Context, query string, kind *string, limit, offset int) ([]*domain.ThirdParty, int, error)

	CreateOpCategory(ctx context.Context, c *domain.OpCategory) error // ErrNameTaken
	UpdateOpCategory(ctx context.Context, c *domain.OpCategory) error
	OpCategoryByID(ctx context.Context, id string) (*domain.OpCategory, error)
	ListOpCategories(ctx context.Context, kind *string, active *bool) ([]*domain.OpCategory, error)

	CreateOperation(ctx context.Context, o *domain.Operation) error
	UpdateOperation(ctx context.Context, o *domain.Operation) error // ErrFrozenOperation
	OperationByID(ctx context.Context, id string, lock bool) (*domain.Operation, error)
	ListOperations(ctx context.Context, f OperationFilter, limit, offset int) ([]*domain.Operation, OpTotals, int, error)
	DeleteOperation(ctx context.Context, id string) error
	SetOperationEntry(ctx context.Context, opID string, entryID *string) error
	DuplicateOperationExists(ctx context.Context, o *domain.Operation) (bool, error)
	AddOperationPayment(ctx context.Context, opID string, p domain.OperationPayment, by string) error

	CreateRecurring(ctx context.Context, r *domain.Recurring) error
	UpdateRecurring(ctx context.Context, r *domain.Recurring) error
	RecurringByID(ctx context.Context, id string) (*domain.Recurring, error)
	ListRecurring(ctx context.Context) ([]*domain.Recurring, error)
	CreateOperationIfAbsent(ctx context.Context, o *domain.Operation, templateID string, month time.Time) (bool, error)
	SetRecurringLastMonth(ctx context.Context, id string, month time.Time) error

	AddAttachment(ctx context.Context, a *domain.Attachment, data []byte, by string) error // ErrDuplicateFile
	ListAttachments(ctx context.Context, opID string) ([]*domain.Attachment, error)
	AttachmentByID(ctx context.Context, id string) (*domain.Attachment, []byte, error)
	DeleteAttachment(ctx context.Context, id string) error

	// ── F3 : trésorerie ──
	CreateBankAccount(ctx context.Context, b *domain.BankAccount, numberEnc []byte, last4 *string, by string) error // ErrCodeTaken, ErrBankLinked
	UpdateBankAccount(ctx context.Context, b *domain.BankAccount, numberEnc []byte, last4 *string, replaceNumber bool) error
	BankAccountByID(ctx context.Context, id string) (*domain.BankAccount, error)
	ListBankAccounts(ctx context.Context, active *bool) ([]*domain.BankAccount, error)
	StatementCount(ctx context.Context, bankAccountID string) (int, error)
	StatementFileExists(ctx context.Context, bankAccountID, sha string) (bool, error)
	ExistingFingerprints(ctx context.Context, bankAccountID string, fps []string) (map[string]bool, error)
	InsertStatement(ctx context.Context, st *domain.Statement) error // ErrFileImported
	InsertStatementLines(ctx context.Context, st *domain.Statement, lines []domain.ParsedLine, newID func() string) (int, error)
	ListStatements(ctx context.Context, bankAccountID string) ([]*domain.Statement, error)
	ListBankLines(ctx context.Context, f BankLineFilter, limit, offset int) ([]*domain.BankLine, int, error)
	BankLineByID(ctx context.Context, id string) (*domain.BankLine, error)
	ListLedgerLines(ctx context.Context, bankAccountID string, onlyUnmatched bool, query string, limit, offset int) ([]domain.LedgerCandidate, int, error)
	LedgerCandidates(ctx context.Context, bankAccountID string, inflow bool, limit int) ([]domain.LedgerCandidate, error)
	LedgerLineRemaining(ctx context.Context, entryLineID string) (accountID string, remaining int64, err error)
	CreateReconciliation(ctx context.Context, id, bankAccountID, lineID, entryLineID string, amount int64, by string, at time.Time) error // ErrRecon*
	ReconciliationLine(ctx context.Context, id string) (lineID string, err error)
	DeleteReconciliation(ctx context.Context, id string) error
	ReconSummary(ctx context.Context, bankAccountID string) (*domain.ReconSummary, error)
	ForecastData(ctx context.Context, today time.Time) (*ForecastData, error)

	GeneralLedger(ctx context.Context, accountID string, from, to *time.Time, limit, offset int) (*Ledger, error)
	TrialBalance(ctx context.Context, from, to time.Time) ([]TrialRow, error)
}

type AccountFilter struct {
	Query  string
	Kind   *string
	Active *bool
}

type EntryFilter struct {
	Query                        string
	Status, JournalID, AccountID *string
	From, To                     *time.Time
}

type LedgerLine struct {
	EntryID, EntryNumber, JournalCode, Label string
	AccountingDate                           time.Time
	Reference, Memo                          *string
	Debit, Credit, Balance                   int64
}

type Ledger struct {
	Account          *domain.Account
	Opening, Closing int64
	Lines            []LedgerLine
	Total            int
}

type TrialRow struct {
	AccountID, Code, Label, Kind    string
	Opening, Debit, Credit, Closing int64
}

type OperationFilter struct {
	Query                                  string
	Kind, Status, CategoryID, ThirdPartyID *string
	From, To                               *time.Time
}

type OpTotals struct{ Net, Tax, Gross int64 }

type BankLineFilter struct {
	BankAccountID, Query string
	Status               *string
	From, To             *time.Time
}

type OpenFlow struct {
	Inflow bool
	Due    *time.Time
	Amount int64
}

type ForecastData struct {
	OpeningBalance int64
	BankAccounts   int
	Open           []OpenFlow
	DraftCount     int
	DraftNetAmount int64
}

// Sealer chiffre les données sensibles au repos (AES-256-GCM, clé hors base). Il n'existe volontairement aucune méthode de déchiffrement ici :
// le numéro de compte complet n'est jamais relu par l'application.
type Sealer interface {
	Seal(plain []byte) ([]byte, error)
}
