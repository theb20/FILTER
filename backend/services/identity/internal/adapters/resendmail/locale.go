package resendmail

import (
	"fmt"
	"strings"
)

// Langues servies par les e-mails. Ce sont celles que le service pose dans les événements (voir `domain.User.Locale`
// et la normalisation OAuth) et celles de la vitrine. Toute autre valeur retombe sur le français.
const defaultLocale = "fr"

// mailTexts regroupe les libellés d'UN e-mail dans UNE langue.
// Deux marqueurs y sont remplacés au rendu : {brand} (nom de la marque) et {validity} (durée de validité du code).
type mailTexts struct {
	Subject    string
	Preheader  string
	Title      string
	Paragraphs []string
	CTALabel   string
	Note       string
}

// langPack réunit les libellés communs à tous les e-mails d'une langue, plus un jeu de textes par type d'événement.
type langPack struct {
	Code          string // attribut lang= du document
	DateFormat    string // gabarit de date Go (référence 2006-01-02 15:04)
	FallbackLabel string
	FooterLine    string
	CodeLabel     string // préfixe du code dans la version texte
	ValidityFmt   string // une seule substitution : le nombre de minutes
	RowFailures   string
	RowLastIP     string
	RowDate       string
	RowNewEmail   string
	RowDevice     string
	RowPlace      string
	Mails         map[string]mailTexts // clé : type d'événement
}

// packFor choisit la langue à partir du champ `locale` de l'événement (la première valeur non vide l'emporte).
// Une langue inconnue, vide ou mal formée retombe sur le français : un e-mail en français vaut mieux qu'aucun e-mail.
func packFor(locales ...string) langPack {
	for _, l := range locales {
		l = strings.ToLower(strings.TrimSpace(l))
		if len(l) >= 2 {
			if p, ok := packs[l[:2]]; ok {
				return p
			}
		}
	}
	return packs[defaultLocale]
}

// validity rend « Ce code est valable N minutes. » dans la langue du destinataire.
func (p langPack) validity(minutes int) string { return fmt.Sprintf(p.ValidityFmt, minutes) }

var packs = map[string]langPack{

	// ── Français ──────────────────────────────────────────────────────────────
	"fr": {
		Code:          "fr",
		DateFormat:    "02/01/2006 à 15:04 UTC",
		FallbackLabel: "Le bouton ne fonctionne pas ? Copiez ce lien dans votre navigateur :",
		FooterLine:    "Message automatique : merci de ne pas y répondre.",
		CodeLabel:     "Code : ",
		ValidityFmt:   "Ce code est valable %d minutes.",
		RowFailures:   "Tentatives échouées",
		RowLastIP:     "Dernière adresse IP",
		RowDate:       "Date",
		RowNewEmail:   "Nouvelle adresse",
		RowDevice:     "Appareil",
		RowPlace:      "Lieu approximatif",
		Mails: map[string]mailTexts{
			evVerification: {
				Subject:    "Confirmez votre adresse e-mail",
				Preheader:  "Une dernière étape pour activer votre compte.",
				Title:      "Confirmez votre adresse e-mail",
				Paragraphs: []string{"Bienvenue ! Confirmez cette adresse pour activer votre compte et suivre vos devis et vos commandes."},
				CTALabel:   "Confirmer mon adresse",
				Note:       "Vous n'êtes pas à l'origine de cette inscription ? Ignorez ce message, aucun compte ne sera activé.",
			},
			evPasswordReset: {
				Subject:    "Votre code de réinitialisation",
				Preheader:  "{validity}",
				Title:      "Réinitialisation du mot de passe",
				Paragraphs: []string{"Saisissez ce code sur la page de réinitialisation pour choisir un nouveau mot de passe."},
				CTALabel:   "Ouvrir la page",
				Note:       "{validity} Si vous n'avez rien demandé, ignorez ce message : votre mot de passe reste inchangé.",
			},
			evStaffInvitation: {
				Subject:   "Votre accès au backoffice {brand}",
				Preheader: "Créez votre mot de passe et activez la double authentification.",
				Title:     "Vous avez été invité au backoffice",
				Paragraphs: []string{
					"Un administrateur vous a ouvert un accès au backoffice {brand}.",
					"Suivez le lien ci-dessous pour choisir votre mot de passe, puis enregistrer votre double authentification (obligatoire).",
				},
				CTALabel: "Activer mon accès",
				Note:     "Ce lien est personnel et à usage unique. Ne le transmettez à personne.",
			},
			evTwoFactorReset: {
				Subject:   "Votre double authentification a été réinitialisée",
				Preheader: "Un nouvel enregistrement vous sera demandé à la prochaine connexion.",
				Title:     "Double authentification réinitialisée",
				Paragraphs: []string{
					"Un administrateur a réinitialisé la double authentification de votre compte {brand}.",
					"À votre prochaine connexion, le backoffice vous demandera d'enregistrer à nouveau votre application d'authentification.",
				},
				CTALabel: "Se connecter au backoffice",
				Note:     "Vous n'avez rien demandé ? Prévenez immédiatement votre administrateur.",
			},
			evExistingAccount: {
				Subject:   "Un compte existe déjà avec cette adresse",
				Preheader: "Aucune action n'est nécessaire de votre part.",
				Title:     "Un compte existe déjà avec cette adresse",
				Paragraphs: []string{
					"Quelqu'un vient d'essayer de créer un compte avec cette adresse. Comme elle est déjà utilisée, aucun nouveau compte n'a été créé.",
					"Si c'était vous : connectez-vous normalement, ou réinitialisez votre mot de passe si vous l'avez oublié.",
				},
				CTALabel: "Me connecter",
				Note:     "Si ce n'était pas vous, aucune action n'est nécessaire : votre compte n'a pas été modifié.",
			},
			evEmailChangeReq: {
				Subject:    "Confirmez votre nouvelle adresse e-mail",
				Preheader:  "Le changement n'aura lieu qu'après votre confirmation.",
				Title:      "Confirmez votre nouvelle adresse e-mail",
				Paragraphs: []string{"Vous avez demandé à utiliser cette adresse pour votre compte {brand}. Confirmez-la pour finaliser le changement."},
				CTALabel:   "Confirmer ma nouvelle adresse",
				Note:       "Vous n'êtes pas à l'origine de cette demande ? Ignorez ce message : rien ne sera modifié.",
			},
			evEmailChangeDone: {
				Subject:   "L'adresse e-mail de votre compte a été modifiée",
				Preheader: "Si ce n'était pas vous, agissez maintenant.",
				Title:     "L'adresse e-mail de votre compte a changé",
				Paragraphs: []string{
					"L'adresse e-mail de votre compte {brand} vient d'être remplacée. Vous ne recevrez plus de messages à cette adresse.",
					"Pour votre sécurité, tous vos appareils ont été déconnectés.",
				},
				CTALabel: "Sécuriser mon compte",
				Note:     "Vous n'êtes pas à l'origine de ce changement ? Contactez-nous sans attendre.",
			},
			evNewDeviceLogin: {
				Subject:   "Nouvelle connexion à votre compte",
				Preheader: "Un nouvel appareil vient de se connecter.",
				Title:     "Nouvelle connexion à votre compte",
				Paragraphs: []string{
					"Votre compte {brand} vient d'être ouvert depuis un appareil que nous ne connaissions pas. Le lieu est approximatif : il est déduit de l'adresse IP (souvent la ville de l'opérateur).",
					"C'était vous ? Vous n'avez rien à faire. Ce n'était pas vous ? Cliquez ci-dessous : tous les appareils seront déconnectés, puis choisissez un nouveau mot de passe.",
				},
				CTALabel: "Ce n'était pas moi",
				Note:     "Ce lien est valable 7 jours et ne sert qu'une fois. Nous ne vous demanderons jamais votre mot de passe par e-mail.",
			},
			evSuspiciousLogin: {
				Subject:   "Tentatives de connexion échouées sur votre compte",
				Preheader: "Vérifiez qu'il s'agit bien de vous.",
				Title:     "Tentatives de connexion échouées",
				Paragraphs: []string{
					"Plusieurs tentatives de connexion à votre compte ont échoué. Le compte n'a pas été ouvert.",
					"S'il s'agissait de vous, vous pouvez réinitialiser votre mot de passe. Sinon, nous vous conseillons d'en choisir un nouveau, unique à ce site.",
				},
				CTALabel: "Changer mon mot de passe",
				Note:     "Nous ne vous demanderons jamais votre mot de passe par e-mail.",
			},
		},
	},

	// ── English ───────────────────────────────────────────────────────────────
	"en": {
		Code:          "en",
		DateFormat:    "Jan 2, 2006 at 3:04 PM UTC",
		FallbackLabel: "Button not working? Copy this link into your browser:",
		FooterLine:    "Automated message — please do not reply.",
		CodeLabel:     "Code: ",
		ValidityFmt:   "This code is valid for %d minutes.",
		RowFailures:   "Failed attempts",
		RowLastIP:     "Last IP address",
		RowDate:       "Date",
		RowNewEmail:   "New address",
		RowDevice:     "Device",
		RowPlace:      "Approximate location",
		Mails: map[string]mailTexts{
			evVerification: {
				Subject:    "Confirm your email address",
				Preheader:  "One last step to activate your account.",
				Title:      "Confirm your email address",
				Paragraphs: []string{"Welcome! Confirm this address to activate your account and keep track of your quotes and orders."},
				CTALabel:   "Confirm my address",
				Note:       "Didn't sign up? Ignore this message — no account will be activated.",
			},
			evPasswordReset: {
				Subject:    "Your password reset code",
				Preheader:  "{validity}",
				Title:      "Password reset",
				Paragraphs: []string{"Enter this code on the reset page to choose a new password."},
				CTALabel:   "Open the page",
				Note:       "{validity} If you didn't request this, ignore this message: your password stays unchanged.",
			},
			evStaffInvitation: {
				Subject:   "Your {brand} backoffice access",
				Preheader: "Create your password and set up two-factor authentication.",
				Title:     "You've been invited to the backoffice",
				Paragraphs: []string{
					"An administrator has opened a backoffice account for you on {brand}.",
					"Follow the link below to choose your password, then register your two-factor authentication (required).",
				},
				CTALabel: "Activate my access",
				Note:     "This link is personal and single-use. Do not share it with anyone.",
			},
			evTwoFactorReset: {
				Subject:   "Your two-factor authentication has been reset",
				Preheader: "You'll be asked to set it up again at your next sign-in.",
				Title:     "Two-factor authentication reset",
				Paragraphs: []string{
					"An administrator has reset the two-factor authentication on your {brand} account.",
					"At your next sign-in, the backoffice will ask you to register your authenticator app again.",
				},
				CTALabel: "Sign in to the backoffice",
				Note:     "Weren't expecting this? Tell your administrator immediately.",
			},
			evExistingAccount: {
				Subject:   "An account already exists with this address",
				Preheader: "No action is needed on your side.",
				Title:     "An account already exists with this address",
				Paragraphs: []string{
					"Someone just tried to create an account with this address. As it is already in use, no new account was created.",
					"If it was you: sign in as usual, or reset your password if you've forgotten it.",
				},
				CTALabel: "Sign in",
				Note:     "If it wasn't you, there is nothing to do: your account has not been changed.",
			},
			evEmailChangeReq: {
				Subject:    "Confirm your new email address",
				Preheader:  "The change only happens once you confirm.",
				Title:      "Confirm your new email address",
				Paragraphs: []string{"You asked to use this address for your {brand} account. Confirm it to complete the change."},
				CTALabel:   "Confirm my new address",
				Note:       "Didn't ask for this? Ignore this message: nothing will change.",
			},
			evEmailChangeDone: {
				Subject:   "The email address on your account was changed",
				Preheader: "If this wasn't you, act now.",
				Title:     "Your account's email address changed",
				Paragraphs: []string{
					"The email address of your {brand} account has just been replaced. You will no longer receive messages at this address.",
					"For your security, all your devices have been signed out.",
				},
				CTALabel: "Secure my account",
				Note:     "Didn't make this change? Contact us right away.",
			},
			evNewDeviceLogin: {
				Subject:   "New sign-in to your account",
				Preheader: "A new device just signed in.",
				Title:     "New sign-in to your account",
				Paragraphs: []string{
					"Your {brand} account was just opened from a device we did not recognise. The location is approximate: it is derived from the IP address (often the network operator's city).",
					"Was it you? There is nothing to do. Wasn't it you? Click below: all devices will be signed out, then choose a new password.",
				},
				CTALabel: "This wasn't me",
				Note:     "This link is valid for 7 days and works only once. We will never ask you for your password by email.",
			},
			evSuspiciousLogin: {
				Subject:   "Failed sign-in attempts on your account",
				Preheader: "Check that this was you.",
				Title:     "Failed sign-in attempts",
				Paragraphs: []string{
					"Several sign-in attempts on your account have failed. The account was not opened.",
					"If this was you, you can reset your password. If not, we recommend choosing a new one, unique to this site.",
				},
				CTALabel: "Change my password",
				Note:     "We will never ask you for your password by email.",
			},
		},
	},

	// ── Español ───────────────────────────────────────────────────────────────
	"es": {
		Code:          "es",
		DateFormat:    "02/01/2006 a las 15:04 UTC",
		FallbackLabel: "¿El botón no funciona? Copie este enlace en su navegador:",
		FooterLine:    "Mensaje automático: por favor, no responda.",
		CodeLabel:     "Código: ",
		ValidityFmt:   "Este código es válido durante %d minutos.",
		RowFailures:   "Intentos fallidos",
		RowLastIP:     "Última dirección IP",
		RowDate:       "Fecha",
		RowNewEmail:   "Nueva dirección",
		RowDevice:     "Dispositivo",
		RowPlace:      "Ubicación aproximada",
		Mails: map[string]mailTexts{
			evVerification: {
				Subject:    "Confirme su dirección de correo",
				Preheader:  "Un último paso para activar su cuenta.",
				Title:      "Confirme su dirección de correo",
				Paragraphs: []string{"¡Bienvenido! Confirme esta dirección para activar su cuenta y hacer seguimiento de sus presupuestos y pedidos."},
				CTALabel:   "Confirmar mi dirección",
				Note:       "¿No se ha registrado usted? Ignore este mensaje: no se activará ninguna cuenta.",
			},
			evPasswordReset: {
				Subject:    "Su código de restablecimiento",
				Preheader:  "{validity}",
				Title:      "Restablecer la contraseña",
				Paragraphs: []string{"Introduzca este código en la página de restablecimiento para elegir una nueva contraseña."},
				CTALabel:   "Abrir la página",
				Note:       "{validity} Si usted no lo ha solicitado, ignore este mensaje: su contraseña no se modificará.",
			},
			evStaffInvitation: {
				Subject:   "Su acceso al backoffice de {brand}",
				Preheader: "Cree su contraseña y active la autenticación en dos pasos.",
				Title:     "Le han invitado al backoffice",
				Paragraphs: []string{
					"Un administrador le ha abierto un acceso al backoffice de {brand}.",
					"Siga el enlace de abajo para elegir su contraseña y después registrar su autenticación en dos pasos (obligatoria).",
				},
				CTALabel: "Activar mi acceso",
				Note:     "Este enlace es personal y de un solo uso. No lo comparta con nadie.",
			},
			evTwoFactorReset: {
				Subject:   "Se ha restablecido su autenticación en dos pasos",
				Preheader: "Se le pedirá registrarla de nuevo en su próximo inicio de sesión.",
				Title:     "Autenticación en dos pasos restablecida",
				Paragraphs: []string{
					"Un administrador ha restablecido la autenticación en dos pasos de su cuenta de {brand}.",
					"En su próximo inicio de sesión, el backoffice le pedirá registrar de nuevo su aplicación de autenticación.",
				},
				CTALabel: "Iniciar sesión en el backoffice",
				Note:     "¿No lo esperaba? Avise inmediatamente a su administrador.",
			},
			evExistingAccount: {
				Subject:   "Ya existe una cuenta con esta dirección",
				Preheader: "No es necesaria ninguna acción por su parte.",
				Title:     "Ya existe una cuenta con esta dirección",
				Paragraphs: []string{
					"Alguien acaba de intentar crear una cuenta con esta dirección. Como ya está en uso, no se ha creado ninguna cuenta nueva.",
					"Si ha sido usted: inicie sesión con normalidad o restablezca su contraseña si la ha olvidado.",
				},
				CTALabel: "Iniciar sesión",
				Note:     "Si no ha sido usted, no hay nada que hacer: su cuenta no se ha modificado.",
			},
			evEmailChangeReq: {
				Subject:    "Confirme su nueva dirección de correo",
				Preheader:  "El cambio solo se realizará cuando lo confirme.",
				Title:      "Confirme su nueva dirección de correo",
				Paragraphs: []string{"Ha solicitado usar esta dirección en su cuenta de {brand}. Confírmela para completar el cambio."},
				CTALabel:   "Confirmar mi nueva dirección",
				Note:       "¿No lo ha solicitado usted? Ignore este mensaje: no se modificará nada.",
			},
			evEmailChangeDone: {
				Subject:   "Se ha cambiado la dirección de correo de su cuenta",
				Preheader: "Si no fue usted, actúe ahora.",
				Title:     "La dirección de correo de su cuenta ha cambiado",
				Paragraphs: []string{
					"La dirección de correo de su cuenta de {brand} acaba de ser sustituida. Ya no recibirá mensajes en esta dirección.",
					"Por su seguridad, se han cerrado todas sus sesiones.",
				},
				CTALabel: "Proteger mi cuenta",
				Note:     "¿No hizo usted este cambio? Contáctenos de inmediato.",
			},
			evNewDeviceLogin: {
				Subject:   "Nuevo inicio de sesión en su cuenta",
				Preheader: "Un nuevo dispositivo acaba de iniciar sesión.",
				Title:     "Nuevo inicio de sesión en su cuenta",
				Paragraphs: []string{
					"Su cuenta {brand} se acaba de abrir desde un dispositivo que no conocíamos. La ubicación es aproximada: se deduce de la dirección IP (a menudo la ciudad del operador).",
					"¿Fue usted? No tiene que hacer nada. ¿No fue usted? Haga clic abajo: se cerrarán todas las sesiones y luego elija una nueva contraseña.",
				},
				CTALabel: "No fui yo",
				Note:     "Este enlace es válido durante 7 días y solo funciona una vez. Nunca le pediremos su contraseña por correo electrónico.",
			},
			evSuspiciousLogin: {
				Subject:   "Intentos de inicio de sesión fallidos en su cuenta",
				Preheader: "Compruebe que ha sido usted.",
				Title:     "Intentos de inicio de sesión fallidos",
				Paragraphs: []string{
					"Varios intentos de inicio de sesión en su cuenta han fallado. No se ha abierto la cuenta.",
					"Si ha sido usted, puede restablecer su contraseña. Si no, le recomendamos elegir una nueva, exclusiva de este sitio.",
				},
				CTALabel: "Cambiar mi contraseña",
				Note:     "Nunca le pediremos su contraseña por correo electrónico.",
			},
		},
	},
}
