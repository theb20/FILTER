package postgres_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/services/finance/internal/adapters/secretbox"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
)

type trEnv struct {
	*opEnv
	box  *secretbox.Box
	bank *domain.BankAccount
}

var testKey = bytes.Repeat([]byte{7}, 32)

// trRig : un compte bancaire lié au compte T-CASH (ouverture au 1er janvier 2026, solde d'ouverture 0).
func trRig(t *testing.T) *trEnv {
	t.Helper()
	o := opRig(t)
	box, err := secretbox.FromKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	o.s = application.New(application.Config{Store: o.st, Events: &outbox.Publisher{}, Box: box})
	b, err := o.s.CreateBankAccount(ctx, alice, domain.BankAccount{Name: "Compte courant test", BankName: "Banque de test", AccountID: o.cash.ID, OpeningDate: d("2026-01-01")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &trEnv{opEnv: o, box: box, bank: b}
}

// bankEntry : écriture validée sur le compte banque (positif = entrée d'argent, négatif = sortie), ligne d'écriture retournée.
func (e *trEnv) bankEntry(t *testing.T, date string, amount int64) *domain.EntryLine {
	t.Helper()
	var lines []domain.EntryLine
	if amount > 0 {
		lines = []domain.EntryLine{line(e.cash, amount, 0), line(e.sales, 0, amount)}
	} else {
		lines = []domain.EntryLine{line(e.rent, -amount, 0), line(e.cash, 0, -amount)}
	}
	en := e.draft(t, alice, e.bq, date, lines...)
	p, err := e.s.PostEntry(ctx, bob, en.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i := range p.Lines {
		if p.Lines[i].AccountID == e.cash.ID {
			return &p.Lines[i]
		}
	}
	t.Fatal("ligne banque introuvable")
	return nil
}

func (e *trEnv) importCSV(t *testing.T, csv string) *application.StatementImport {
	t.Helper()
	r, err := e.s.ImportBankStatement(ctx, alice, e.bank.ID, "releve.csv", csv, nil, false)
	if err != nil {
		t.Fatalf("import : %v", err)
	}
	return r
}

func (e *trEnv) lines(t *testing.T, status string) []*domain.BankLine {
	t.Helper()
	var st *string
	if status != "" {
		st = &status
	}
	p, err := e.s.ListBankLines(ctx, application.BankLineFilter{BankAccountID: e.bank.ID, Status: st}, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	return p.Items
}

func lineOf(t *testing.T, ls []*domain.BankLine, label string) *domain.BankLine {
	t.Helper()
	for _, l := range ls {
		if l.Label == label {
			return l
		}
	}
	t.Fatalf("ligne %q introuvable", label)
	return nil
}

func TestBankAccountNumberIsEncryptedAndNeverReturned(t *testing.T) {
	e := trRig(t)
	acc, _ := e.s.CreateAccount(ctx, alice, domain.Account{Code: "T-BK2", Label: "Banque 2", Kind: "asset"})
	b, err := e.s.CreateBankAccount(ctx, alice, domain.BankAccount{Name: "Second", BankName: "Banque", AccountID: acc.ID, OpeningDate: d("2026-01-01")}, ptrStr("CI93 CI008 01111 123456789012 34"))
	if err != nil || !b.HasNumber || b.NumberLast4 == nil || *b.NumberLast4 != "1234" || b.Currency != "XOF" {
		t.Fatalf("%+v %v", b, err)
	}
	var enc []byte
	var l4 string
	if err := e.st.Pool.QueryRow(ctx, `SELECT number_encrypted, number_last4 FROM bank_accounts WHERE id=$1`, b.ID).Scan(&enc, &l4); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(enc, []byte("123456789012")) || bytes.Contains(enc, []byte("CI93")) || l4 != "1234" {
		t.Fatal("le numéro est lisible en base")
	}
	plain, err := e.box.Open(enc)
	if err != nil || string(plain) != "CI93CI008011111234567890121234"[:0]+strings.ToUpper("CI93CI0080111112345678901234") {
		t.Fatalf("déchiffrement : %q %v", plain, err)
	}
	// deux chiffrements du même numéro diffèrent (nonce aléatoire)
	b2, _ := e.s.UpdateBankAccount(ctx, alice, b.ID, domain.BankAccount{Name: "Second", BankName: "Banque", AccountID: acc.ID, OpeningDate: d("2026-01-01")}, true, ptrStr("CI93 CI008 01111 123456789012 34"))
	var enc2 []byte
	_ = e.st.Pool.QueryRow(ctx, `SELECT number_encrypted FROM bank_accounts WHERE id=$1`, b2.ID).Scan(&enc2)
	if bytes.Equal(enc, enc2) {
		t.Fatal("deux chiffrements identiques : nonce réutilisé")
	}
	// numéro absent à la modification = inchangé ; numéro invalide refusé ; sans clé : numéro refusé, compte créé sans numéro
	if u, err := e.s.UpdateBankAccount(ctx, alice, b.ID, domain.BankAccount{Name: "Renommé", BankName: "Banque", AccountID: acc.ID, OpeningDate: d("2026-01-01")}, true, nil); err != nil || !u.HasNumber || u.Name != "Renommé" {
		t.Fatalf("%+v %v", u, err)
	}
	_, err = e.s.UpdateBankAccount(ctx, alice, b.ID, domain.BankAccount{Name: "x", BankName: "Banque", AccountID: acc.ID, OpeningDate: d("2026-01-01")}, true, ptrStr("12"))
	wantCode(t, err, apperr.CodeValidation)
	noKey := application.New(application.Config{Store: e.st, Events: &outbox.Publisher{}})
	acc3, _ := e.s.CreateAccount(ctx, alice, domain.Account{Code: "T-BK3", Label: "Banque 3", Kind: "asset"})
	_, err = noKey.CreateBankAccount(ctx, alice, domain.BankAccount{Name: "Sans clé", BankName: "Banque", AccountID: acc3.ID, OpeningDate: d("2026-01-01")}, ptrStr("CI93CI00801111123456789012"))
	wantCode(t, err, apperr.CodeConflict)
	if c, err := noKey.CreateBankAccount(ctx, alice, domain.BankAccount{Name: "Sans numéro", BankName: "Banque", AccountID: acc3.ID, OpeningDate: d("2026-01-01")}, nil); err != nil || c.HasNumber {
		t.Fatalf("%+v %v", c, err)
	}
	// le numéro n'apparaît nulle part dans l'audit
	var n int
	_ = e.st.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE convert_from(payload,'UTF8') ILIKE '%123456789012%'`).Scan(&n)
	if n != 0 {
		t.Fatalf("le numéro figure dans %d événements", n)
	}
}

func TestBankAccountRules(t *testing.T) {
	e := trRig(t)
	_, err := e.s.CreateBankAccount(ctx, alice, domain.BankAccount{Name: "Autre", BankName: "B", AccountID: e.cash.ID, OpeningDate: d("2026-01-01")}, nil)
	wantCode(t, err, apperr.CodeConflict) // compte comptable déjà lié
	_, err = e.s.CreateBankAccount(ctx, alice, domain.BankAccount{Name: "compte courant TEST", BankName: "B", AccountID: e.expAcc.ID, OpeningDate: d("2026-01-01")}, nil)
	wantCode(t, err, apperr.CodeConflict) // nom unique
	_, err = e.s.CreateBankAccount(ctx, alice, domain.BankAccount{Name: "Fantôme", BankName: "B", AccountID: "00000000-0000-0000-0000-0000000000aa", OpeningDate: d("2026-01-01")}, nil)
	wantCode(t, err, apperr.CodeValidation)
	_, err = e.s.CreateBankAccount(ctx, alice, domain.BankAccount{BankName: "B", AccountID: e.expAcc.ID, OpeningDate: d("2026-01-01")}, nil)
	wantCode(t, err, apperr.CodeValidation)
	// après un relevé importé : compte comptable lié et ouverture figés
	e.importCSV(t, "2026-01-10;Virement;R1;1000")
	_, err = e.s.UpdateBankAccount(ctx, alice, e.bank.ID, domain.BankAccount{Name: "x", BankName: "B", AccountID: e.expAcc.ID, OpeningDate: d("2026-01-01")}, true, nil)
	wantCode(t, err, apperr.CodeConflict)
	_, err = e.s.UpdateBankAccount(ctx, alice, e.bank.ID, domain.BankAccount{Name: "x", BankName: "B", AccountID: e.cash.ID, OpeningDate: d("2026-01-01"), OpeningBalance: 5}, true, nil)
	wantCode(t, err, apperr.CodeConflict)
}

func TestStatementImport(t *testing.T) {
	e := trRig(t)
	csv := "date;libelle;reference;montant\n2026-02-03;Virement client A;FAC-1;1 250 000\n03/02/2026;Frais bancaires;;-2 500\n2026-02-04;Retrait;;-10 000\n2026-02-04;Retrait;;-10 000\n"
	sim, err := e.s.ImportBankStatement(ctx, alice, e.bank.ID, "fev.csv", csv, nil, true)
	if err != nil || !sim.Simulated || sim.Total != 4 || sim.Inserted != 4 || sim.Skipped != 0 || len(sim.Errors) != 0 || sim.SumAmount != 1250000-2500-20000 || sim.PeriodStart.Format("2006-01-02") != "2026-02-03" {
		t.Fatalf("simulation : %+v %v", sim, err)
	}
	if got := e.lines(t, ""); len(got) != 0 {
		t.Fatal("la simulation a écrit")
	}
	r := e.importCSV(t, csv)
	if r.Inserted != 4 || r.Statement == nil || r.Statement.LineCount != 4 {
		t.Fatalf("import : %+v", r)
	}
	// deux retraits identiques le même jour = deux lignes (numéro d'occurrence)
	ls := e.lines(t, "")
	if len(ls) != 4 || lineOf(t, ls, "Frais bancaires").Amount != -2500 || lineOf(t, ls, "Virement client A").Amount != 1250000 || lineOf(t, ls, "Virement client A").Reference == nil {
		t.Fatalf("lignes : %+v", ls)
	}
	// le même fichier ne s'importe qu'une fois ; un fichier qui le recouvre n'ajoute que les lignes nouvelles
	_, err = e.s.ImportBankStatement(ctx, alice, e.bank.ID, "fev-bis.csv", csv, nil, false)
	wantCode(t, err, apperr.CodeConflict)
	ov := "2026-02-04;Retrait;;-10000\n2026-02-04;Retrait;;-10000\n2026-02-05;Salaire reçu;;300000\n"
	r2 := e.importCSV(t, ov)
	if r2.Inserted != 1 || r2.Skipped != 2 {
		t.Fatalf("recouvrement : %+v", r2)
	}
	if got := len(e.lines(t, "")); got != 5 {
		t.Fatalf("%d lignes", got)
	}
	// tout ou rien : une ligne en erreur refuse l'import, la simulation les détaille
	bad := "2026-03-01;Bon;;100\n2026-13-45;Date fausse;;100\n2026-03-02;Libellé;;1,5\n2026-03-03;;;100\n2025-12-31;Avant ouverture;;100\nsurvol;un;deux\n2026-03-04;Nul;;0\n"
	sim, _ = e.s.ImportBankStatement(ctx, alice, e.bank.ID, "mars.csv", bad, nil, true)
	if len(sim.Errors) != 6 || sim.Total != 7 {
		t.Fatalf("erreurs détaillées : %+v", sim.Errors)
	}
	_, err = e.s.ImportBankStatement(ctx, alice, e.bank.ID, "mars.csv", bad, nil, false)
	wantCode(t, err, apperr.CodeValidation)
	if got := len(e.lines(t, "")); got != 5 {
		t.Fatalf("un import refusé a écrit : %d", got)
	}
	_, err = e.s.ImportBankStatement(ctx, alice, e.bank.ID, "vide.csv", "\n\n", nil, false)
	wantCode(t, err, apperr.CodeValidation)
	// solde du relevé = ouverture + lignes ; relevés immuables
	b, _ := e.s.GetBankAccount(ctx, e.bank.ID)
	if b.StatementBalance != 1250000-2500-20000+300000 || b.LastStatementDate == nil || b.LastStatementDate.Format("2006-01-02") != "2026-02-05" {
		t.Fatalf("solde : %+v", b)
	}
	for name, q := range map[string]string{"modifier une ligne": `UPDATE bank_statement_lines SET amount = 1`, "supprimer une ligne": `DELETE FROM bank_statement_lines`,
		"modifier un relevé": `UPDATE bank_statements SET filename = 'x'`, "supprimer un relevé": `DELETE FROM bank_statements`} {
		if _, err := e.st.Pool.Exec(ctx, q); err == nil {
			t.Errorf("la base a accepté : %s", name)
		}
	}
	if sts, _ := e.s.ListStatements(ctx, e.bank.ID); len(sts) != 2 {
		t.Fatalf("relevés : %d", len(sts))
	}
}

func TestReconciliationFlow(t *testing.T) {
	e := trRig(t)
	in1, in2, out1 := e.bankEntry(t, "2026-02-03", 1000), e.bankEntry(t, "2026-02-03", 1000), e.bankEntry(t, "2026-02-04", -400)
	e.importCSV(t, "2026-02-03;Virement client A;FAC-1;1000\n2026-02-04;Loyer février;;-400\n2026-02-05;Frais inattendus;;-50\n")
	ls := e.lines(t, "")
	vir, loyer, frais := lineOf(t, ls, "Virement client A"), lineOf(t, ls, "Loyer février"), lineOf(t, ls, "Frais inattendus")
	// suggestions : même sens, montant égal en tête, jamais appliquées d'elles-mêmes
	cs, err := e.s.ListCandidates(ctx, vir.ID)
	if err != nil || len(cs) != 2 || !cs[0].ExactAmount || cs[0].Score <= 100 {
		t.Fatalf("candidats : %+v %v", cs, err)
	}
	for _, c := range cs {
		if c.Amount <= 0 {
			t.Fatalf("un candidat de sens inverse : %+v", c)
		}
	}
	if cs, _ := e.s.ListCandidates(ctx, loyer.ID); len(cs) != 1 || cs[0].Amount != -400 {
		t.Fatalf("candidats loyer : %+v", cs)
	}
	if lines := e.lines(t, "unmatched"); len(lines) != 3 {
		t.Fatal("aucun rapprochement ne doit exister avant confirmation")
	}
	// rapprochement complet (montant 0 = maximum)
	l, err := e.s.CreateReconciliation(ctx, alice, vir.ID, in1.ID, 0)
	if err != nil || l.Status != "matched" || l.Matched != 1000 || l.Remaining != 0 || len(l.Matches) != 1 || !strings.HasPrefix(l.Matches[0].EntryNumber, "BQ-2026-") {
		t.Fatalf("%+v %v", l, err)
	}
	if l, err = e.s.CreateReconciliation(ctx, alice, loyer.ID, out1.ID, 0); err != nil || l.Status != "matched" || l.Matched != 400 {
		t.Fatalf("%+v %v", l, err)
	}
	// règles : ligne déjà soldée, sens inverse, mauvais compte, écriture non validée, dépassement, avant l'ouverture
	_, err = e.s.CreateReconciliation(ctx, alice, vir.ID, in2.ID, 1)
	wantCode(t, err, apperr.CodeValidation)
	_, err = e.s.CreateReconciliation(ctx, alice, frais.ID, in2.ID, 0)
	wantCode(t, err, apperr.CodeValidation) // frais = sortie, ligne = débit
	other := e.draft(t, alice, e.od, "2026-02-06", line(e.rent, 50, 0), line(e.sales, 0, 50))
	if _, err := e.s.PostEntry(ctx, bob, other.ID); err != nil {
		t.Fatal(err)
	}
	oth, _ := e.s.GetEntry(ctx, other.ID)
	_, err = e.s.CreateReconciliation(ctx, alice, frais.ID, oth.Lines[0].ID, 0)
	wantCode(t, err, apperr.CodeValidation) // autre compte que le compte banque
	draftE := e.draft(t, alice, e.bq, "2026-02-06", line(e.rent, 50, 0), line(e.cash, 0, 50))
	_, err = e.s.CreateReconciliation(ctx, alice, frais.ID, draftE.Lines[1].ID, 0)
	wantCode(t, err, apperr.CodeConflict) // brouillon
	// partiel : 1 ligne de relevé de 1000 contre deux écritures (600 + 400) — saisies à part
	e.importCSV(t, "2026-02-10;Remise de chèques;;1000\n")
	remise := lineOf(t, e.lines(t, "unmatched"), "Remise de chèques")
	a6 := e.bankEntry(t, "2026-02-10", 600)
	a4 := e.bankEntry(t, "2026-02-10", 400)
	if l, err = e.s.CreateReconciliation(ctx, alice, remise.ID, a6.ID, 0); err != nil || l.Status != "partial" || l.Remaining != 400 {
		t.Fatalf("partiel : %+v %v", l, err)
	}
	_, err = e.s.CreateReconciliation(ctx, alice, remise.ID, in2.ID, 401)
	wantCode(t, err, apperr.CodeValidation) // plus que le reste du relevé
	if l, err = e.s.CreateReconciliation(ctx, alice, remise.ID, a4.ID, 0); err != nil || l.Status != "matched" || len(l.Matches) != 2 {
		t.Fatalf("%+v %v", l, err)
	}
	// l'écart non résolu reste visible : « Frais inattendus » (−50) n'a aucune écriture
	sum, err := e.s.GetReconciliationSummary(ctx, e.bank.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.UnmatchedStatement != -50 || sum.UnmatchedStatementCount != 1 || sum.UnmatchedLedger != 1000 || sum.UnmatchedLedgerCount != 1 || sum.OpeningGap != 0 || sum.Unexplained != 0 {
		t.Fatalf("résumé : %+v", sum)
	}
	// grand livre : 1000+1000−400+600+400 = 2600 ; relevé : 1000−400−50+1000 = 1550 ; écart = 1050 = 1000 (écriture non encore au relevé) + 50 (frais non comptabilisés)
	if sum.LedgerBalance != 2600 || sum.StatementBalance != 1550 || sum.Difference != 1050 {
		t.Fatalf("soldes : %+v", sum)
	}
	// annuler un rapprochement : l'écart réapparaît ; jamais absorbé
	full, _ := e.s.Store.BankLineByID(ctx, remise.ID)
	if _, err := e.s.DeleteReconciliation(ctx, alice, full.Matches[0].ID); err != nil {
		t.Fatal(err)
	}
	cur, _ := e.s.Store.BankLineByID(ctx, remise.ID)
	if cur.Status != "partial" || cur.Remaining != 600 {
		t.Fatalf("après annulation : %+v", cur)
	}
	if sum, _ = e.s.GetReconciliationSummary(ctx, e.bank.ID); sum.UnmatchedStatement != 600-50 || sum.Unexplained != 0 {
		t.Fatalf("résumé après annulation : %+v", sum)
	}
	// la base protège les invariants même sans passer par le service
	if _, err := e.st.Pool.Exec(ctx, `INSERT INTO reconciliations (id, bank_account_id, statement_line_id, entry_line_id, amount, matched_by) VALUES (gen_random_uuid(), $1, $2, $3, 1000, $4)`, e.bank.ID, frais.ID, in2.ID, alice.ID); err == nil {
		t.Fatal("la base a accepté un sens incompatible")
	}
	if _, err := e.st.Pool.Exec(ctx, `UPDATE reconciliations SET amount = amount + 1`); err == nil {
		t.Fatal("un rapprochement est modifiable")
	}
}

func TestReconciliationBeforeOpeningAndOpeningGap(t *testing.T) {
	e := trRig(t)
	// écriture d'ouverture AVANT la date d'ouverture du compte : elle ne se rapproche pas, elle se compare au solde d'ouverture
	if _, err := e.st.Pool.Exec(ctx, `UPDATE bank_accounts SET opening_date='2026-02-01', opening_balance=500 WHERE id=$1`, e.bank.ID); err != nil {
		t.Fatal(err)
	}
	openLine := e.bankEntry(t, "2026-01-31", 500)
	e.importCSV(t, "2026-02-02;Encaissement;;200\n")
	l := e.lines(t, "")[0]
	_, err := e.s.CreateReconciliation(ctx, alice, l.ID, openLine.ID, 0)
	wantCode(t, err, apperr.CodeValidation)
	sum, _ := e.s.GetReconciliationSummary(ctx, e.bank.ID)
	if sum.OpeningGap != 0 || sum.LedgerBalance != 500 || sum.StatementBalance != 700 || sum.Difference != -200 || sum.UnmatchedStatement != 200 || sum.Unexplained != 0 {
		t.Fatalf("résumé : %+v", sum)
	}
	// un solde d'ouverture différent du grand livre est signalé (non bloquant) et n'est jamais absorbé
	if _, err := e.st.Pool.Exec(ctx, `UPDATE bank_accounts SET opening_balance=450 WHERE id=$1`, e.bank.ID); err != nil {
		t.Fatal(err)
	}
	if sum, _ = e.s.GetReconciliationSummary(ctx, e.bank.ID); sum.OpeningGap != 50 || sum.Unexplained != 0 {
		t.Fatalf("écart d'ouverture : %+v", sum)
	}
}

func TestConcurrentReconciliationNeverOverAllocates(t *testing.T) {
	e := trRig(t)
	var ls []*domain.EntryLine
	for i := 0; i < 8; i++ {
		ls = append(ls, e.bankEntry(t, "2026-02-03", 1000))
	}
	e.importCSV(t, "2026-02-03;Gros virement;;1000\n")
	target := e.lines(t, "")[0]
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for _, el := range ls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.s.CreateReconciliation(ctx, alice, target.ID, el.ID, 0)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				ok++
			} else if c := code(err); c != apperr.CodeValidation {
				t.Errorf("l'échec doit être un refus de validation : %v", err)
			}
		}()
	}
	wg.Wait()
	cur, _ := e.s.Store.BankLineByID(ctx, target.ID)
	if ok != 1 || cur.Matched != 1000 || cur.Remaining != 0 {
		t.Fatalf("%d rapprochements, ligne : %+v", ok, cur)
	}
	// et la même ligne d'écriture ne se partage pas au-delà de son montant
	e.importCSV(t, "2026-02-04;V1;;600\n2026-02-04;V2;;600\n")
	var wg2 sync.WaitGroup
	okc := 0
	for _, l := range e.lines(t, "unmatched") {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			if _, err := e.s.CreateReconciliation(ctx, alice, l.ID, ls[0].ID, 0); err == nil {
				mu.Lock()
				okc++
				mu.Unlock()
			}
		}()
	}
	wg2.Wait()
	acc, rem, _ := e.st.LedgerLineRemaining(ctx, ls[0].ID)
	if acc == "" || rem < 0 || rem > 1000 {
		t.Fatalf("reste de l'écriture : %d", rem)
	}
}

// Propriété : quelles que soient les opérations de rapprochement, le contrôle de cohérence « non expliqué » vaut toujours 0.
func TestReconciliationInvariantsRandomized(t *testing.T) {
	e := trRig(t)
	rnd := rand.New(rand.NewSource(7))
	var ledger []*domain.EntryLine
	var csv strings.Builder
	for i := 0; i < 12; i++ {
		amt := int64(100 + rnd.Intn(900))
		if rnd.Intn(2) == 0 {
			amt = -amt
		}
		ledger = append(ledger, e.bankEntry(t, fmt.Sprintf("2026-03-%02d", 1+rnd.Intn(27)), amt))
		if rnd.Intn(3) > 0 {
			fmt.Fprintf(&csv, "2026-03-%02d;Mouvement %d;;%d\n", 1+rnd.Intn(27), i, amt)
		}
	}
	e.importCSV(t, csv.String())
	for round := 0; round < 60; round++ {
		bl := e.lines(t, "")
		l := bl[rnd.Intn(len(bl))]
		el := ledger[rnd.Intn(len(ledger))]
		if rnd.Intn(4) == 0 && len(l.Matches) > 0 {
			_, _ = e.s.DeleteReconciliation(ctx, alice, l.Matches[0].ID)
		} else {
			_, _ = e.s.CreateReconciliation(ctx, alice, l.ID, el.ID, int64(rnd.Intn(300)))
		}
		sum, err := e.s.GetReconciliationSummary(ctx, e.bank.ID)
		if err != nil || sum.Unexplained != 0 {
			t.Fatalf("tour %d : non expliqué = %d (%v)", round, sum.Unexplained, err)
		}
	}
	for _, l := range e.lines(t, "") {
		if l.Matched < 0 || l.Matched > abs64(l.Amount) {
			t.Fatalf("sur-allocation : %+v", l)
		}
	}
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func TestCashForecast(t *testing.T) {
	e := trRig(t)
	// solde de départ : grand livre du compte banque (écritures validées datées d'aujourd'hui ou avant)
	e.bankEntry(t, "2026-01-05", 2000000)
	today := e.s.Now().Truncate(24 * 3600e9)
	mk := func(kind string, due *string, amount int64, ref string) {
		op := e.expense(amount, 0, "on_account")
		op.Reference = ptrStr(ref)
		op.OperationDate = d("2026-01-10")
		if kind == "income" {
			op = domain.Operation{Kind: "income", OperationDate: d("2026-01-10"), ThirdPartyID: &e.customer.ID, CategoryID: e.incCat.ID, Label: "Vente " + ref, Reference: ptrStr(ref),
				AmountNet: amount, CounterpartyAccountID: e.payable.ID, SettlementMode: "on_account"}
		}
		if due != nil {
			dd := d(*due)
			op.DueDate = &dd
		}
		created, err := e.s.CreateOperation(ctx, alice, op, false)
		if err != nil {
			t.Fatal(err)
		}
		created, err = e.s.GenerateOperationEntry(ctx, alice, created.ID, e.od.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.s.PostEntry(ctx, bob, *created.EntryID); err != nil {
			t.Fatal(err)
		}
	}
	in := func(days int) *string { s := today.AddDate(0, 0, days).Format("2006-01-02"); return &s }
	mk("expense", in(3), 100000, "E1")   // 1re semaine
	mk("income", in(9), 300000, "I1")    // 2e semaine
	mk("expense", in(-5), 40000, "E2")   // en retard
	mk("income", nil, 70000, "I2")       // sans échéance
	mk("expense", in(200), 999999, "E3") // au-delà de l'horizon
	// un brouillon d'opération (non comptabilisé) est exclu et signalé
	if _, err := e.s.CreateOperation(ctx, alice, func() domain.Operation { x := e.expense(5000, 0, "immediate"); x.Reference = ptrStr("DR"); return x }(), false); err != nil {
		t.Fatal(err)
	}
	f, err := e.s.GetCashForecast(ctx, 30)
	if err != nil {
		t.Fatal(err)
	}
	if f.OpeningBalance != 2000000 || len(f.Buckets) != 5 || f.BankAccounts != 1 || f.HorizonDays != 30 {
		t.Fatalf("départ : %+v", f)
	}
	if f.Buckets[0].Outflows != 100000 || f.Buckets[0].Inflows != 0 || f.Buckets[0].Balance != 1900000 || f.Buckets[1].Inflows != 300000 || f.Buckets[1].Balance != 2200000 || f.Buckets[4].Balance != 2200000 {
		t.Fatalf("semaines : %+v", f.Buckets)
	}
	if f.OverdueOutflows != 40000 || f.UndatedInflows != 70000 || f.OverdueInflows != 0 || f.UndatedOutflows != 0 {
		t.Fatalf("à part : overdue=%d/%d undated=%d/%d", f.OverdueInflows, f.OverdueOutflows, f.UndatedInflows, f.UndatedOutflows)
	}
	if f.ExcludedDrafts != 1 || f.ExcludedDraftsAmount != -5000 || len(f.Assumptions) < 4 {
		t.Fatalf("exclus : %d %d %v", f.ExcludedDrafts, f.ExcludedDraftsAmount, f.Assumptions)
	}
	if f2, _ := e.s.GetCashForecast(ctx, 180); f2.Buckets[len(f2.Buckets)-1].Balance != 2000000-100000+300000 { // E3 (à 200 jours) reste au-delà de l'horizon de 180 jours
		t.Fatalf("horizon 180 : %d", f2.Buckets[len(f2.Buckets)-1].Balance)
	}
	_, err = e.s.GetCashForecast(ctx, 45)
	wantCode(t, err, apperr.CodeValidation)
	// occurrences récurrentes à venir (non encore générées) : projetées à leur date
	day := today.AddDate(0, 0, 10).Day()
	if day > 28 {
		day = 28
	}
	rt, err := e.s.CreateRecurring(ctx, alice, domain.Recurring{Kind: "expense", CategoryID: e.expCat.ID, Label: "Abonnement", AmountNet: 10000, CounterpartyAccountID: e.payable.ID, JournalID: e.od.ID,
		SettlementMode: "on_account", DayOfMonth: day, Start: today.AddDate(0, 0, 10)})
	if err != nil {
		t.Fatal(err)
	}
	_ = rt
	f3, _ := e.s.GetCashForecast(ctx, 90)
	var out int64
	for _, b := range f3.Buckets {
		out += b.Outflows
	}
	if out < 100000+10000 {
		t.Fatalf("la récurrence n'est pas projetée : %d", out)
	}
}

func TestTreasuryAuditEvents(t *testing.T) {
	e := trRig(t)
	count := func() int {
		var n int
		_ = e.st.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE subject = 'audit.recorded.v1'`).Scan(&n)
		return n
	}
	in1 := e.bankEntry(t, "2026-02-03", 1000)
	start := count()
	if _, err := e.s.ImportBankStatement(ctx, alice, e.bank.ID, "s.csv", "2026-02-03;V;;1000\n", nil, true); err != nil { // simulation : aucun événement
		t.Fatal(err)
	}
	e.importCSV(t, "2026-02-03;V;;1000\n") // 1
	l := e.lines(t, "")[0]
	if _, err := e.s.CreateReconciliation(ctx, alice, l.ID, in1.ID, 0); err != nil { // 2
		t.Fatal(err)
	}
	_, _ = e.s.CreateReconciliation(ctx, alice, l.ID, in1.ID, 0) // refusé : aucun événement
	if got := count() - start; got != 2 {
		t.Fatalf("%d événements d'audit, 2 attendus", got)
	}
	if _, err := e.s.DeleteReconciliation(ctx, alice, e.lines(t, "")[0].Matches[0].ID); err != nil { // 3
		t.Fatal(err)
	}
	if got := count() - start; got != 3 {
		t.Fatalf("%d événements d'audit, 3 attendus", got)
	}
}
