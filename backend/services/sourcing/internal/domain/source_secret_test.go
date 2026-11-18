package domain

import "testing"

// Le champ `secretRef` ne porte que le NOM d'une variable d'environnement : y coller la clé (ou la ligne entière
// « NOM=valeur ») ferait entrer un secret en base et dans le journal d'audit.
func TestSecretRefRefuseUneCle(t *testing.T) {
	base := func(ref string) *Source {
		return &Source{Kind: KindWebSearchAPI, Name: "Brave", BaseURL: "https://api.search.brave.com/res/v1/web/search",
			SecretRef: ref, Currency: "USD", TimeoutMs: 3000}
	}
	for ref, ok := range map[string]bool{
		"BRAVE_SEARCH_API_KEY":                      true,
		"  BRAVE_SEARCH_API_KEY  ":                  true, // espaces autour : tolérés et retirés
		"BRAVE_SEARCH_API_KEY=BSA_cle_secrete_1234": false,
		"BSA_cle_secrete_1234":                      false, // minuscules : ce n'est pas un nom de variable
		"brave_search_api_key":                      false,
		"BRAVE SEARCH":                              false,
		"":                                          false,
	} {
		err := ValidateSource(base(ref))
		if (err == nil) != ok {
			t.Errorf("secretRef %q : erreur = %v", ref, err)
		}
	}
	s := base("  BRAVE_SEARCH_API_KEY  ")
	if err := ValidateSource(s); err != nil || s.SecretRef != "BRAVE_SEARCH_API_KEY" {
		t.Fatalf("normalisation : %q (%v)", s.SecretRef, err)
	}
}
