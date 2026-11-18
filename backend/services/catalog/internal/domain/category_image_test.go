package domain

import "testing"

func TestImageDeCategorieFacultative(t *testing.T) {
	ptr := func(s string) *string { return &s }
	base := func(img *string) *Category {
		return &Category{Slug: "filtres", Name: Localized{"fr": "Filtres"}, ImageURL: img}
	}
	for name, tc := range map[string]struct {
		img *string
		ok  bool
		out string
	}{
		"absente":           {nil, true, ""},
		"vide = absente":    {ptr("  "), true, ""},
		"https":             {ptr("https://cdn.exemple.com/a.png"), true, "https://cdn.exemple.com/a.png"},
		"chemin du site":    {ptr(" /img/a.png "), true, "/img/a.png"},
		"protocole relatif": {ptr("//evil.com/a.png"), false, ""},
		"javascript":        {ptr("javascript:alert(1)"), false, ""},
	} {
		c := base(tc.img)
		err := c.Validate()
		if (err == nil) != tc.ok {
			t.Fatalf("%s : erreur = %v", name, err)
		}
		if tc.ok && ((tc.out == "") != (c.ImageURL == nil) || (c.ImageURL != nil && *c.ImageURL != tc.out)) {
			t.Fatalf("%s : image = %v", name, c.ImageURL)
		}
	}
}
