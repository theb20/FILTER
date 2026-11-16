package domain

// Catalogue des permissions : une entrée par valeur de `x-required-permission` du contrat (openapi.yaml).
// Le test TestPermissionsSontCelleDuContrat garantit que ce catalogue et le contrat ne divergent jamais.
// Les libellés sont dérivés du code (domaine + action) : aucune règle d'accès n'est inventée ici.

// Permission est une capacité attribuable à un rôle.
type Permission struct{ Code, Description, Group string }

// Permissions est le catalogue complet, trié par code.
var Permissions = []Permission{
	{Code: "admin:read", Group: "Administration", Description: "Consulter — administration"},
	{Code: "admin:write", Group: "Administration", Description: "Créer et modifier — administration"},
	{Code: "analytics:read", Group: "Tableau de bord", Description: "Consulter — tableau de bord"},
	{Code: "audit:read", Group: "Journal d'audit", Description: "Consulter — journal d'audit"},
	{Code: "catalog:publish", Group: "Catalogue", Description: "Publier — catalogue"},
	{Code: "catalog:read", Group: "Catalogue", Description: "Consulter — catalogue"},
	{Code: "catalog:write", Group: "Catalogue", Description: "Créer et modifier — catalogue"},
	{Code: "companies:approve", Group: "Entreprises clientes", Description: "Approuver ou rejeter — entreprises clientes"},
	{Code: "content:publish", Group: "Contenu", Description: "Publier — contenu"},
	{Code: "content:read", Group: "Contenu", Description: "Consulter — contenu"},
	{Code: "content:write", Group: "Contenu", Description: "Créer et modifier — contenu"},
	{Code: "customers:privacy", Group: "Clients", Description: "Traiter les demandes RGPD — clients"},
	{Code: "customers:read", Group: "Clients", Description: "Consulter — clients"},
	{Code: "customers:write", Group: "Clients", Description: "Créer et modifier — clients"},
	{Code: "engine:budget", Group: "Moteur de recherche", Description: "Gérer les budgets et plafonds — moteur de recherche"},
	{Code: "engine:export", Group: "Moteur de recherche", Description: "Exporter — moteur de recherche"},
	{Code: "engine:read", Group: "Moteur de recherche", Description: "Consulter — moteur de recherche"},
	{Code: "engine:write", Group: "Moteur de recherche", Description: "Créer et modifier — moteur de recherche"},
	{Code: "inventory:read", Group: "Stock", Description: "Consulter — stock"},
	{Code: "inventory:write", Group: "Stock", Description: "Créer et modifier — stock"},
	{Code: "invoices:read", Group: "Factures", Description: "Consulter — factures"},
	{Code: "notifications:read", Group: "Notifications", Description: "Consulter — notifications"},
	{Code: "notifications:write", Group: "Notifications", Description: "Créer et modifier — notifications"},
	{Code: "orders:read", Group: "Commandes", Description: "Consulter — commandes"},
	{Code: "orders:refund", Group: "Commandes", Description: "Rembourser — commandes"},
	{Code: "orders:write", Group: "Commandes", Description: "Créer et modifier — commandes"},
	{Code: "payments:read", Group: "Paiements", Description: "Consulter — paiements"},
	{Code: "payments:reconcile", Group: "Paiements", Description: "Rapprocher manuellement — paiements"},
	{Code: "payments:settings", Group: "Paiements", Description: "Gérer les réglages — paiements"},
	{Code: "pricing:read", Group: "Prix et promotions", Description: "Consulter — prix et promotions"},
	{Code: "pricing:write", Group: "Prix et promotions", Description: "Créer et modifier — prix et promotions"},
	{Code: "quotes:convert", Group: "Devis", Description: "Convertir en commande — devis"},
	{Code: "quotes:read", Group: "Devis", Description: "Consulter — devis"},
	{Code: "quotes:write", Group: "Devis", Description: "Créer et modifier — devis"},
	{Code: "reviews:moderate", Group: "Avis", Description: "Modérer — avis"},
	{Code: "reviews:read", Group: "Avis", Description: "Consulter — avis"},
	{Code: "shipping:read", Group: "Livraison", Description: "Consulter — livraison"},
	{Code: "shipping:write", Group: "Livraison", Description: "Créer et modifier — livraison"},
	{Code: "support:read", Group: "Support", Description: "Consulter — support"},
	{Code: "support:write", Group: "Support", Description: "Créer et modifier — support"},
}

// IsPermission indique si le code appartient au catalogue (ou est le joker réservé au rôle système).
func IsPermission(code string) bool {
	if code == AllPermissions {
		return true
	}
	for _, p := range Permissions {
		if p.Code == code {
			return true
		}
	}
	return false
}

// HasPermission : le joker `*` donne toutes les permissions.
func HasPermission(granted []string, required string) bool {
	for _, g := range granted {
		if g == AllPermissions || g == required {
			return true
		}
	}
	return false
}
