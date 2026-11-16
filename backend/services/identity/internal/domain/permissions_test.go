package domain

import (
	"os"
	"regexp"
	"slices"
	"sort"
	"testing"
)

// Le catalogue de permissions du code et celui du contrat doivent être IDENTIQUES.
func TestPermissionsSontCelleDuContrat(t *testing.T) {
	raw, err := os.ReadFile("../../../../api/openapi.yaml")
	if err != nil {
		t.Skipf("contrat introuvable : %v", err)
	}
	set := map[string]bool{}
	for _, m := range regexp.MustCompile(`x-required-permission: ([a-z]+:[a-z]+)`).FindAllStringSubmatch(string(raw), -1) {
		set[m[1]] = true
	}
	var want, got []string
	for c := range set {
		want = append(want, c)
	}
	for _, p := range Permissions {
		got = append(got, p.Code)
	}
	sort.Strings(want)
	if !slices.Equal(want, got) {
		t.Fatalf("catalogue ≠ contrat\n contrat : %v\n code    : %v", want, got)
	}
}

func TestHasPermission(t *testing.T) {
	if !HasPermission([]string{"*"}, "catalog:write") || !HasPermission([]string{"a:read", "b:write"}, "b:write") || HasPermission([]string{"a:read"}, "a:write") || HasPermission(nil, "a:read") {
		t.Fatal("HasPermission incorrect")
	}
	if !IsPermission("*") || !IsPermission("orders:refund") || IsPermission("orders:fly") {
		t.Fatal("IsPermission incorrect")
	}
}
