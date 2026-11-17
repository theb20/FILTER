package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"github.com/theb20/FILTER/backend/services/finance/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
)

var ctx = context.Background()

// Deux personnes distinctes : la double validation ne se teste qu'avec deux acteurs.
var (
	alice = application.Actor{ID: "00000000-0000-0000-0000-00000000000a", IP: "127.0.0.1", TraceID: "t-a"}
	bob   = application.Actor{ID: "00000000-0000-0000-0000-00000000000b", IP: "127.0.0.1", TraceID: "t-b"}
)

type env struct {
	s                          *application.Service
	st                         *postgres.Store
	vt, bq, od                 *domain.Journal
	cash, sales, rent, capital *domain.Account
	year                       *domain.FiscalYear
}

func d(s string) time.Time { t, _ := domain.ParseDate(s); return t }

func rig(t *testing.T) *env {
	t.Helper()
	url := testsupport.Postgres(t)
	if err := postgres.Migrate(ctx, url); err != nil {
		t.Fatal(err)
	}
	st, err := postgres.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	s := application.New(application.Config{Store: st, Events: &outbox.Publisher{}})
	e := &env{s: s, st: st}
	mkJ := func(code, kind string) *domain.Journal {
		j, err := s.CreateJournal(ctx, alice, domain.Journal{Code: code, Label: "Journal " + code, Kind: kind})
		if err != nil {
			t.Fatal(err)
		}
		return j
	}
	mkA := func(code, label, kind string) *domain.Account {
		a, err := s.CreateAccount(ctx, alice, domain.Account{Code: code, Label: label, Kind: kind})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	// Données de TEST uniquement (le service, lui, ne livre aucun plan ni journal).
	e.vt, e.bq, e.od = mkJ("VT", "sales"), mkJ("BQ", "bank"), mkJ("OD", "general")
	e.cash, e.sales, e.rent, e.capital = mkA("T-CASH", "Caisse de test", "asset"), mkA("T-SALES", "Ventes de test", "income"), mkA("T-RENT", "Loyer de test", "expense"), mkA("T-CAP", "Capital de test", "equity")
	y, err := s.CreateFiscalYear(ctx, alice, domain.FiscalYear{Label: "2026", Start: d("2026-01-01"), End: d("2026-12-31")})
	if err != nil {
		t.Fatal(err)
	}
	e.year = y
	return e
}

func code(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	if err == nil {
		return ""
	}
	return "OTHER:" + err.Error()
}

func wantCode(t *testing.T, err error, want string) {
	t.Helper()
	if got := code(err); got != want {
		t.Fatalf("code = %q, attendu %q (%v)", got, want, err)
	}
}

func line(acc *domain.Account, debit, credit int64) domain.EntryLine {
	return domain.EntryLine{AccountID: acc.ID, Debit: debit, Credit: credit}
}

func (e *env) draft(t *testing.T, a application.Actor, j *domain.Journal, date string, lines ...domain.EntryLine) *domain.Entry {
	t.Helper()
	en, err := e.s.CreateEntry(ctx, a, domain.Entry{JournalID: j.ID, AccountingDate: d(date), Label: "Écriture de test", Lines: lines})
	if err != nil {
		t.Fatalf("brouillon : %v", err)
	}
	return en
}

// sale : encaissement de 1000 (débit caisse, crédit ventes), brouillon saisi par `a`.
func (e *env) sale(t *testing.T, a application.Actor, date string, amount int64) *domain.Entry {
	return e.draft(t, a, e.vt, date, line(e.cash, amount, 0), line(e.sales, 0, amount))
}

func (e *env) posted(t *testing.T, date string, amount int64) *domain.Entry {
	t.Helper()
	en := e.sale(t, alice, date, amount)
	p, err := e.s.PostEntry(ctx, bob, en.ID)
	if err != nil {
		t.Fatalf("validation : %v", err)
	}
	return p
}

func TestSettingsAndBaseCurrency(t *testing.T) {
	e := rig(t)
	st, _ := e.s.GetSettings(ctx)
	if st.BaseCurrency != "XOF" || st.FourEyesMinAmount != 0 {
		t.Fatalf("défauts : %+v", st)
	}
	if _, err := e.s.PutSettings(ctx, alice, domain.Settings{BaseCurrency: "xof"}); code(err) != apperr.CodeValidation {
		t.Fatalf("devise invalide acceptée : %v", err)
	}
	if _, err := e.s.PutSettings(ctx, alice, domain.Settings{BaseCurrency: "XOF", FourEyesMinAmount: -1}); code(err) != apperr.CodeValidation {
		t.Fatalf("seuil négatif accepté : %v", err)
	}
	if _, err := e.s.PutSettings(ctx, alice, domain.Settings{BaseCurrency: "EUR", FourEyesMinAmount: 500}); err != nil {
		t.Fatal(err) // aucune écriture : changement permis
	}
	if _, err := e.s.PutSettings(ctx, alice, domain.Settings{BaseCurrency: "XOF", FourEyesMinAmount: 0}); err != nil {
		t.Fatal(err)
	}
	e.sale(t, alice, "2026-02-01", 100)
	_, err := e.s.PutSettings(ctx, alice, domain.Settings{BaseCurrency: "EUR"})
	wantCode(t, err, apperr.CodeConflict) // des écritures existent
}

func TestAccountsAndImport(t *testing.T) {
	e := rig(t)
	_, err := e.s.CreateAccount(ctx, alice, domain.Account{Code: "t-cash", Label: "Doublon casse différente", Kind: "asset"})
	wantCode(t, err, apperr.CodeConflict)
	for name, in := range map[string]domain.Account{
		"code vide":        {Code: "", Label: "x", Kind: "asset"},
		"code avec espace": {Code: "A B", Label: "x", Kind: "asset"},
		"nature inconnue":  {Code: "X1", Label: "x", Kind: "banane"},
		"libellé vide":     {Code: "X2", Label: "  ", Kind: "asset"},
	} {
		if _, err := e.s.CreateAccount(ctx, alice, in); code(err) != apperr.CodeValidation {
			t.Errorf("%s : %q", name, code(err))
		}
	}
	// code figé ; nature figée dès qu'il y a des écritures
	_, err = e.s.UpdateAccount(ctx, alice, e.cash.ID, domain.Account{Code: "AUTRE", Label: "x", Kind: "asset"}, true)
	wantCode(t, err, apperr.CodeValidation)
	if _, err := e.s.UpdateAccount(ctx, alice, e.cash.ID, domain.Account{Code: "T-CASH", Label: "Caisse renommée", Kind: "liability"}, true); err != nil {
		t.Fatalf("nature modifiable tant que le compte est vide : %v", err)
	}
	e.sale(t, alice, "2026-02-01", 100)
	_, err = e.s.UpdateAccount(ctx, alice, e.cash.ID, domain.Account{Code: "T-CASH", Label: "x", Kind: "asset"}, true)
	wantCode(t, err, apperr.CodeValidation)

	csv := "code;libelle;nature\nT-NEW1;Nouveau compte;asset\nT-CASH;Existe déjà;asset\n\nT-BAD;Mauvaise nature;zzz\nONLY;deux colonnes\nT-NEW1;Doublon fichier;asset\n=EVIL();Formule;expense\n"
	sim, err := e.s.ImportAccounts(ctx, alice, csv, true)
	if err != nil || !sim.Simulated || sim.Total != 6 || sim.Created != 1 || sim.Skipped != 1 || len(sim.Errors) != 4 {
		t.Fatalf("simulation : %+v %v", sim, err)
	}
	if l, _ := e.s.ListAccounts(ctx, application.AccountFilter{Query: "T-NEW1"}, 1, 10); l.Total != 0 {
		t.Fatal("la simulation a écrit")
	}
	res, err := e.s.ImportAccounts(ctx, alice, csv, false)
	if err != nil || res.Created != 1 {
		t.Fatalf("import : %+v %v", res, err)
	}
	if l, _ := e.s.ListAccounts(ctx, application.AccountFilter{Query: "T-NEW1"}, 1, 10); l.Total != 1 {
		t.Fatal("import non écrit")
	}
	again, _ := e.s.ImportAccounts(ctx, alice, csv, false)
	if again.Created != 0 || again.Skipped != 2 {
		t.Fatalf("ré-import : %+v", again)
	}
}

func TestJournalsFrozenCode(t *testing.T) {
	e := rig(t)
	_, err := e.s.CreateJournal(ctx, alice, domain.Journal{Code: "VT", Label: "Doublon", Kind: "sales"})
	wantCode(t, err, apperr.CodeConflict)
	_, err = e.s.CreateJournal(ctx, alice, domain.Journal{Code: "vt", Label: "minuscule", Kind: "sales"})
	wantCode(t, err, apperr.CodeValidation)
	_, err = e.s.UpdateJournal(ctx, alice, e.vt.ID, domain.Journal{Code: "XX", Label: "x", Kind: "sales"}, true)
	wantCode(t, err, apperr.CodeValidation)
	e.sale(t, alice, "2026-02-01", 100)
	_, err = e.s.UpdateJournal(ctx, alice, e.vt.ID, domain.Journal{Code: "VT", Label: "x", Kind: "bank"}, true)
	wantCode(t, err, apperr.CodeValidation) // nature figée : le journal porte des écritures
}

func TestFiscalYears(t *testing.T) {
	e := rig(t)
	if len(e.year.Periods) != 12 || e.year.Status() != "open" || e.year.Periods[0].Start.Format("2006-01-02") != "2026-01-01" || e.year.Periods[11].End.Format("2006-01-02") != "2026-12-31" {
		t.Fatalf("périodes : %+v", e.year.Periods)
	}
	// 18 mois à cheval, premier mois partiel
	y, err := e.s.CreateFiscalYear(ctx, alice, domain.FiscalYear{Label: "Long", Start: d("2027-01-15"), End: d("2028-06-30")})
	if err != nil || len(y.Periods) != 18 || y.Periods[0].End.Format("2006-01-02") != "2027-01-31" {
		t.Fatalf("exercice long : %d %v", len(y.Periods), err)
	}
	for name, in := range map[string]domain.FiscalYear{
		"chevauchement":   {Label: "Chevauche", Start: d("2026-12-15"), End: d("2027-01-20")},
		"libellé repris":  {Label: "2026", Start: d("2030-01-01"), End: d("2030-12-31")},
		"fin avant début": {Label: "Inverse", Start: d("2031-02-01"), End: d("2031-01-01")},
		"plus de 24 mois": {Label: "Trop long", Start: d("2032-01-01"), End: d("2034-06-01")},
	} {
		_, err := e.s.CreateFiscalYear(ctx, alice, in)
		if c := code(err); c != apperr.CodeConflict && c != apperr.CodeValidation {
			t.Errorf("%s : %q", name, c)
		}
	}
}

func TestDraftValidation(t *testing.T) {
	e := rig(t)
	ghost := "00000000-0000-0000-0000-0000000000aa"
	inactive, _ := e.s.CreateAccount(ctx, alice, domain.Account{Code: "T-OFF", Label: "Inactif", Kind: "asset"})
	if _, err := e.s.UpdateAccount(ctx, alice, inactive.ID, domain.Account{Code: "T-OFF", Label: "Inactif", Kind: "asset"}, false); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*domain.Entry){
		"une seule ligne": func(x *domain.Entry) { x.Lines = x.Lines[:1] },
		"débit et crédit": func(x *domain.Entry) { x.Lines[0].Credit = 5 },
		"ligne à zéro":    func(x *domain.Entry) { x.Lines[0].Debit = 0 },
		"montant négatif": func(x *domain.Entry) { x.Lines[0].Debit = -5 },
		"compte inconnu":  func(x *domain.Entry) { x.Lines[0].AccountID = ghost },
		"compte inactif":  func(x *domain.Entry) { x.Lines[0].AccountID = inactive.ID },
		"journal inconnu": func(x *domain.Entry) { x.JournalID = ghost },
		"hors exercice":   func(x *domain.Entry) { x.AccountingDate = d("2031-01-01") },
		"libellé vide":    func(x *domain.Entry) { x.Label = " " },
		"débordement":     func(x *domain.Entry) { x.Lines[0].Debit = 1 << 62; x.Lines[1].Debit = 1 << 62; x.Lines[1].Credit = 0 },
	} {
		in := domain.Entry{JournalID: e.vt.ID, AccountingDate: d("2026-03-01"), Label: "t", Lines: []domain.EntryLine{line(e.cash, 10, 0), line(e.sales, 0, 10)}}
		mut(&in)
		if _, err := e.s.CreateEntry(ctx, alice, in); code(err) != apperr.CodeValidation {
			t.Errorf("%s : code %q (%v)", name, code(err), err)
		}
	}
	// un brouillon peut être déséquilibré ; il n'a ni numéro ni validateur
	dr := e.draft(t, alice, e.vt, "2026-03-01", line(e.cash, 10, 0), line(e.sales, 0, 7))
	if dr.Status != "draft" || dr.EntryNumber != nil || dr.Number() != nil || dr.PostedBy != nil || dr.Currency != "XOF" || dr.Origin != "manual" {
		t.Fatalf("brouillon : %+v", dr)
	}
	_, err := e.s.PostEntry(ctx, bob, dr.ID)
	wantCode(t, err, apperr.CodeValidation) // déséquilibrée
	// modification du brouillon, puis validation
	dr, err = e.s.UpdateEntry(ctx, alice, dr.ID, domain.Entry{JournalID: e.vt.ID, AccountingDate: d("2026-03-02"), Label: "corrigé", Lines: []domain.EntryLine{line(e.cash, 10, 0), line(e.sales, 0, 10)}})
	if err != nil || dr.Label != "corrigé" || dr.TotalDebit != 10 || dr.TotalCredit != 10 {
		t.Fatalf("%+v %v", dr, err)
	}
	if p, err := e.s.PostEntry(ctx, bob, dr.ID); err != nil || p.Status != "posted" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestPostingNumbersAndFourEyes(t *testing.T) {
	e := rig(t)
	a := e.sale(t, alice, "2026-02-01", 1000)
	// même auteur et validateur : refusé (seuil 0 = toujours deux personnes)
	_, err := e.s.PostEntry(ctx, alice, a.ID)
	wantCode(t, err, apperr.CodeSelfActionForbidden)
	if cur, _ := e.s.GetEntry(ctx, a.ID); cur.Status != "draft" {
		t.Fatal("l'écriture ne doit pas avoir changé")
	}
	p, err := e.s.PostEntry(ctx, bob, a.ID)
	if err != nil || p.EntryNumber == nil || *p.EntryNumber != 1 || *p.Number() != "VT-2026-000001" || p.PostedBy == nil || *p.PostedBy != bob.ID || p.PostedAt == nil {
		t.Fatalf("%+v %v", p, err)
	}
	_, err = e.s.PostEntry(ctx, bob, a.ID)
	wantCode(t, err, apperr.CodeConflict) // déjà validée
	// numérotation indépendante par journal ; continue dans un journal
	b := e.draft(t, alice, e.bq, "2026-02-02", line(e.cash, 50, 0), line(e.capital, 0, 50))
	pb, _ := e.s.PostEntry(ctx, bob, b.ID)
	c := e.sale(t, alice, "2026-02-03", 20)
	pc, _ := e.s.PostEntry(ctx, bob, c.ID)
	if *pb.Number() != "BQ-2026-000001" || *pc.Number() != "VT-2026-000002" {
		t.Fatalf("numéros : %s %s", *pb.Number(), *pc.Number())
	}
	// seuil : sous le seuil l'auteur valide lui-même, au-dessus non
	if _, err := e.s.PutSettings(ctx, alice, domain.Settings{BaseCurrency: "XOF", FourEyesMinAmount: 500}); err != nil {
		t.Fatal(err)
	}
	small := e.sale(t, alice, "2026-02-04", 499)
	if _, err := e.s.PostEntry(ctx, alice, small.ID); err != nil {
		t.Fatalf("sous le seuil : %v", err)
	}
	big := e.sale(t, alice, "2026-02-04", 500)
	_, err = e.s.PostEntry(ctx, alice, big.ID)
	wantCode(t, err, apperr.CodeSelfActionForbidden)
	// la base refuse aussi, même sans passer par le service
	if _, err := e.st.Pool.Exec(ctx, `UPDATE entries SET status='posted', entry_number=99, posted_by=created_by, posted_at=now() WHERE id=$1`, big.ID); err == nil {
		t.Fatal("la base a accepté un auteur = validateur")
	}
	// un compte inactif bloque la validation
	d1 := e.sale(t, alice, "2026-02-05", 10)
	if _, err := e.s.UpdateAccount(ctx, alice, e.sales.ID, domain.Account{Code: "T-SALES", Label: "Ventes", Kind: "income"}, false); err != nil {
		t.Fatal(err)
	}
	_, err = e.s.PostEntry(ctx, bob, d1.ID)
	wantCode(t, err, apperr.CodeValidation)
}

func TestNumberingGaplessUnderConcurrency(t *testing.T) {
	e := rig(t)
	// un échec de validation ne consomme AUCUN numéro
	bad := e.draft(t, alice, e.vt, "2026-02-01", line(e.cash, 10, 0), line(e.sales, 0, 9))
	if _, err := e.s.PostEntry(ctx, bob, bad.ID); err == nil {
		t.Fatal("écriture déséquilibrée validée")
	}
	const n = 24
	ids := make([]string, n)
	for i := range ids {
		ids[i] = e.sale(t, alice, "2026-02-10", int64(100+i)).ID
	}
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.s.PostEntry(ctx, bob, id); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("validation concurrente : %v", err)
	}
	rows, err := e.st.Pool.Query(ctx, `SELECT entry_number FROM entries WHERE status='posted' ORDER BY entry_number`)
	if err != nil {
		t.Fatal(err)
	}
	i := 0
	for rows.Next() {
		var num int
		_ = rows.Scan(&num)
		i++
		if num != i {
			t.Fatalf("numérotation avec trou : %d attendu %d", num, i)
		}
	}
	if i != n {
		t.Fatalf("%d écritures validées, %d attendues", i, n)
	}
	// le brouillon déséquilibré n'a toujours pas de numéro, et le suivant est bien n+1
	if cur, _ := e.s.GetEntry(ctx, bad.ID); cur.EntryNumber != nil {
		t.Fatal("un brouillon a un numéro")
	}
	next := e.sale(t, alice, "2026-02-11", 5)
	pn, _ := e.s.PostEntry(ctx, bob, next.ID)
	if *pn.EntryNumber != n+1 {
		t.Fatalf("numéro suivant : %d", *pn.EntryNumber)
	}
}

func TestPostedEntriesAreImmutable(t *testing.T) {
	e := rig(t)
	p := e.posted(t, "2026-02-01", 1000)
	_, err := e.s.UpdateEntry(ctx, alice, p.ID, domain.Entry{JournalID: e.vt.ID, AccountingDate: d("2026-02-01"), Label: "x", Lines: []domain.EntryLine{line(e.cash, 1, 0), line(e.sales, 0, 1)}})
	wantCode(t, err, apperr.CodeConflict)
	wantCode(t, e.s.DeleteEntry(ctx, alice, p.ID), apperr.CodeConflict)
	// la base elle-même refuse, par toutes les voies
	for name, q := range map[string]string{
		"modifier le libellé": `UPDATE entries SET label='piraté' WHERE id=$1`,
		"supprimer":           `DELETE FROM entries WHERE id=$1`,
		"modifier une ligne":  `UPDATE entry_lines SET debit=debit+1 WHERE entry_id=$1`,
		"supprimer une ligne": `DELETE FROM entry_lines WHERE entry_id=$1`,
		"ajouter une ligne":   fmt.Sprintf(`INSERT INTO entry_lines (id, entry_id, line_no, account_id, debit) VALUES (gen_random_uuid(), $1, 9, '%s', 5)`, e.cash.ID),
		"dévalider":           `UPDATE entries SET status='draft', entry_number=NULL, posted_by=NULL, posted_at=NULL WHERE id=$1`,
		"renuméroter":         `UPDATE entries SET entry_number=777 WHERE id=$1`,
		"changer la période":  fmt.Sprintf(`UPDATE entries SET fiscal_period_id='%s' WHERE id=$1`, e.year.Periods[5].ID),
	} {
		if _, err := e.st.Pool.Exec(ctx, q, p.ID); err == nil {
			t.Errorf("la base a accepté : %s", name)
		}
	}
	// un brouillon se supprime
	dr := e.sale(t, alice, "2026-02-02", 5)
	if err := e.s.DeleteEntry(ctx, alice, dr.ID); err != nil {
		t.Fatal(err)
	}
	_, err = e.s.GetEntry(ctx, dr.ID)
	wantCode(t, err, apperr.CodeNotFound)
	// l'écriture validée est intacte
	cur, _ := e.s.GetEntry(ctx, p.ID)
	if cur.Label != "Écriture de test" || cur.TotalDebit != 1000 {
		t.Fatalf("%+v", cur)
	}
}

func TestDirectInsertCannotBypassDraft(t *testing.T) {
	e := rig(t)
	_, err := e.st.Pool.Exec(ctx, `INSERT INTO entries (id, journal_id, fiscal_year_id, fiscal_period_id, accounting_date, status, entry_number, label, currency, created_by, posted_by, posted_at)
		VALUES (gen_random_uuid(), $1, $2, $3, '2026-01-10', 'posted', 1, 'direct', 'XOF', $4, $5, now())`, e.vt.ID, e.year.ID, e.year.Periods[0].ID, alice.ID, bob.ID)
	if err == nil {
		t.Fatal("une écriture a été insérée directement à l'état validé")
	}
}

func TestReversal(t *testing.T) {
	e := rig(t)
	dr := e.sale(t, alice, "2026-02-01", 700)
	_, err := e.s.ReverseEntry(ctx, alice, dr.ID, "motif valable", nil)
	wantCode(t, err, apperr.CodeConflict) // un brouillon ne se contre-passe pas
	p, _ := e.s.PostEntry(ctx, bob, dr.ID)
	_, err = e.s.ReverseEntry(ctx, alice, p.ID, "x", nil)
	wantCode(t, err, apperr.CodeValidation) // motif trop court
	r, err := e.s.ReverseEntry(ctx, alice, p.ID, "Erreur de saisie du client", ptrTime(d("2026-02-15")))
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "draft" || r.Origin != "reversal" || r.ReversesEntryID == nil || *r.ReversesEntryID != p.ID || r.ReversalReason == nil ||
		r.Lines[0].Credit != 700 || r.Lines[1].Debit != 700 || r.AccountingDate.Format("2006-01-02") != "2026-02-15" || !strings.HasPrefix(r.Label, "Contre-passation de VT-2026-000001") {
		t.Fatalf("contre-passation : %+v", r)
	}
	if o, _ := e.s.GetEntry(ctx, p.ID); o.ReversedByEntryID == nil || *o.ReversedByEntryID != r.ID {
		t.Fatalf("lien inverse : %+v", o.ReversedByEntryID)
	}
	// ni deux contre-passations, ni contre-passation d'une contre-passation, ni auto-validation
	_, err = e.s.ReverseEntry(ctx, alice, p.ID, "une seconde fois", nil)
	wantCode(t, err, apperr.CodeConflict)
	_, err = e.s.PostEntry(ctx, alice, r.ID)
	wantCode(t, err, apperr.CodeSelfActionForbidden)
	pr, err := e.s.PostEntry(ctx, bob, r.ID)
	if err != nil || pr.Status != "posted" {
		t.Fatal(err)
	}
	_, err = e.s.ReverseEntry(ctx, bob, pr.ID, "annuler l'annulation", nil)
	wantCode(t, err, apperr.CodeConflict)
	// l'effet net est nul, et l'historique reste lisible
	tb, _ := e.s.GetTrialBalance(ctx, d("2026-01-01"), d("2026-12-31"))
	for _, row := range tb.Rows {
		if row.Closing != 0 {
			t.Fatalf("solde non nul après contre-passation : %+v", row)
		}
	}
	if !tb.Balanced || tb.TotalDebit != 1400 {
		t.Fatalf("balance : %+v", tb)
	}
	// la base protège le lien (un original ne peut avoir qu'une contre-passation)
	if _, err := e.st.Pool.Exec(ctx, `UPDATE entries SET reverses_entry_id = NULL WHERE id=$1`, r.ID); err == nil {
		t.Fatal("la base a laissé détacher une contre-passation validée")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestConcurrentReversalCreatesOneDraft(t *testing.T) {
	e := rig(t)
	p := e.posted(t, "2026-02-01", 100)
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.s.ReverseEntry(ctx, alice, p.ID, "contre-passation concurrente", nil)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				ok++
			} else if code(err) != apperr.CodeConflict {
				t.Errorf("l'échec doit être un conflit propre : %v", err)
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("%d contre-passations créées", ok)
	}
}

func TestPeriodCloseAndApprovedReopen(t *testing.T) {
	e := rig(t)
	feb := e.year.Periods[1]
	dr := e.sale(t, alice, "2026-02-10", 100)
	_, err := e.s.ClosePeriod(ctx, alice, feb.ID)
	wantCode(t, err, apperr.CodeConflict) // un brouillon subsiste
	if _, err := e.st.Pool.Exec(ctx, `UPDATE fiscal_periods SET status='closed', closed_at=now(), closed_by=$2 WHERE id=$1`, feb.ID, alice.ID); err == nil {
		t.Fatal("la base a clos une période avec un brouillon")
	}
	if _, err := e.s.PostEntry(ctx, bob, dr.ID); err != nil {
		t.Fatal(err)
	}
	p, err := e.s.ClosePeriod(ctx, alice, feb.ID)
	if err != nil || p.Status != "closed" || p.ClosedBy == nil {
		t.Fatalf("%+v %v", p, err)
	}
	if y, _ := e.s.GetFiscalYear(ctx, e.year.ID); y.Status() != "partially_closed" {
		t.Fatalf("statut de l'exercice : %s", y.Status())
	}
	// plus aucune écriture, brouillon ou non, dans la période close
	_, err = e.s.CreateEntry(ctx, alice, domain.Entry{JournalID: e.vt.ID, AccountingDate: d("2026-02-20"), Label: "t", Lines: []domain.EntryLine{line(e.cash, 1, 0), line(e.sales, 0, 1)}})
	wantCode(t, err, apperr.CodeValidation)
	if _, err := e.st.Pool.Exec(ctx, `UPDATE fiscal_periods SET status='open', closed_at=NULL, closed_by=NULL WHERE id=$1`, feb.ID); err == nil {
		t.Fatal("la base a rouvert une période sans approbation")
	}
	_, err = e.s.ClosePeriod(ctx, alice, feb.ID)
	wantCode(t, err, apperr.CodeConflict) // déjà close
	// réouverture : demande, puis approbation par UNE AUTRE personne
	_, err = e.s.RequestPeriodReopen(ctx, alice, e.year.Periods[2].ID, "période encore ouverte")
	wantCode(t, err, apperr.CodeConflict)
	_, err = e.s.RequestPeriodReopen(ctx, alice, feb.ID, "oui")
	wantCode(t, err, apperr.CodeValidation)
	ap, err := e.s.RequestPeriodReopen(ctx, alice, feb.ID, "Régularisation d'une facture oubliée")
	if err != nil || ap.Status != "pending" || ap.SubjectLabel != "2026-02-01 → 2026-02-28" {
		t.Fatalf("%+v %v", ap, err)
	}
	_, err = e.s.RequestPeriodReopen(ctx, bob, feb.ID, "seconde demande identique")
	wantCode(t, err, apperr.CodeConflict)
	_, err = e.s.DecideApproval(ctx, alice, ap.ID, true, nil)
	wantCode(t, err, apperr.CodeSelfActionForbidden)
	if cur, _ := e.s.Store.PeriodByID(ctx, feb.ID, false); cur.Status != "closed" {
		t.Fatal("la période a été rouverte sans approbation valide")
	}
	// la base refuse aussi une approbation par le demandeur
	if _, err := e.st.Pool.Exec(ctx, `UPDATE approval_requests SET status='approved', decided_by=requested_by, decided_at=now() WHERE id=$1`, ap.ID); err == nil {
		t.Fatal("la base a accepté un approbateur = demandeur")
	}
	done, err := e.s.DecideApproval(ctx, bob, ap.ID, true, nil)
	if err != nil || done.Status != "approved" || done.DecidedBy == nil || *done.DecidedBy != bob.ID {
		t.Fatalf("%+v %v", done, err)
	}
	if cur, _ := e.s.Store.PeriodByID(ctx, feb.ID, false); cur.Status != "open" {
		t.Fatal("période non rouverte")
	}
	_, err = e.s.DecideApproval(ctx, bob, ap.ID, true, nil)
	wantCode(t, err, apperr.CodeConflict) // déjà traitée
	// une approbation ne sert qu'UNE fois : après une nouvelle clôture, la base refuse de rouvrir avec la même
	if _, err := e.s.ClosePeriod(ctx, alice, feb.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.st.Pool.Exec(ctx, `UPDATE fiscal_periods SET status='open', closed_at=NULL, closed_by=NULL WHERE id=$1`, feb.ID); err == nil {
		t.Fatal("une approbation a servi deux fois")
	}
	// rejet
	ap2, _ := e.s.RequestPeriodReopen(ctx, alice, feb.ID, "Nouvelle demande de réouverture")
	rj, err := e.s.DecideApproval(ctx, bob, ap2.ID, false, ptrStr("non justifié"))
	if err != nil || rj.Status != "rejected" || rj.DecisionNote == nil {
		t.Fatalf("%+v %v", rj, err)
	}
	if l, _ := e.s.ListApprovals(ctx, ptrStr("pending"), 1, 10); l.Total != 0 {
		t.Fatalf("en attente : %d", l.Total)
	}
	if l, _ := e.s.ListApprovals(ctx, nil, 1, 10); l.Total != 2 {
		t.Fatalf("toutes : %d", l.Total)
	}
}

func ptrStr(s string) *string { return &s }

func TestReports(t *testing.T) {
	e := rig(t)
	e.posted(t, "2026-01-10", 1000) // ouverture à janvier
	e.posted(t, "2026-02-05", 250)
	e.draft(t, alice, e.vt, "2026-02-06", line(e.cash, 999, 0), line(e.sales, 0, 999)) // brouillon : ignoré
	rent := e.draft(t, alice, e.od, "2026-02-07", line(e.rent, 400, 0), line(e.cash, 0, 400))
	if _, err := e.s.PostEntry(ctx, bob, rent.ID); err != nil {
		t.Fatal(err)
	}
	tb, err := e.s.GetTrialBalance(ctx, d("2026-02-01"), d("2026-02-28"))
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]application.TrialRow{}
	for _, r := range tb.Rows {
		by[r.Code] = r
	}
	cash := by["T-CASH"]
	if cash.Opening != 1000 || cash.Debit != 250 || cash.Credit != 400 || cash.Closing != 850 {
		t.Fatalf("caisse : %+v", cash)
	}
	if s := by["T-SALES"]; s.Opening != -1000 || s.Credit != 250 || s.Closing != -1250 {
		t.Fatalf("ventes : %+v", s)
	}
	if !tb.Balanced || tb.TotalDebit != 650 || tb.TotalCredit != 650 {
		t.Fatalf("balance : %+v", tb)
	}
	if _, err := e.s.GetTrialBalance(ctx, d("2026-03-01"), d("2026-02-01")); code(err) != apperr.CodeValidation {
		t.Fatalf("période inversée : %v", err)
	}
	// grand livre : solde d'ouverture, cumul, brouillon exclu, pagination
	l, err := e.s.GeneralLedger(ctx, e.cash.ID, ptrTime(d("2026-02-01")), ptrTime(d("2026-02-28")), 1, 10)
	if err != nil || l.Opening != 1000 || l.Closing != 850 || len(l.Lines) != 2 || l.Lines[0].Balance != 1250 || l.Lines[1].Balance != 850 || l.Currency != "XOF" {
		t.Fatalf("grand livre : %+v %v", l, err)
	}
	if l.Lines[0].EntryNumber != "VT-2026-000002" {
		t.Fatalf("numéro : %s", l.Lines[0].EntryNumber)
	}
	p1, _ := e.s.GeneralLedger(ctx, e.cash.ID, nil, nil, 1, 2)
	p2, _ := e.s.GeneralLedger(ctx, e.cash.ID, nil, nil, 2, 2)
	if p1.Total != 3 || len(p1.Lines) != 2 || len(p2.Lines) != 1 || p2.Lines[0].Balance != p1.Closing || p1.Pages != 2 {
		t.Fatalf("pagination : %+v %+v", p1.Lines, p2.Lines)
	}
	// export : BOM, séparateur « ; », neutralisation des formules
	bad, _ := e.s.CreateAccount(ctx, alice, domain.Account{Code: "T-F", Label: "=HYPERLINK(\"x\")", Kind: "other"})
	f := e.draft(t, alice, e.od, "2026-03-01", line(bad, 5, 0), line(e.cash, 0, 5))
	if _, err := e.s.PostEntry(ctx, bob, f.ID); err != nil {
		t.Fatal(err)
	}
	csv, err := e.s.ExportTrialBalance(ctx, d("2026-01-01"), d("2026-12-31"))
	s := string(csv)
	if err != nil || !strings.HasPrefix(s, "\xef\xbb\xbfcompte;libelle") || strings.Contains(s, ";=HYPERLINK") || !strings.Contains(s, "'=HYPERLINK") || !strings.Contains(s, "TOTAL") {
		t.Fatalf("export : %v\n%s", err, s)
	}
	lc, err := e.s.ExportGeneralLedger(ctx, e.cash.ID, nil, nil)
	if err != nil || !strings.Contains(string(lc), "Solde de clôture") {
		t.Fatalf("export grand livre : %v", err)
	}
}

// Propriété : quelles que soient les écritures validées, la balance est équilibrée et le grand livre concorde avec elle.
func TestLedgerInvariantsRandomized(t *testing.T) {
	e := rig(t)
	rnd := rand.New(rand.NewSource(42))
	accs := []*domain.Account{e.cash, e.sales, e.rent, e.capital}
	for i := 0; i < 60; i++ {
		n := 2 + rnd.Intn(3)
		var lines []domain.EntryLine
		var total int64
		for k := 0; k < n-1; k++ {
			amt := int64(1 + rnd.Intn(5000))
			total += amt
			lines = append(lines, domain.EntryLine{AccountID: accs[rnd.Intn(4)].ID, Debit: amt})
		}
		lines = append(lines, domain.EntryLine{AccountID: accs[rnd.Intn(4)].ID, Credit: total})
		month := 1 + rnd.Intn(12)
		en := e.draft(t, alice, e.od, fmt.Sprintf("2026-%02d-%02d", month, 1+rnd.Intn(27)), lines...)
		if _, err := e.s.PostEntry(ctx, bob, en.ID); err != nil {
			t.Fatalf("validation : %v", err)
		}
	}
	tb, _ := e.s.GetTrialBalance(ctx, d("2026-01-01"), d("2026-12-31"))
	if !tb.Balanced {
		t.Fatalf("balance déséquilibrée : %+v", tb)
	}
	var sum int64
	for _, row := range tb.Rows {
		l, err := e.s.GeneralLedger(ctx, row.AccountID, ptrTime(d("2026-01-01")), ptrTime(d("2026-12-31")), 1, 100)
		if err != nil || l.Closing != row.Closing {
			t.Fatalf("grand livre ≠ balance pour %s : %d / %d (%v)", row.Code, l.Closing, row.Closing, err)
		}
		if l.Total > 0 && l.Lines[len(l.Lines)-1].Balance != l.Closing && l.Total <= 100 {
			t.Fatalf("cumul final ≠ clôture pour %s", row.Code)
		}
		sum += row.Closing
	}
	if sum != 0 {
		t.Fatalf("Σ soldes = %d", sum)
	}
	// à une date intermédiaire aussi
	mid, _ := e.s.GetTrialBalance(ctx, d("2026-04-01"), d("2026-08-31"))
	if !mid.Balanced {
		t.Fatalf("balance intermédiaire : %+v", mid)
	}
}

func TestAuditEvents(t *testing.T) {
	e := rig(t)
	count := func() int {
		var n int
		_ = e.st.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE subject = 'audit.recorded.v1'`).Scan(&n)
		return n
	}
	start := count()                                                                                                             // journaux, comptes, exercice de l'installation de test
	a := e.sale(t, alice, "2026-02-01", 10)                                                                                      // création
	_, _ = e.s.CreateEntry(ctx, alice, domain.Entry{JournalID: e.vt.ID, AccountingDate: d("2026-02-01"), Label: "", Lines: nil}) // refus : aucun événement
	_, _ = e.s.PostEntry(ctx, alice, a.ID)                                                                                       // refus (même personne)
	if _, err := e.s.PostEntry(ctx, bob, a.ID); err != nil {                                                                     // validation
		t.Fatal(err)
	}
	if _, err := e.s.ReverseEntry(ctx, alice, a.ID, "motif de contre-passation", nil); err != nil { // contre-passation
		t.Fatal(err)
	}
	if got := count() - start; got != 3 {
		t.Fatalf("%d événements d'audit, 3 attendus (création, validation, contre-passation)", got)
	}
	var action, actor string
	if err := e.st.Pool.QueryRow(ctx, `SELECT convert_from(payload, 'UTF8')::jsonb->>'action', convert_from(payload, 'UTF8')::jsonb->>'actorId' FROM outbox WHERE subject='audit.recorded.v1' ORDER BY id DESC LIMIT 1`).Scan(&action, &actor); err != nil {
		t.Logf("lecture du payload d'audit : %v", err) // la forme dépend de l'enveloppe ; le décompte ci-dessus fait foi
	}
}
