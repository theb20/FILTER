package domain

import (
	"strings"
	"testing"
)

// Toute règle doit être complète et cohérente avec son audience : c'est le garde-fou d'une table écrite à la main.
func TestEveryRuleIsWellFormed(t *testing.T) {
	for _, r := range Rules {
		name := r.Event + " → " + r.TemplateKey
		if r.Event == "" || r.TemplateKey == "" {
			t.Errorf("%s : événement ou clé de gabarit vide", name)
		}
		if r.Why == "" {
			t.Errorf("%s : une règle sans raison d'être n'est pas relisible", name)
		}
		switch r.To {
		case ToEventField:
			if r.Field == "" {
				t.Errorf("%s : audience « champ de l'événement » sans nom de champ", name)
			}
			if r.Permission != "" {
				t.Errorf("%s : permission inutile pour cette audience", name)
			}
		case ToStaffWithPermission:
			if r.Permission == "" {
				t.Errorf("%s : alerte interne sans permission : personne ne la recevrait", name)
			}
			if r.Field != "" {
				t.Errorf("%s : le personnel se déduit des rôles, pas d'un champ de l'événement", name)
			}
		default:
			if r.Field != "" || r.Permission != "" {
				t.Errorf("%s : audience résolue par un autre service, ni champ ni permission attendus", name)
			}
		}
	}
}

// Un couple (événement, gabarit) ne doit apparaître qu'une fois : sinon le destinataire reçoit deux fois le même message.
func TestNoDuplicateRule(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Rules {
		k := r.Event + "|" + r.TemplateKey
		if seen[k] {
			t.Errorf("règle en double : %s", k)
		}
		seen[k] = true
	}
}

// Les clés de gabarit suivent « domaine.contexte » : la migration d'amorçage et le backoffice s'appuient dessus.
func TestTemplateKeyShape(t *testing.T) {
	for _, k := range TemplateKeys() {
		parts := strings.Split(k, ".")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			t.Errorf("clé %q : forme attendue « domaine.contexte »", k)
		}
		if strings.ToLower(k) != k {
			t.Errorf("clé %q : minuscules uniquement", k)
		}
	}
}

// Les permissions citées doivent exister dans le vocabulaire du projet (docs/DECISIONS-PHASE2.md, ligne 90) :
// une faute de frappe enverrait l'alerte à personne, sans erreur visible.
func TestStaffPermissionsExist(t *testing.T) {
	known := map[string]bool{
		"admin:read": true, "admin:write": true, "analytics:read": true, "audit:read": true,
		"catalog:read": true, "catalog:write": true, "companies:approve": true,
		"content:publish": true, "content:read": true, "content:write": true,
		"customers:privacy": true, "customers:read": true, "customers:write": true,
		"inventory:read": true, "inventory:write": true, "invoices:read": true,
		"notifications:read": true, "notifications:write": true,
		"orders:read": true, "orders:refund": true, "orders:write": true,
		"payments:read": true, "payments:reconcile": true, "payments:settings": true,
		"pricing:read": true, "pricing:write": true,
		"quotes:convert": true, "quotes:read": true, "quotes:write": true,
		"reviews:moderate": true, "reviews:read": true,
		"shipping:read": true, "shipping:write": true,
		"support:read": true, "support:write": true,
	}
	for _, p := range StaffPermissions() {
		if !known[p] {
			t.Errorf("permission %q inconnue du projet", p)
		}
	}
}

// Les règles actives correspondent EXACTEMENT aux événements publiés aujourd'hui par identity.
// Si ce test casse, c'est qu'un service s'est mis à publier (passer la règle en Live) ou a cessé de le faire.
func TestLiveEventsMatchWhatIdentityPublishes(t *testing.T) {
	published := map[string]bool{
		EvEmailVerification: true, EvPasswordReset: true, EvRegistrationExists: true,
		EvSuspiciousLogin: true, EvEmailChangeRequest: true, EvEmailChangeDone: true,
		EvStaffInvitation: true, EvStaffTwoFactorReset: true,
	}
	live := LiveEvents()
	if len(live) != len(published) {
		t.Errorf("%d événements actifs, %d publiés par identity : %v", len(live), len(published), live)
	}
	for _, e := range live {
		if !published[e] {
			t.Errorf("%s marqué actif mais aucun service ne le publie", e)
		}
	}
	// Réciproque : un événement publié sans règle part à la poubelle en silence.
	for e := range published {
		if len(RulesFor(e)) == 0 {
			t.Errorf("%s est publié mais aucune règle ne le traite", e)
		}
	}
}

// Tout message né d'un événement est transactionnel : aucune campagne ne passe par cette table.
func TestEventDrivenMessagesAreTransactional(t *testing.T) {
	for _, r := range Rules {
		if !r.Mandatory {
			t.Errorf("%s → %s : message non transactionnel dans une table pilotée par les événements ; "+
				"une campagne doit passer par la newsletter, pas par une règle", r.Event, r.TemplateKey)
		}
	}
}

// Un changement d'adresse prévient les DEUX adresses : la nouvelle (vérification) et l'ancienne (alerte).
func TestEmailChangeWarnsBothAddresses(t *testing.T) {
	rules := RulesFor(EvEmailChangeRequest)
	if len(rules) != 2 {
		t.Fatalf("%d règles pour un changement d'adresse, 2 attendues", len(rules))
	}
	fields := map[string]bool{rules[0].Field: true, rules[1].Field: true}
	if !fields["new_email"] || !fields["old_email"] {
		t.Errorf("les deux adresses doivent être prévenues, trouvé %v", fields)
	}
}

// Une demande de devis produit l'accusé au client ET l'alerte aux commerciaux.
func TestQuoteRequestNotifiesBothSides(t *testing.T) {
	rules := RulesFor(EvQuoteRequested)
	if len(rules) != 2 {
		t.Fatalf("%d règles pour une demande de devis, 2 attendues", len(rules))
	}
	var client, staff bool
	for _, r := range rules {
		if r.To == ToEventField {
			client = true
		}
		if r.To == ToStaffWithPermission && r.Permission == "quotes:write" {
			staff = true
		}
	}
	if !client || !staff {
		t.Errorf("accusé client = %v, alerte commerciale = %v", client, staff)
	}
}

func TestTemplateKeysAreDeduplicatedAndSorted(t *testing.T) {
	keys := TemplateKeys()
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Errorf("clés non triées ou en double : %q puis %q", keys[i-1], keys[i])
		}
	}
	if len(keys) == 0 {
		t.Fatal("aucune clé de gabarit")
	}
}
