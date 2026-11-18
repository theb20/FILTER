// Package domain porte les règles du service notification : QUI reçoit QUOI, dans QUELLE langue, et à quelles
// conditions. Rien ici ne parle de Resend, de PostgreSQL ni de gRPC : ce sont des décisions, pas de la plomberie.
package domain

import "sort"

// Audience dit COMMENT trouver le ou les destinataires d'un message. C'est le point qui distingue un e-mail client
// d'une alerte interne : l'adresse d'un client voyage dans l'événement, celle du personnel se DÉDUIT de ses rôles.
type Audience int

const (
	// ToEventField : l'adresse est portée par l'événement lui-même (champ nommé par Rule.Field).
	// Aucun appel à un autre service : c'est le cas le plus sûr et le plus rapide.
	ToEventField Audience = iota
	// ToUser : l'événement ne porte qu'un `user_id` ; l'adresse et la langue viennent du compte (service identity).
	ToUser
	// ToOrderCustomer : l'événement ne porte qu'un `order_id` ; le client se déduit de la commande (service order).
	ToOrderCustomer
	// ToQuoteContact : l'événement ne porte qu'un identifiant de devis ou de demande (service quote).
	ToQuoteContact
	// ToStaffWithPermission : TOUT le personnel actif détenant la permission Rule.Permission, super-admins compris
	// (permission « * »). Suit les rôles définis au backoffice : aucune adresse à maintenir à la main.
	ToStaffWithPermission
)

func (a Audience) String() string {
	switch a {
	case ToEventField:
		return "adresse portée par l'événement"
	case ToUser:
		return "titulaire du compte (identity)"
	case ToOrderCustomer:
		return "client de la commande (order)"
	case ToQuoteContact:
		return "contact du devis (quote)"
	case ToStaffWithPermission:
		return "personnel détenant la permission"
	}
	return "inconnue"
}

// Status dit si la règle peut se déclencher AUJOURD'HUI. Une règle « planifiée » est écrite, testée et inerte : le
// service qui doit publier son événement n'existe pas encore (phases 4 et 5 du phasage). Le nom de sujet d'un
// événement planifié reste à confirmer le jour où son service est construit : il suit la convention
// `<service>.<message>.v1` mais n'est pas encore figé par du code qui publie.
type Status int

const (
	// Live : l'événement est publié par du code existant ; l'e-mail part réellement.
	Live Status = iota
	// Planned : l'événement est déclaré au contrat (events.proto) mais aucun service ne le publie encore.
	Planned
)

// Rule est UNE règle d'envoi : un événement produit un message, pour une audience, à partir d'un gabarit.
// Un même événement peut porter PLUSIEURS règles (ex. une demande de devis : accusé au client ET alerte aux
// commerciaux ; un changement d'adresse : vérification à la nouvelle adresse ET avertissement à l'ancienne).
type Rule struct {
	Event       string   // sujet de l'événement (voir proto/events/v1)
	TemplateKey string   // clé du gabarit en base (table `templates`), décliné par langue
	To          Audience //
	Field       string   // ToEventField : nom du champ de l'événement portant l'adresse
	Permission  string   // ToStaffWithPermission : permission requise (ex. « companies:approve »)
	// Mandatory : message TRANSACTIONNEL, réponse directe à une action de la personne ou à la sécurité de son compte.
	// Il passe outre une DÉSINSCRIPTION — on ne prive personne d'un code de réinitialisation, d'une facture ou d'une
	// alerte de sécurité. Il ne passe JAMAIS outre un rebond définitif ni une plainte pour pourriel : une adresse morte
	// ou qui nous a signalés reste bloquée quoi qu'il arrive, sous peine de perdre la réputation du domaine.
	// Tout message de cette table est transactionnel par construction : il naît d'un événement, jamais d'une campagne.
	// Les envois de newsletter, eux, ne passent pas par ici et respectent la désinscription.
	Mandatory bool
	Status    Status
	Why       string // la raison d'être de la règle, en une ligne : sert la relecture et le backoffice
}

// Sujets des événements. Les huit premiers sont publiés par identity aujourd'hui ; les suivants sont déclarés au
// contrat (proto/events/v1/events.proto) et attendent leur service producteur.
const (
	// ── identity (publiés) ───────────────────────────────────────────────────
	EvEmailVerification   = "identity.email_verification_requested.v1"
	EvPasswordReset       = "identity.password_reset_requested.v1"
	EvRegistrationExists  = "identity.registration_attempt_on_existing_account.v1"
	EvSuspiciousLogin     = "identity.suspicious_login_alert_requested.v1"
	EvEmailChangeRequest  = "identity.email_change_requested.v1"
	EvEmailChangeDone     = "identity.email_change_confirmed.v1"
	EvStaffInvitation     = "identity.staff_invitation_requested.v1"
	EvStaffTwoFactorReset = "identity.staff_two_factor_reset.v1"

	// ── identity (déclarés, pas encore publiés) ──────────────────────────────
	EvUserRegistered = "identity.user_registered.v1"
	EvCompanyDecided = "identity.company_decided.v1"
	EvDataExport     = "identity.user_data_export_requested.v1"
	EvErasureDone    = "identity.user_erasure_completed.v1"

	// ── order / payment (service non construit) ──────────────────────────────
	EvOrderPlaced         = "order.placed.v1"
	EvOrderPaid           = "order.paid.v1"
	EvOrderStatusChanged  = "order.status_changed.v1"
	EvOrderCancelled      = "order.cancelled.v1"
	EvOrderPaymentExpired = "order.payment_expired.v1"
	EvPaymentFailed       = "payment.failed.v1"
	EvRefundSucceeded     = "payment.refund_succeeded.v1"
	EvRefundFailed        = "payment.refund_failed.v1"

	// ── quote (service non construit) ────────────────────────────────────────
	EvQuoteRequested = "quote.requested.v1"
	EvQuoteSent      = "quote.sent.v1"
	EvQuoteAccepted  = "quote.accepted.v1"

	// ── support / content (services non construits) ──────────────────────────
	EvContactReceived = "support.contact_received.v1"
	EvReviewSubmitted = "content.review_submitted.v1"

	// ── notification (produit par ce service) ────────────────────────────────
	EvNewsletterConfirm = "notification.newsletter_confirmation_requested.v1"
)

// Rules est LA TABLE DES RÈGLES : la seule source de vérité sur qui reçoit quoi.
//
// Lecture : un événement arrive, on prend toutes les règles qui le citent, on résout l'audience de chacune, on rend le
// gabarit `TemplateKey` dans la langue du destinataire, on envoie. Ajouter un e-mail = ajouter une ligne ici et un
// gabarit en base. Rien d'autre dans le code n'a le droit de décider d'un envoi.
var Rules = []Rule{

	// ── Compte client : sécurité et accès ────────────────────────────────────
	{
		Event: EvEmailVerification, TemplateKey: "account.email_verification",
		To: ToEventField, Field: "email", Mandatory: true, Status: Live,
		Why: "Active le compte : sans ce lien, l'inscription reste inachevée.",
	},
	{
		Event: EvPasswordReset, TemplateKey: "account.password_reset",
		To: ToEventField, Field: "email", Mandatory: true, Status: Live,
		Why: "Porte le code à usage unique : seule voie de récupération du compte.",
	},
	{
		Event: EvRegistrationExists, TemplateKey: "account.registration_on_existing",
		To: ToEventField, Field: "email", Mandatory: true, Status: Live,
		Why: "La réponse HTTP reste volontairement uniforme ; seul le titulaire apprend que son compte existe.",
	},
	{
		Event: EvSuspiciousLogin, TemplateKey: "account.suspicious_login",
		To: ToEventField, Field: "email", Mandatory: true, Status: Live,
		Why: "Alerte de sécurité : le titulaire doit pouvoir réagir à une série d'échecs.",
	},
	{
		Event: EvEmailChangeRequest, TemplateKey: "account.email_change_verify",
		To: ToEventField, Field: "new_email", Mandatory: true, Status: Live,
		Why: "Vérifie que la NOUVELLE adresse appartient bien au titulaire avant la bascule.",
	},
	{
		Event: EvEmailChangeRequest, TemplateKey: "account.email_change_alert",
		To: ToEventField, Field: "old_email", Mandatory: true, Status: Live,
		Why: "Avertit l'ANCIENNE adresse : une prise de contrôle doit être visible de son titulaire légitime.",
	},
	{
		Event: EvEmailChangeDone, TemplateKey: "account.email_change_done",
		To: ToEventField, Field: "old_email", Mandatory: true, Status: Live,
		Why: "Confirme la bascule à l'adresse qui perd l'accès, avec le recours à suivre.",
	},

	// ── Compte client : entreprise et données personnelles ───────────────────
	{
		Event: EvCompanyDecided, TemplateKey: "account.company_decided",
		To: ToUser, Mandatory: true, Status: Planned,
		Why: "La décision (validée ou refusée) conditionne l'accès aux prix professionnels ; le gabarit traite les deux cas.",
	},
	{
		Event: EvDataExport, TemplateKey: "account.data_export_ready",
		To: ToUser, Mandatory: true, Status: Planned,
		Why: "RGPD : l'archive est prête, le lien est à durée limitée.",
	},
	{
		Event: EvErasureDone, TemplateKey: "account.erasure_done",
		To: ToUser, Mandatory: true, Status: Planned,
		Why: "RGPD : preuve de l'effacement, dernier message envoyé à cette adresse.",
	},

	// ── Commandes et paiements ───────────────────────────────────────────────
	{
		Event: EvOrderPlaced, TemplateKey: "order.confirmation",
		To: ToEventField, Field: "customer_email", Mandatory: true, Status: Planned,
		Why: "Accusé de commande : récapitulatif et référence de suivi.",
	},
	{
		Event: EvOrderPaid, TemplateKey: "order.paid",
		To: ToOrderCustomer, Mandatory: true, Status: Planned,
		Why: "Confirme l'encaissement et déclenche la préparation.",
	},
	{
		Event: EvOrderStatusChanged, TemplateKey: "order.status_changed",
		To: ToOrderCustomer, Mandatory: true, Status: Planned,
		Why: "Expédition, livraison, retour : chaque étape visible du client.",
	},
	{
		Event: EvOrderCancelled, TemplateKey: "order.cancelled",
		To: ToOrderCustomer, Mandatory: true, Status: Planned,
		Why: "Annulation : motif et remboursement éventuel.",
	},
	{
		Event: EvOrderPaymentExpired, TemplateKey: "order.payment_expired",
		To: ToOrderCustomer, Mandatory: true, Status: Planned,
		Why: "Le prépaiement n'est pas arrivé à temps : la commande est annulée, le stock libéré.",
	},
	{
		Event: EvPaymentFailed, TemplateKey: "payment.failed",
		To: ToOrderCustomer, Mandatory: true, Status: Planned,
		Why: "Le paiement a échoué : le client doit pouvoir réessayer avant expiration.",
	},
	{
		Event: EvRefundSucceeded, TemplateKey: "payment.refund_succeeded",
		To: ToOrderCustomer, Mandatory: true, Status: Planned,
		Why: "Remboursement effectué : montant, délai bancaire et avoir.",
	},
	{
		Event: EvRefundFailed, TemplateKey: "staff.refund_failed",
		To: ToStaffWithPermission, Permission: "payments:reconcile", Mandatory: true, Status: Planned,
		Why: "Échec de remboursement : demande une reprise manuelle, le client n'est pas alerté.",
	},

	// ── Devis ────────────────────────────────────────────────────────────────
	{
		Event: EvQuoteRequested, TemplateKey: "quote.acknowledgement",
		To: ToEventField, Field: "contact_email", Mandatory: true, Status: Planned,
		Why: "Accuse réception de la demande et annonce le délai de réponse.",
	},
	{
		Event: EvQuoteRequested, TemplateKey: "staff.quote_requested",
		To: ToStaffWithPermission, Permission: "quotes:write", Mandatory: true, Status: Planned,
		Why: "Alerte les commerciaux : une demande sans réponse est une vente perdue.",
	},
	{
		Event: EvQuoteSent, TemplateKey: "quote.sent",
		To: ToEventField, Field: "contact_email", Mandatory: true, Status: Planned,
		Why: "Porte le devis et sa date de validité.",
	},
	{
		Event: EvQuoteAccepted, TemplateKey: "staff.quote_accepted",
		To: ToStaffWithPermission, Permission: "quotes:convert", Mandatory: true, Status: Planned,
		Why: "Le client a accepté : la conversion en commande doit suivre sans délai.",
	},

	// ── Support et modération ────────────────────────────────────────────────
	{
		Event: EvContactReceived, TemplateKey: "contact.acknowledgement",
		To: ToEventField, Field: "email", Mandatory: true, Status: Planned,
		Why: "Accuse réception du message avec sa référence.",
	},
	{
		Event: EvContactReceived, TemplateKey: "staff.contact_received",
		To: ToStaffWithPermission, Permission: "support:write", Mandatory: true, Status: Planned,
		Why: "Met le message dans les mains du support sans attendre une consultation du backoffice.",
	},
	{
		Event: EvReviewSubmitted, TemplateKey: "staff.review_submitted",
		To: ToStaffWithPermission, Permission: "reviews:moderate", Mandatory: true, Status: Planned,
		Why: "Un avis attend modération avant publication.",
	},

	// ── Validation d'entreprise (interne) ────────────────────────────────────
	{
		Event: EvUserRegistered, TemplateKey: "staff.company_pending",
		To: ToStaffWithPermission, Permission: "companies:approve", Mandatory: true, Status: Planned,
		Why: "Une inscription d'entreprise attend une décision ; le client reste bloqué tant qu'elle n'est pas prise.",
	},

	// ── Personnel : accès au backoffice ──────────────────────────────────────
	{
		Event: EvStaffInvitation, TemplateKey: "staff.invitation",
		To: ToEventField, Field: "email", Mandatory: true, Status: Live,
		Why: "Seule voie d'entrée au backoffice : lien personnel à usage unique.",
	},
	{
		Event: EvStaffTwoFactorReset, TemplateKey: "staff.two_factor_reset",
		To: ToEventField, Field: "email", Mandatory: true, Status: Live,
		Why: "La personne doit savoir que sa double authentification a été remise à zéro, et par qui.",
	},

	// ── Newsletter (seul message NON obligatoire) ────────────────────────────
	{
		Event: EvNewsletterConfirm, TemplateKey: "newsletter.confirmation",
		To: ToEventField, Field: "email", Mandatory: true, Status: Planned,
		Why: "Double opt-in : sans cette confirmation, aucune adresse n'entre dans la liste. Obligatoire car c'est la " +
			"réponse directe à une action de la personne — les envois de la newsletter, eux, respectent la liste de suppression.",
	},
}

// RulesFor renvoie les règles déclenchées par un événement, dans l'ordre de la table.
func RulesFor(event string) []Rule {
	var out []Rule
	for _, r := range Rules {
		if r.Event == event {
			out = append(out, r)
		}
	}
	return out
}

// LiveEvents renvoie les sujets dont au moins une règle peut se déclencher aujourd'hui : c'est à eux que le service
// s'abonne. S'abonner aux sujets planifiés ne coûterait rien, mais masquerait ce qui fonctionne vraiment.
func LiveEvents() []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range Rules {
		if r.Status == Live && !seen[r.Event] {
			seen[r.Event] = true
			out = append(out, r.Event)
		}
	}
	sort.Strings(out)
	return out
}

// TemplateKeys renvoie toutes les clés de gabarit citées par la table, triées et dédoublonnées.
// La migration d'amorçage et les tests s'en servent pour garantir qu'aucune règle ne pointe vers un gabarit absent.
func TemplateKeys() []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range Rules {
		if !seen[r.TemplateKey] {
			seen[r.TemplateKey] = true
			out = append(out, r.TemplateKey)
		}
	}
	sort.Strings(out)
	return out
}

// StaffPermissions renvoie les permissions citées par les règles internes : utile au backoffice pour avertir qu'un
// rôle supprimé laisserait une alerte sans destinataire.
func StaffPermissions() []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range Rules {
		if r.To == ToStaffWithPermission && !seen[r.Permission] {
			seen[r.Permission] = true
			out = append(out, r.Permission)
		}
	}
	sort.Strings(out)
	return out
}
