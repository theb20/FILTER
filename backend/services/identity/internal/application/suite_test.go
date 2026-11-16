package application_test

import "testing"

// TestSuite exécute LES MÊMES cas sur chaque adaptateur de persistance : mémoire (toujours) et PostgreSQL
// (TEST_DATABASE_URL, ou conteneur testcontainers ; REQUIRE_INTEGRATION=1 interdit de l'ignorer).
func TestSuite(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"InscriptionUniforme", testInscriptionUniforme},
		{"InscriptionErreursDeFormeSeulement", testInscriptionErreursDeFormeSeulement},
		{"PasswordPolitique8a128SansComplexite", testPasswordPolitique8a128SansComplexite},
		{"VerificationEmailUsageUniqueEtExpiration", testVerificationEmailUsageUniqueEtExpiration},
		{"ConnexionSansEmailVerifie", testConnexionSansEmailVerifie},
		{"IdentifiantsInvalidesIndistinguables", testIdentifiantsInvalidesIndistinguables},
		{"RalentissementProgressifSansVerrouillage", testRalentissementProgressifSansVerrouillage},
		{"UnTiersNePeutPasBloquerLeClient", testUnTiersNePeutPasBloquerLeClient},
		{"AlerteAuTitulaireApresPlusieursEchecs", testAlerteAuTitulaireApresPlusieursEchecs},
		{"RotationEtRejeuRevoqueLaFamille", testRotationEtRejeuRevoqueLaFamille},
		{"RefreshExpireEtInconnu", testRefreshExpireEtInconnu},
		{"RefreshConcurrentUnSeulGagnant", testRefreshConcurrentUnSeulGagnant},
		{"LogoutIdempotentEtLogoutAll", testLogoutIdempotentEtLogoutAll},
		{"ChangementMotDePasse", testChangementMotDePasse},
		{"JWKSAudienceInconnue", testJWKSAudienceInconnue},
		{"StaffAmorcageUnSeulSuperAdmin", testStaffAmorcageUnSeulSuperAdmin},
		{"StaffPremiereConnexionInscritLe2FA", testStaffPremiereConnexionInscritLe2FA},
		{"StaffConnexionAvecTOTPEtUsageUnique", testStaffConnexionAvecTOTPEtUsageUnique},
		{"StaffCodeDeSecoursUneSeuleFois", testStaffCodeDeSecoursUneSeuleFois},
		{"StaffCinqCodesErronesInvalidentLeJeton", testStaffCinqCodesErronesInvalidentLeJeton},
		{"StaffJetonDeDefiLieAIPPortéeEtExpiration", testStaffJetonDeDefiLieAIPPorteeEtExpiration},
		{"StaffEnrollConsommeLeJeton", testStaffEnrollConsommeLeJeton},
		{"StaffConnexionRefusee", testStaffConnexionRefusee},
		{"StaffEtClientsSontDeuxMondes", testStaffEtClientsSontDeuxMondes},
		{"StaffRefreshRotationRejeuEtDeconnexion", testStaffRefreshRotationRejeuEtDeconnexion},
		{"StaffUnCodeTOTPNePasseQuUneFoisEnConcurrence", testStaffUnCodeTOTPNePasseQuUneFoisEnConcurrence},
		{"StaffDeconnexionDeToutesLesSessions", testStaffDeconnexionDeToutesLesSessions},
		{"AdminRolesCRUD", testAdminRolesCRUD},
		{"AdminRolesSystemEtAttribues", testAdminRolesSystemEtAttribues},
		{"AdminRolesListeRechercheEtPagination", testAdminRolesListeRechercheEtPagination},
		{"AdminPasDElevationDePrivileges", testAdminPasDElevationDePrivileges},
		{"AdminInvitationComplete", testAdminInvitationComplete},
		{"AdminInvitationExpireEtRenvoi", testAdminInvitationExpireEtRenvoi},
		{"AdminGardeFousSoiMemeEtDernierSuperAdmin", testAdminGardeFousSoiMemeEtDernierSuperAdmin},
		{"AdminDesactivationRevoqueLesSessions", testAdminDesactivationRevoqueLesSessions},
		{"AdminModifierUnMembre", testAdminModifierUnMembre},
		{"AdminReinitialiserLe2FA", testAdminReinitialiserLe2FA},
		{"AdminListeDuPersonnel", testAdminListeDuPersonnel},
		{"AdminChangerSonMotDePasse", testAdminChangerSonMotDePasse},
		{"AdminJournalDAudit", testAdminJournalDAudit},
		{"AdminAuditEtActionSontAtomiques", testAdminAuditEtActionSontAtomiques},
		{"AdminClientsListeEtSuspension", testAdminClientsListeEtSuspension},
		{"AdminEntrepriseApprobation", testAdminEntrepriseApprobation},
		{"AdminEntrepriseRejet", testAdminEntrepriseRejet},
		{"AdminEntrepriseExigeLaPermissionEnBase", testAdminEntrepriseExigeLaPermissionEnBase},
		{"ReglagesParDefautEtValidation", testReglagesParDefautEtValidation},
		{"ReglagesModifiesAppliquesEtAudites", testReglagesModifiesAppliquesEtAudites},
	}
	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			current = b
			for _, c := range cases {
				t.Run(c.name, c.fn)
			}
		})
	}
}
