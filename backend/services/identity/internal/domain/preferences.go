package domain

import (
	"fmt"
	"regexp"

	"github.com/theb20/FILTER/backend/pkg/apperr"
)

// PrincipalKind distingue les deux mondes de comptes (jamais mélangés) : le personnel et les clients.
type PrincipalKind string

const (
	PrincipalStaff    PrincipalKind = "staff"
	PrincipalCustomer PrincipalKind = "customer"
)

// Bornes du contrat (components.schemas.GuidePreferences).
const (
	MaxGuideSeen   = 200
	GuideSeenIDMax = 64
)

var guideSeenID = regexp.MustCompile(`^[a-z0-9-]+$`)

// GuidePreferences : préférences du guide visuel (visites guidées).
type GuidePreferences struct {
	Enabled    bool
	Animations string // "auto" | "off"
	Seen       []string
}

// Preferences : préférences d'interface d'un compte, synchronisées entre appareils.
// Chaque section a ses défauts : un compte qui n'a jamais rien enregistré les reçoit.
type Preferences struct {
	Guide GuidePreferences
}

func DefaultPreferences() Preferences {
	return Preferences{Guide: GuidePreferences{Enabled: true, Animations: "auto", Seen: []string{}}}
}

// Clone renvoie une copie indépendante (la liste `Seen` n'est jamais partagée entre appelants).
func (p Preferences) Clone() Preferences {
	p.Guide.Seen = append([]string{}, p.Guide.Seen...)
	return p
}

// Validate applique les bornes du contrat ; chaque champ fautif est signalé.
func (p Preferences) Validate() error {
	var f []apperr.Field
	add := func(ptr, msg string) {
		f = append(f, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
	}
	if p.Guide.Animations != "auto" && p.Guide.Animations != "off" {
		add("/guide/animations", `doit valoir « auto » ou « off »`)
	}
	if len(p.Guide.Seen) > MaxGuideSeen {
		add("/guide/seen", fmt.Sprintf("au plus %d visites", MaxGuideSeen))
	}
	seen := make(map[string]bool, len(p.Guide.Seen))
	for i, id := range p.Guide.Seen {
		ptr := fmt.Sprintf("/guide/seen/%d", i)
		switch {
		case id == "" || len(id) > GuideSeenIDMax || !guideSeenID.MatchString(id):
			add(ptr, "identifiant invalide (lettres minuscules, chiffres et tirets, 64 caractères au plus)")
		case seen[id]:
			add(ptr, "identifiant en double")
		}
		seen[id] = true
	}
	if len(f) > 0 {
		return apperr.Validation("préférences invalides", f...)
	}
	return nil
}
