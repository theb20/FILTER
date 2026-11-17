package postgres_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
)

type opEnv struct {
	*env
	payable, tax, expAcc, incAcc, bank *domain.Account
	supplier, customer                 *domain.ThirdParty
	expCat, incCat                     *domain.OpCategory
}

func opRig(t *testing.T) *opEnv {
	t.Helper()
	e := rig(t)
	o := &opEnv{env: e}
	mk := func(code, kind string) *domain.Account {
		a, err := e.s.CreateAccount(ctx, alice, domain.Account{Code: code, Label: "Compte " + code, Kind: kind})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	o.payable, o.tax, o.expAcc, o.incAcc, o.bank = mk("T-PAY", "liability"), mk("T-TAX", "asset"), mk("T-EXP", "expense"), mk("T-INC", "income"), mk("T-BANK", "asset")
	var err error
	if o.supplier, err = e.s.CreateThirdParty(ctx, alice, domain.ThirdParty{Kind: "supplier", DisplayName: "Fournisseur de test"}); err != nil {
		t.Fatal(err)
	}
	if o.customer, err = e.s.CreateThirdParty(ctx, alice, domain.ThirdParty{Kind: "customer", DisplayName: "Client de test", ExternalRef: ptrStr("cust-1")}); err != nil {
		t.Fatal(err)
	}
	nat := "fixed"
	if o.expCat, err = e.s.CreateOpCategory(ctx, alice, domain.OpCategory{Kind: "expense", Label: "Loyer de test", AccountID: o.expAcc.ID, Nature: &nat}); err != nil {
		t.Fatal(err)
	}
	if o.incCat, err = e.s.CreateOpCategory(ctx, alice, domain.OpCategory{Kind: "income", Label: "Subvention de test", AccountID: o.incAcc.ID}); err != nil {
		t.Fatal(err)
	}
	return o
}

func (o *opEnv) expense(net, tax int64, mode string) domain.Operation {
	op := domain.Operation{Kind: "expense", OperationDate: d("2026-03-10"), ThirdPartyID: &o.supplier.ID, CategoryID: o.expCat.ID, Label: "Loyer de mars", Reference: ptrStr("FF-100"),
		AmountNet: net, TaxAmount: tax, CounterpartyAccountID: o.payable.ID, SettlementMode: mode}
	if tax > 0 {
		op.TaxAccountID = &o.tax.ID
	}
	return op
}

func TestThirdPartiesAndCategories(t *testing.T) {
	o := opRig(t)
	_, err := o.s.CreateThirdParty(ctx, alice, domain.ThirdParty{Kind: "supplier", DisplayName: "FOURNISSEUR DE TEST"})
	wantCode(t, err, apperr.CodeConflict) // nom unique par nature, sans casse
	if _, err := o.s.CreateThirdParty(ctx, alice, domain.ThirdParty{Kind: "customer", DisplayName: "Fournisseur de test"}); err != nil {
		t.Fatalf("même nom, autre nature : %v", err)
	}
	_, err = o.s.CreateThirdParty(ctx, alice, domain.ThirdParty{Kind: "customer", DisplayName: "Autre", ExternalRef: ptrStr("cust-1")})
	wantCode(t, err, apperr.CodeConflict) // référence d'identité unique
	_, err = o.s.CreateThirdParty(ctx, alice, domain.ThirdParty{Kind: "voisin", DisplayName: "x"})
	wantCode(t, err, apperr.CodeValidation)
	up, err := o.s.UpdateThirdParty(ctx, alice, o.supplier.ID, domain.ThirdParty{Kind: "customer", DisplayName: "Renommé"}, true)
	if err != nil || up.Kind != "supplier" || up.DisplayName != "Renommé" {
		t.Fatalf("la nature est figée : %+v %v", up, err)
	}
	// catégorie : compte obligatoire, actif, libellé unique par type ; nature réservée aux dépenses
	_, err = o.s.CreateOpCategory(ctx, alice, domain.OpCategory{Kind: "expense", Label: "LOYER DE TEST", AccountID: o.expAcc.ID})
	wantCode(t, err, apperr.CodeConflict)
	_, err = o.s.CreateOpCategory(ctx, alice, domain.OpCategory{Kind: "expense", Label: "Sans compte"})
	wantCode(t, err, apperr.CodeValidation)
	_, err = o.s.CreateOpCategory(ctx, alice, domain.OpCategory{Kind: "expense", Label: "Compte fantôme", AccountID: "00000000-0000-0000-0000-0000000000aa"})
	wantCode(t, err, apperr.CodeValidation)
	_, err = o.s.CreateOpCategory(ctx, alice, domain.OpCategory{Kind: "income", Label: "Revenu fixe", AccountID: o.incAcc.ID, Nature: ptrStr("fixed")})
	wantCode(t, err, apperr.CodeValidation)
	if l, _ := o.s.ListOpCategories(ctx, ptrStr("expense"), nil); len(l) != 1 || l[0].AccountCode != "T-EXP" {
		t.Fatalf("catégories : %+v", l)
	}
	// un compte utilisé par une catégorie ne peut plus changer de nature ? (il reste modifiable : la règle est portée par les écritures)
	if _, err := o.s.UpdateOpCategory(ctx, alice, o.expCat.ID, domain.OpCategory{Label: "Loyer", AccountID: o.expAcc.ID}, false); err != nil {
		t.Fatal(err)
	}
}

func TestOperationValidationAndDuplicates(t *testing.T) {
	o := opRig(t)
	for name, mut := range map[string]func(*domain.Operation){
		"montant nul":               func(x *domain.Operation) { x.AmountNet = 0 },
		"taxe négative":             func(x *domain.Operation) { x.TaxAmount = -1 },
		"taxe sans compte":          func(x *domain.Operation) { x.TaxAmount = 100; x.TaxAccountID = nil },
		"échéance avant date":       func(x *domain.Operation) { dd := d("2026-03-01"); x.DueDate = &dd },
		"catégorie d'un autre type": func(x *domain.Operation) { x.CategoryID = o.incCat.ID },
		"client sur une dépense":    func(x *domain.Operation) { x.ThirdPartyID = &o.customer.ID },
		"mode inconnu":              func(x *domain.Operation) { x.SettlementMode = "plus tard" },
		"compte inexistant":         func(x *domain.Operation) { x.CounterpartyAccountID = "00000000-0000-0000-0000-0000000000aa" },
		"libellé vide":              func(x *domain.Operation) { x.Label = " " },
	} {
		in := o.expense(1000, 0, "on_account")
		mut(&in)
		if _, err := o.s.CreateOperation(ctx, alice, in, false); code(err) != apperr.CodeValidation {
			t.Errorf("%s : %q (%v)", name, code(err), err)
		}
	}
	op, err := o.s.CreateOperation(ctx, alice, o.expense(100000, 18000, "on_account"), false)
	if err != nil || op.Status != "draft" || op.Gross() != 118000 || op.Remaining() != 118000 || op.Nature == nil || *op.Nature != "fixed" || op.Currency != "XOF" || op.CategoryLabel != "Loyer de test" {
		t.Fatalf("%+v %v", op, err)
	}
	// doublon probable : confirmation explicite requise, puis tracée
	_, err = o.s.CreateOperation(ctx, alice, o.expense(100000, 18000, "on_account"), false)
	wantCode(t, err, apperr.CodeConflict)
	var ae *apperr.Error
	if !asErr(err, &ae) || len(ae.Fields) == 0 || ae.Fields[0].Code != "DUPLICATE_SUSPECTED" {
		t.Fatalf("champ de doublon : %v", err)
	}
	dup, err := o.s.CreateOperation(ctx, alice, o.expense(100000, 18000, "on_account"), true)
	if err != nil || !dup.DuplicateConfirmed {
		t.Fatalf("doublon confirmé : %+v %v", dup, err)
	}
	// montants différents ou référence absente : pas de soupçon
	if _, err := o.s.CreateOperation(ctx, alice, o.expense(100001, 18000, "on_account"), false); err != nil {
		t.Fatal(err)
	}
	noRef := o.expense(100000, 18000, "on_account")
	noRef.Reference = nil
	if _, err := o.s.CreateOperation(ctx, alice, noRef, false); err != nil {
		t.Fatal(err)
	}
	// modification d'un brouillon d'opération
	in := o.expense(250000, 0, "immediate")
	in.Reference = ptrStr("FF-200")
	upd, err := o.s.UpdateOperation(ctx, alice, op.ID, in, false)
	if err != nil || upd.AmountNet != 250000 || upd.Gross() != 250000 || upd.TaxAccountID != nil {
		t.Fatalf("%+v %v", upd, err)
	}
	if l, tot, cur, err := listOps(o, application.OperationFilter{Kind: ptrStr("expense")}); err != nil || l.Total != 4 || tot.Net != 250000+100000+100001+100000 || cur != "XOF" {
		t.Fatalf("liste : %+v %+v %v", l, tot, err)
	}
}

func asErr(err error, target **apperr.Error) bool {
	e, ok := err.(*apperr.Error)
	if ok {
		*target = e
	}
	return ok
}

func listOps(o *opEnv, f application.OperationFilter) (*application.Page[*domain.Operation], application.OpTotals, string, error) {
	return o.s.ListOperations(ctx, f, 1, 100)
}

// Cycle complet : opération → écriture en brouillon → validation par une autre personne → règlement partiel puis total.
func TestOperationEntryAndSettlement(t *testing.T) {
	o := opRig(t)
	op, _ := o.s.CreateOperation(ctx, alice, o.expense(100000, 18000, "on_account"), false)
	// règlement impossible tant que l'écriture n'est pas validée
	_, err := o.s.RecordOperationPayment(ctx, alice, op.ID, d("2026-03-20"), 1000, o.bank.ID, o.bq.ID)
	wantCode(t, err, apperr.CodeConflict)
	op, err = o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)
	if err != nil || op.Status != "entry_draft" || op.EntryID == nil || op.EntryNumber != nil {
		t.Fatalf("%+v %v", op, err)
	}
	// une seule écriture par opération
	_, err = o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)
	wantCode(t, err, apperr.CodeConflict)
	en, _ := o.s.GetEntry(ctx, *op.EntryID)
	if en.Status != "draft" || en.Origin != "operation" || en.Source == nil || en.Source.Type != "operation" || en.Source.Kind != "recognition" || en.TotalDebit != 118000 || en.TotalCredit != 118000 || len(en.Lines) != 3 {
		t.Fatalf("écriture générée : %+v", en)
	}
	if en.Lines[0].AccountCode != "T-EXP" || en.Lines[0].Debit != 100000 || en.Lines[1].AccountCode != "T-TAX" || en.Lines[1].Debit != 18000 || en.Lines[2].AccountCode != "T-PAY" || en.Lines[2].Credit != 118000 {
		t.Fatalf("lignes : %+v", en.Lines)
	}
	// opération liée : figée (service et base)
	_, err = o.s.UpdateOperation(ctx, alice, op.ID, o.expense(1, 0, "on_account"), false)
	wantCode(t, err, apperr.CodeConflict)
	wantCode(t, o.s.DeleteOperation(ctx, alice, op.ID), apperr.CodeConflict)
	if _, err := o.st.Pool.Exec(ctx, `UPDATE operations SET amount_net = amount_net + 1 WHERE id = $1`, op.ID); err == nil {
		t.Fatal("la base a modifié une opération liée à une écriture")
	}
	// double validation : l'auteur de l'écriture ne la valide pas
	_, err = o.s.PostEntry(ctx, alice, en.ID)
	wantCode(t, err, apperr.CodeSelfActionForbidden)
	if _, err := o.s.PostEntry(ctx, bob, en.ID); err != nil {
		t.Fatal(err)
	}
	op, _ = o.s.GetOperation(ctx, op.ID)
	if op.Status != "recorded" || op.EntryNumber == nil || *op.EntryNumber != "OD-2026-000001" {
		t.Fatalf("après validation : %+v", op)
	}
	// annulation de l'écriture refusée (validée)
	_, err = o.s.CancelOperationEntry(ctx, alice, op.ID)
	wantCode(t, err, apperr.CodeConflict)
	// règlements : partiel, dépassement refusé, solde
	op, err = o.s.RecordOperationPayment(ctx, alice, op.ID, d("2026-03-25"), 50000, o.bank.ID, o.bq.ID)
	if err != nil || op.SettledAmount != 50000 || op.Remaining() != 68000 || len(op.Payments) != 1 || op.Payments[0].EntryStatus != "draft" {
		t.Fatalf("règlement partiel : %+v %v", op, err)
	}
	pe, _ := o.s.GetEntry(ctx, op.Payments[0].EntryID)
	if pe.Source == nil || pe.Source.Kind != "settlement" || pe.Lines[0].AccountCode != "T-PAY" || pe.Lines[0].Debit != 50000 || pe.Lines[1].AccountCode != "T-BANK" || pe.Lines[1].Credit != 50000 {
		t.Fatalf("écriture de règlement : %+v", pe)
	}
	_, err = o.s.RecordOperationPayment(ctx, alice, op.ID, d("2026-03-26"), 68001, o.bank.ID, o.bq.ID)
	wantCode(t, err, apperr.CodeValidation)
	if op, err = o.s.RecordOperationPayment(ctx, alice, op.ID, d("2026-03-26"), 68000, o.bank.ID, o.bq.ID); err != nil || op.Remaining() != 0 || op.SettledAmount != 118000 {
		t.Fatalf("règlement du solde : %+v %v", op, err)
	}
	_, err = o.s.RecordOperationPayment(ctx, alice, op.ID, d("2026-03-27"), 1, o.bank.ID, o.bq.ID)
	wantCode(t, err, apperr.CodeValidation)
	// une opération réglée immédiatement n'a pas de règlement
	im, _ := o.s.CreateOperation(ctx, alice, func() domain.Operation { x := o.expense(500, 0, "immediate"); x.Reference = ptrStr("IM-1"); return x }(), false)
	im, _ = o.s.GenerateOperationEntry(ctx, alice, im.ID, o.od.ID)
	_, _ = o.s.PostEntry(ctx, bob, *im.EntryID)
	_, err = o.s.RecordOperationPayment(ctx, alice, im.ID, d("2026-03-27"), 1, o.bank.ID, o.bq.ID)
	wantCode(t, err, apperr.CodeConflict)
}

func TestCancelOperationEntryAndReversal(t *testing.T) {
	o := opRig(t)
	op, _ := o.s.CreateOperation(ctx, alice, o.expense(1000, 0, "immediate"), false)
	op, _ = o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)
	eid := *op.EntryID
	// brouillon d'écriture : annulable, l'opération redevient modifiable et peut en régénérer une
	op, err := o.s.CancelOperationEntry(ctx, alice, op.ID)
	if err != nil || op.Status != "draft" || op.EntryID != nil {
		t.Fatalf("%+v %v", op, err)
	}
	if _, err := o.s.GetEntry(ctx, eid); code(err) != apperr.CodeNotFound {
		t.Fatalf("le brouillon d'écriture doit être supprimé : %v", err)
	}
	op, _ = o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)
	if _, err := o.s.PostEntry(ctx, bob, *op.EntryID); err != nil {
		t.Fatal(err)
	}
	// correction d'une opération validée : contre-passation de l'écriture
	rev, err := o.s.ReverseEntry(ctx, alice, *op.EntryID, "montant saisi à tort", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.s.PostEntry(ctx, bob, rev.ID); err != nil {
		t.Fatal(err)
	}
	if op, _ = o.s.GetOperation(ctx, op.ID); op.Status != "reversed" {
		t.Fatalf("statut : %s", op.Status)
	}
}

func TestIncomeEntry(t *testing.T) {
	o := opRig(t)
	in := domain.Operation{Kind: "income", OperationDate: d("2026-04-02"), ThirdPartyID: &o.customer.ID, CategoryID: o.incCat.ID, Label: "Subvention", AmountNet: 300000, TaxAmount: 0,
		CounterpartyAccountID: o.bank.ID, SettlementMode: "immediate"}
	op, err := o.s.CreateOperation(ctx, alice, in, false)
	if err != nil || op.Nature != nil {
		t.Fatalf("%+v %v", op, err)
	}
	op, _ = o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)
	en, _ := o.s.GetEntry(ctx, *op.EntryID)
	if en.Lines[0].AccountCode != "T-BANK" || en.Lines[0].Debit != 300000 || en.Lines[1].AccountCode != "T-INC" || en.Lines[1].Credit != 300000 {
		t.Fatalf("revenu : %+v", en.Lines)
	}
	// une écriture de revenu avec taxe collectée : débit contrepartie TTC, crédits produit + taxe
	in.TaxAmount, in.TaxAccountID, in.Reference = 54000, &o.tax.ID, ptrStr("R-1")
	in.CounterpartyAccountID, in.SettlementMode = o.payable.ID, "on_account"
	op2, _ := o.s.CreateOperation(ctx, alice, in, false)
	op2, _ = o.s.GenerateOperationEntry(ctx, alice, op2.ID, o.od.ID)
	e2, _ := o.s.GetEntry(ctx, *op2.EntryID)
	if e2.TotalDebit != 354000 || e2.Lines[0].Debit != 354000 || e2.Lines[2].Credit != 54000 {
		t.Fatalf("revenu avec taxe : %+v", e2.Lines)
	}
}

func TestOperationEntryInClosedPeriodAndConcurrency(t *testing.T) {
	o := opRig(t)
	op, _ := o.s.CreateOperation(ctx, alice, o.expense(1000, 0, "immediate"), false)
	var wg sync.WaitGroup
	var mu sync.Mutex
	okc := 0
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				okc++
			} else if code(err) != apperr.CodeConflict {
				t.Errorf("l'échec doit être un conflit propre : %v", err)
			}
		}()
	}
	wg.Wait()
	var n int
	_ = o.st.Pool.QueryRow(ctx, `SELECT count(*) FROM entries WHERE source_type='operation' AND source_id=$1`, op.ID).Scan(&n)
	if okc != 1 || n != 1 {
		t.Fatalf("%d générations réussies, %d écritures", okc, n)
	}
	// anti-doublon en base : une même source ne produit qu'une écriture
	if _, err := o.st.Pool.Exec(ctx, `INSERT INTO entries (id, journal_id, fiscal_year_id, fiscal_period_id, accounting_date, label, currency, origin, source_type, source_id, source_kind, created_by)
		SELECT gen_random_uuid(), journal_id, fiscal_year_id, fiscal_period_id, accounting_date, 'doublon', currency, 'operation', source_type, source_id, source_kind, created_by FROM entries WHERE source_id = $1`, op.ID); err == nil {
		t.Fatal("la base a accepté une seconde écriture pour la même source")
	}
	// période close : la génération est refusée proprement
	late := o.expense(5, 0, "immediate")
	late.OperationDate, late.Reference = d("2026-06-10"), ptrStr("L-1")
	lo, _ := o.s.CreateOperation(ctx, alice, late, false)
	if _, err := o.s.ClosePeriod(ctx, alice, o.year.Periods[5].ID); err != nil {
		t.Fatal(err)
	}
	_, err := o.s.GenerateOperationEntry(ctx, alice, lo.ID, o.od.ID)
	wantCode(t, err, apperr.CodeValidation)
	if cur, _ := o.s.GetOperation(ctx, lo.ID); cur.EntryID != nil {
		t.Fatal("l'opération ne doit pas être liée")
	}
}

func TestRecurringGenerationIsIdempotent(t *testing.T) {
	o := opRig(t)
	r, err := o.s.CreateRecurring(ctx, alice, domain.Recurring{Kind: "expense", CategoryID: o.expCat.ID, ThirdPartyID: &o.supplier.ID, Label: "Loyer mensuel", AmountNet: 400000,
		CounterpartyAccountID: o.payable.ID, JournalID: o.od.ID, SettlementMode: "on_account", DayOfMonth: 5, Start: d("2026-01-01")})
	if err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*domain.Recurring){
		"jour 31":         func(x *domain.Recurring) { x.DayOfMonth = 31 },
		"montant nul":     func(x *domain.Recurring) { x.AmountNet = 0 },
		"fin avant début": func(x *domain.Recurring) { e := d("2025-12-31"); x.End = &e },
		"sans journal":    func(x *domain.Recurring) { x.JournalID = "" },
	} {
		in := domain.Recurring{Kind: "expense", CategoryID: o.expCat.ID, Label: "x", AmountNet: 1, CounterpartyAccountID: o.payable.ID, JournalID: o.od.ID, SettlementMode: "on_account", DayOfMonth: 5, Start: d("2026-01-01")}
		mut(&in)
		if _, err := o.s.CreateRecurring(ctx, alice, in); code(err) != apperr.CodeValidation {
			t.Errorf("%s : %q", name, code(err))
		}
	}
	c, k, err := o.s.GenerateRecurring(ctx, alice, d("2026-03-31"))
	if err != nil || c != 3 || k != 0 {
		t.Fatalf("première génération : %d %d %v", c, k, err)
	}
	c, k, _ = o.s.GenerateRecurring(ctx, alice, d("2026-03-31"))
	if c != 0 || k != 0 {
		t.Fatalf("rejeu : %d %d", c, k)
	}
	if c, _, _ := o.s.GenerateRecurring(ctx, alice, d("2026-04-04")); c != 0 { // le 5 avril n'est pas encore échu
		t.Fatalf("avant l'échéance : %d", c)
	}
	if c, _, _ := o.s.GenerateRecurring(ctx, alice, d("2026-05-05")); c != 2 {
		t.Fatalf("avril et mai : %d", c)
	}
	l, tot, _, _ := listOps(o, application.OperationFilter{})
	if l.Total != 5 || tot.Net != 2000000 {
		t.Fatalf("opérations : %d %+v", l.Total, tot)
	}
	for _, x := range l.Items {
		if x.Status != "draft" || x.EntryID != nil || x.RecurringTemplateID == nil || x.OperationDate.Day() != 5 {
			t.Fatalf("une opération récurrente est un brouillon sans écriture : %+v", x)
		}
	}
	// concurrence : deux générations simultanées ne dupliquent rien (clé d'unicité modèle × mois)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, _ = o.s.GenerateRecurring(ctx, alice, d("2026-08-31")) }()
	}
	wg.Wait()
	var n int
	_ = o.st.Pool.QueryRow(ctx, `SELECT count(*) FROM operations WHERE recurring_template_id = $1`, r.ID).Scan(&n)
	if n != 8 {
		t.Fatalf("%d occurrences (8 attendues : janvier à août)", n)
	}
	// un modèle inactif ne génère plus
	if _, err := o.s.UpdateRecurring(ctx, alice, r.ID, domain.Recurring{CategoryID: o.expCat.ID, Label: "Loyer mensuel", AmountNet: 400000, CounterpartyAccountID: o.payable.ID, JournalID: o.od.ID, SettlementMode: "on_account", DayOfMonth: 5}, false); err != nil {
		t.Fatal(err)
	}
	if c, _, _ := o.s.GenerateRecurring(ctx, alice, d("2026-12-31")); c != 0 {
		t.Fatalf("modèle inactif : %d", c)
	}
}

func TestAttachments(t *testing.T) {
	o := opRig(t)
	op, _ := o.s.CreateOperation(ctx, alice, o.expense(1000, 0, "immediate"), false)
	pdf := append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("x"), 100)...)
	a, err := o.s.UploadAttachment(ctx, alice, op.ID, `C:\fakepath\..\facture "juin".pdf`, pdf)
	if err != nil || a.ContentType != "application/pdf" || strings.ContainsAny(a.Filename, `/\"`) || a.Size != int64(len(pdf)) || len(a.SHA256) != 64 {
		t.Fatalf("%+v %v", a, err)
	}
	_, err = o.s.UploadAttachment(ctx, alice, op.ID, "doublon.pdf", pdf)
	wantCode(t, err, apperr.CodeConflict) // même contenu
	// le type est déduit du CONTENU, pas du nom ni de l'en-tête
	for name, data := range map[string][]byte{"script.pdf": []byte("<script>alert(1)</script>"), "vide.pdf": {}, "exe.png": []byte("MZ\x90\x00"), "trop.pdf": append([]byte("%PDF-"), make([]byte, 5<<20)...)} {
		if _, err := o.s.UploadAttachment(ctx, alice, op.ID, name, data); code(err) != apperr.CodeValidation {
			t.Errorf("%s accepté : %v", name, err)
		}
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), 1, 2, 3)
	if _, err := o.s.UploadAttachment(ctx, alice, op.ID, "scan.jpg", png); err != nil {
		t.Fatal(err) // nom trompeur : le contenu est un PNG
	}
	if got, _ := o.s.ListAttachments(ctx, op.ID); len(got) != 2 || got[1].ContentType != "image/png" {
		t.Fatalf("liste : %+v", got)
	}
	_, data, err := o.s.GetAttachment(ctx, a.ID)
	if err != nil || !bytes.Equal(data, pdf) {
		t.Fatalf("contenu relu : %v", err)
	}
	if cur, _ := o.s.GetOperation(ctx, op.ID); cur.AttachmentCount != 2 {
		t.Fatalf("compteur : %d", cur.AttachmentCount)
	}
	// suppression tant que l'écriture n'est pas validée ; ensuite, pièce conservée
	if err := o.s.DeleteAttachment(ctx, alice, a.ID); err != nil {
		t.Fatal(err)
	}
	op, _ = o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)
	if _, err := o.s.PostEntry(ctx, bob, *op.EntryID); err != nil {
		t.Fatal(err)
	}
	left, _ := o.s.ListAttachments(ctx, op.ID)
	wantCode(t, o.s.DeleteAttachment(ctx, alice, left[0].ID), apperr.CodeConflict)
	_, err = o.s.UploadAttachment(ctx, alice, "00000000-0000-0000-0000-0000000000aa", "x.pdf", pdf)
	wantCode(t, err, apperr.CodeNotFound)
}

func TestOperationAuditEvents(t *testing.T) {
	o := opRig(t)
	count := func() int {
		var n int
		_ = o.st.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE subject = 'audit.recorded.v1'`).Scan(&n)
		return n
	}
	start := count()
	op, _ := o.s.CreateOperation(ctx, alice, o.expense(100, 0, "immediate"), false)  // 1
	_, _ = o.s.CreateOperation(ctx, alice, domain.Operation{Kind: "expense"}, false) // refusé
	op, _ = o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)                   // 2 (opération) + 1 (écriture) = 3
	_, _ = o.s.GenerateOperationEntry(ctx, alice, op.ID, o.od.ID)                    // refusé
	if got := count() - start; got != 3 {
		t.Fatalf("%d événements d'audit, 3 attendus", got)
	}
	_ = time.Now
}
