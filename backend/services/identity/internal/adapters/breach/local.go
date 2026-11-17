package breach

import "strings"

// Local est la liste de REPLI : quelques mots de passe parmi les plus courants (clients francophones et anglophones).
// Elle ne remplace PAS le service par préfixe : c'est un filet de sécurité quand il est injoignable.
type Local struct{ set map[string]struct{} }

// NewLocal crée la liste de repli par défaut, ou une liste fournie (tests, fichier).
func NewLocal(extra ...string) *Local {
	l := &Local{set: make(map[string]struct{}, len(commonPasswords)+len(extra))}
	for _, p := range commonPasswords {
		l.set[norm(p)] = struct{}{}
	}
	for _, p := range extra {
		l.set[norm(p)] = struct{}{}
	}
	return l
}

// Contains indique si le mot de passe figure dans la liste (insensible à la casse).
func (l *Local) Contains(password string) bool {
	_, ok := l.set[norm(password)]
	return ok
}

func norm(p string) string { return strings.ToLower(strings.TrimSpace(p)) }

// Seuls les mots de passe de 8 caractères ou plus comptent : la politique en refuse déjà de plus courts.
var commonPasswords = []string{
	"12345678", "123456789", "1234567890", "11111111", "00000000", "87654321", "987654321", "12341234", "123123123", "1q2w3e4r",
	"password", "password1", "password12", "password123", "passw0rd", "p@ssw0rd", "p@ssword", "motdepasse", "motdepasse1", "motdepasse123",
	"azertyui", "azertyuiop", "azerty123", "azerty1234", "qwertyui", "qwertyuiop", "qwerty123", "qwerty1234", "qwerty12", "1qaz2wsx",
	"1qazxsw2", "zaq12wsx", "abcd1234", "abc12345", "abcdefgh", "abcd@1234", "iloveyou", "iloveyou1", "welcome1", "welcome123",
	"bonjour1", "bonjour123", "bonsoir1", "soleil12", "doudou12", "chouchou", "marseille", "mickey12", "football", "baseball",
	"superman", "batman12", "letmein1", "letmein123", "trustno1", "dragon12", "monkey12", "sunshine", "princess", "starwars",
	"admin123", "administrator", "changeme", "changeme1", "default123", "test1234", "testtest", "master12", "login123", "access12",
	"abidjan1", "abidjan12", "abidjan123", "cotedivoire", "ivoirien", "yamoussoukro", "ledan1234", "filterfinder", "azerty@123", "motdepasse@1",
}
