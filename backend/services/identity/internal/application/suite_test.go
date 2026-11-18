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
		{"MotDePasseOublieParcoursComplet", testMotDePasseOublieParcoursComplet},
		{"MotDePasseOublieAucuneFuiteEtEssaisRalentis", testMotDePasseOublieAucuneFuiteEtEssaisRalentis},
		{"OAuthNouveauCompteEtReconnexion", testOAuthNouveauCompteEtReconnexion},
		{"OAuthLiaisonSeulementSiAdresseVerifiee", testOAuthLiaisonSeulementSiAdresseVerifiee},
		{"OAuthRefusDesFalsifications", testOAuthRefusDesFalsifications},
		{"ReglagesParDefautEtValidation", testReglagesParDefautEtValidation},
		{"ReglagesModifiesAppliquesEtAudites", testReglagesModifiesAppliquesEtAudites},
		{"PreferencesDefautsEtValidation", testPreferencesDefautsEtValidation},
		{"PreferencesEnregistreesRemplaceesEtIsolees", testPreferencesEnregistreesRemplaceesEtIsolees},
		{"PreferencesCompteInconnuOuMondeInconnu", testPreferencesCompteInconnuOuMondeInconnu},
		{"ProfilModificationPartielleEtValidation", testProfilModificationPartielleEtValidation},
		{"EmailChangementParcoursComplet", testEmailChangementParcoursComplet},
		{"EmailChangementRefusEtUniformite", testEmailChangementRefusEtUniformite},
		{"EmailChangementUnSeulLienEtUsageUnique", testEmailChangementUnSeulLienEtUsageUnique},
		{"EmailChangementCourseSurLAdresse", testEmailChangementCourseSurLAdresse},
		{"EntrepriseLectureEtRevalidation", testEntrepriseLectureEtRevalidation},
		{"EntrepriseValidationEtParticulier", testEntrepriseValidationEtParticulier},
		{"ConsentementsJournalAppendOnly", testConsentementsJournalAppendOnly},
		{"ConsentementsValidationEtVisiteur", testConsentementsValidationEtVisiteur},
		{"VerifierUnePermissionLueEnBase", testVerifierUnePermissionLueEnBase},
		{"SecuriteAppareilsConnectesEtDeconnexion", testSecuriteAppareilsConnectesEtDeconnexion},
		{"SecuriteAlerteNouvelAppareilEtLienDeSecours", testSecuriteAlerteNouvelAppareilEtLienDeSecours},
		{"SecuriteHistoriqueConserve90Jours", testSecuriteHistoriqueConserve90Jours},
		{"SecuritePhotoEtCouverture", testSecuritePhotoEtCouverture},
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
