# Périmètre gateway + identity — relecture route par route

Statut : **validé sous réserve de corrections de sécurité, appliquées ci-dessous.** Les ajouts et modifications d'événements sont validés.
Cette table est **vérifiée par machine** (`tools/check_contract.py`) : chaque route d'un tag `validated` figure dans [`api/route-rpc-map.yaml`](../api/route-rpc-map.yaml), chaque RPC cité existe, et les règles de sécurité ci-dessous sont contrôlées (le contrôle échoue si l'une d'elles est violée ; testé par mutation).

---

## 1. Corrections de sécurité demandées et leur application

| # | Demande | Application dans le contrat |
|---|---|---|
| 1 | Jeton de rafraîchissement en cookie, jamais dans le corps ; protection CSRF ; CORS avec identifiants ; backoffice sans jeton dans le navigateur | Cookie `ff_refresh` : `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/v1/auth`. `AuthTokens` ne contient plus que `accessToken` (en mémoire côté site). `POST /auth/refresh` et `/auth/logout` lisent le cookie et exigent la **protection CSRF** (G16). G6 adapté. Routes du personnel : appelées **uniquement par le serveur Next.js**, qui garde les jetons côté serveur (G17). |
| 2 | Pas de verrouillage dur | `ACCOUNT_LOCKED` supprimé. **Ralentissement progressif par couple (compte, IP)** (`429` + `Retry-After`) ; **e-mail d'alerte** au titulaire après plusieurs échecs (`SuspiciousLoginAlertRequested`). Paramètres dans `identity-settings.loginThrottle` et `loginAlertAfterFailures`. |
| 3 | 2FA du personnel : jeton temporaire à usage unique | Schéma de sécurité `staffChallenge` (Bearer, audience `staff-challenge`). Émis par `StaffLogin` ou `AcceptStaffInvitation`, **portée** `verify` / `enroll` / `confirm`, courte durée (réglage), **5 codes erronés maximum**, lié à l'IP. `enroll` **consomme** son jeton et en renvoie un nouveau (`confirm`). Routes 2FA : plus de jeton dans le corps. |
| 4 | Plus aucun corps « objet » | Schémas nommés partout (**vérifié sur les 356 opérations**, pas seulement le périmètre validé) ; `additionalProperties: false` sur tout corps de requête. Seule exception volontaire : `PaymentWebhookPayload`, corps opaque propre au prestataire. |
| 5 | Personnel : désactivation, pas suppression | `DELETE /admin/staff-users/{id}` **supprimé** ; `POST …/deactivate` et `…/reactivate` ; statut `deactivated` ; audit conservé. Interdits (`422`) : dernier super-admin (`LAST_SUPER_ADMIN`), se désactiver soi-même, se retirer son propre rôle d'administration (`SELF_ACTION_FORBIDDEN`). Réinitialisation du 2FA et renvoi d'invitation **notifient** la personne (`StaffTwoFactorReset`, `StaffInvitationRequested`). |
| 6 | OAuth : `state`, PKCE, `nonce` | Routes `/auth/oauth/{provider}/start` et `/callback` (abstraction du fournisseur, `google` seul). `state`, **PKCE S256** et **`nonce`** générés côté serveur, mémorisés 10 min, **cookie de liaison** `ff_oauth` (HttpOnly, Secure, SameSite=Lax) ; validation du jeton d'identité (signature, `iss`, `aud`, `exp`, `nonce`, `email_verified`) ; `redirectUri` en liste blanche. |
| 7 | Audit : une seule voie d'écriture | `AppendAudit` **supprimé**. Seule voie : événements `audit.*` (`AuditRecorded`) publiés par chaque service via **outbox**, consommés par identity de façon idempotente (`event_id`). Lecture seule via `GET /admin/audit-log`. |
| 8 | Inscription sans fuite d'existence | `POST /auth/register` répond **toujours 202** avec le même corps (`Accepted`) ; plus de `409`. Nouvelle adresse : compte créé + e-mail de vérification. Adresse existante : aucun compte, e-mail « un compte existe déjà » au titulaire (`RegistrationAttemptOnExistingAccount`). **Même travail** (hachage) dans les deux cas. **Aucun jeton** renvoyé : l'utilisateur se connecte ensuite. Même principe pour le changement d'e-mail et « mot de passe oublié ». |

## 2. Réponses aux 10 questions

| # | Décision | Application |
|---|---|---|
| 1 | E-mail non vérifié : navigation, panier, commande autorisés ; **bloqués** : avis, validation d'un compte entreprise, paiement différé | Code `EMAIL_NOT_VERIFIED` : `POST /products/{id}/reviews` (403), `POST /admin/companies/{id}/approve` (422), `deferred_invoice` absent de `GET /payment-methods` |
| 2 | Changement d'e-mail : mot de passe, vérification de la nouvelle adresse, notification de l'ancienne | `POST /me/email` (202 uniforme) puis `POST /me/email/confirm` ; événements `EmailChangeRequested`, `EmailChangeConfirmed` |
| 3 | « Déconnecter partout » ; automatique après changement/réinitialisation du mot de passe | `POST /auth/logout-all`, `POST /admin/auth/logout-all` ; appelé automatiquement par `ChangePassword` / `ResetPassword` |
| 4 | Mot de passe du personnel : renvoi d'invitation | `POST /admin/staff-users/{id}/resend-invitation` |
| 5 | 8–128 (clients), ≥ 12 (personnel), pas de règle de complexité, **refus des mots de passe compromis** | Code `PASSWORD_COMPROMISED` sur inscription, changement, réinitialisation, invitation. **Source de la liste : à choisir** (voir §6) |
| 6 | Accès 15 min ; rafraîchissement 30 j (clients) / 8 h (personnel) ; code 6 chiffres, 15 min, 5 essais ; **tout en réglages** | `identity-settings` (valeurs par défaut = les tiennes) |
| 7 | `pro` dans le jeton pour l'affichage ; **statut revérifié en direct** à la commande et pour tout différé | G3 ; `POST /orders` et `GET /payment-methods` le précisent |
| 8 | Revérification sans cache des actions sensibles | G4 |
| 9 | Limites de débit validées, **sauf le point 2** | G5 (verrouillage remplacé par ralentissement progressif) |
| 10 | Google seul, abstraction conservée | `{provider}` dans les routes, `oauth_identities(provider, subject)` |

---

## 3. Gateway : règles transverses

Valeurs numériques = défauts techniques, tous configurables.

| # | Sujet | Règle |
|---|---|---|
| G1 | Routage | `/v1/**` vers le service propriétaire du tag ; `/v1/admin/**` réservé au personnel ; un seul point d'entrée public. |
| G2 | Authentification | JWT clients (`aud = customer`) et personnel (`aud = staff`, `amr` ∋ `totp`), vérifiés **localement** avec `identity.GetJwks` (rafraîchi toutes les 10 min et sur `kid` inconnu, tolérance 30 s). **Jeton d'accès en mémoire** côté site ; **rafraîchissement par cookie** (G16). Cookie invité `guest_token` (HttpOnly, Secure, SameSite=Lax). Jeton 2FA temporaire : audience `staff-challenge` (G17). |
| G3 | Contenu des jetons | Clients : `sub`, `aud`, `exp`, `jti`, `sid` (session : famille de rafraîchissement, sert à conserver la session courante lors d'un changement de mot de passe), `kind`, `pro`, `email_verified`. Personnel : `sub`, `aud`, `exp`, `jti`, `roles`, `perms`, `amr`. `pro` et `email_verified` servent à l'**affichage** et au filtrage grossier ; le statut de l'entreprise est **revérifié en direct** à la commande et pour tout paiement différé. Accès 15 min ; rotation du rafraîchissement, famille révoquée en cas de rejeu. |
| G4 | Autorisation du personnel | Le gateway applique `x-required-permission` d'après `perms`. Actions sensibles (`orders:refund`, `payments:reconcile`, `companies:approve`, `quotes:convert`, `customers:privacy`, `admin:write`) **revérifiées sans cache** par le service propriétaire via `identity.CheckPermission` ; révocation immédiate pour elles, sous 15 min pour le reste. |
| G5 | Limitation de débit (Redis) | Anonyme 120 req/min/IP ; connecté 600 req/min/utilisateur. **Authentification** : connexion 10/min/IP ; **ralentissement progressif par couple (compte, IP)** — 3 échecs gratuits puis délai doublé à chaque échec (1 s, 2 s, 4 s…) plafonné à 900 s, remis à zéro après un succès ou 1 h sans échec (**valeurs proposées**, réglables) ; **aucun verrouillage de compte** ; inscription 5/h/IP ; mot de passe oublié 5/h par IP et par e-mail ; code de réinitialisation 5 essais ; 2FA : 5 codes erronés par jeton. En-têtes `RateLimit-*`, `Retry-After`. |
| G6 | CORS **avec identifiants** | Liste blanche exacte par environnement : `Access-Control-Allow-Origin` = l'origine reçue (jamais `*`), `Access-Control-Allow-Credentials: true`, `Vary: Origin`, uniquement pour l'origine du site web. En-têtes acceptés : `Authorization`, `Content-Type`, `Accept-Language`, `Idempotency-Key`, `If-Match`, `X-Requested-With`. Expose `ETag`, `RateLimit-*`, `Retry-After`. Le backoffice n'est pas une origine navigateur (G17). |
| G7 | Idempotence | `Idempotency-Key` obligatoire où déclaré ; réponse mémorisée 24 h sous (acteur, méthode, route, clé) avec l'empreinte du corps ; corps différent → `409 IDEMPOTENCY_KEY_REUSED`. |
| G8 | Erreurs gRPC → HTTP | `INVALID_ARGUMENT` → 400/422 · `FAILED_PRECONDITION` → 422 (code métier dans `ErrorInfo.reason`) · `NOT_FOUND` → 404 · `ALREADY_EXISTS`/`ABORTED` → 409 · `PERMISSION_DENIED` → 403 · `UNAUTHENTICATED` → 401 · `RESOURCE_EXHAUSTED` → 429 (+ `Retry-After`) · `UNAVAILABLE`/`DEADLINE_EXCEEDED` → 503/504. Corps RFC 9457 avec `code` et `traceId`, jamais de détail interne. |
| G9 | Métadonnées transmises | `traceparent`, `x-request-id`, `x-actor-id`, `x-actor-type`, `x-locale`, `x-currency`, `x-client-ip`, `x-idempotency-key` ; crues uniquement si elles viennent du gateway (réseau privé). |
| G10 | Validation | Contre l'OpenAPI : **champs inconnus refusés** (d'où les schémas nommés), JSON ≤ 1 Mio, imports CSV ≤ 20 Mio, fichiers volumineux par URL pré-signée. |
| G11 | Cas particuliers | Webhooks de paiement : corps **brut** + en-têtes transmis à `payment.HandleWebhook`. Chat : WebSocket vers `support.StreamChat`. |
| G12 | Sécurité HTTP | HSTS, `nosniff`, en-tête `Server` neutre ; IP cliente depuis `X-Forwarded-For` seulement derrière un proxy de confiance. |
| G13 | Versionnage | `/v1` ; rupture → `/v2` ; `Deprecation` et `Sunset`. |
| G14 | Sondes | `/healthz` : vivant. `/readyz` : Redis + canal gRPC vers `identity`. |
| G15 | Observabilité | OpenTelemetry, logs JSON ; **jamais** de secret, jeton ou code de réinitialisation dans les logs. |
| **G16** | **CSRF** (routes qui lisent `ff_refresh` : `/auth/refresh`, `/auth/logout`) | Défense en couches : (1) `SameSite=Strict` et `Path=/v1/auth` ; (2) en-tête **`Origin` obligatoire** et dans la liste blanche (et `Sec-Fetch-Site` ∈ same-origin / same-site quand présent) ; (3) en-tête personnalisé **`X-Requested-With: ff-web`** obligatoire (impose un contrôle CORS préalable) ; (4) corps vide ou JSON uniquement. Sinon `403 CSRF_FAILED`. Aucun jeton CSRF stocké : un cookie lisible par JavaScript serait en contradiction avec la consigne. |
| **G17** | **Backoffice** | Les routes `/admin/auth/*` ne sont appelées que par le **serveur Next.js** : jetons d'accès, de rafraîchissement et jeton 2FA **gardés côté serveur** (session chiffrée en cookie `HttpOnly` propre au backoffice), jamais dans le navigateur. Le gateway restreint `/v1/admin/**` à l'origine ou à l'adresse réseau du backoffice (liste blanche) en plus du JWT. |

---

## 4. Identity : correspondance exacte route ↔ RPC

Légende de l'accès : `publique`, `client` (JWT `customer`), `personnel` (JWT `staff` + permission), `cookie de rafraîchissement + CSRF`, `jeton temporaire 2FA`. Un `422` porte un `code` métier.

### Compte (identity)

| # | Route | Accès | Corps → réponse | RPC | Permission | Erreurs |
|---|---|---|---|---|---|---|
| 1 | `POST /auth/email/resend` | client | — → 202 Accepted | `identity.ResendEmailVerification` | — | 401, 403, 409, 429 |
| 2 | `POST /auth/email/verify` | publique | VerifyEmailRequest → 204 vide | `identity.VerifyEmail` | — | 422, 429 |
| 3 | `POST /auth/login` | publique | LoginRequest → 200 AuthResult + Set-Cookie | `identity.Login` | — | 401, 422, 429 |
| 4 | `POST /auth/logout` | cookie de rafraîchissement + CSRF | — → 204 vide + Set-Cookie | `identity.Logout` | — | 403, 429 |
| 5 | `POST /auth/logout-all` | client | — → 204 vide + Set-Cookie | `identity.LogoutAll` | — | 401, 403, 409, 429 |
| 6 | `POST /auth/oauth/exchange` | publique | OAuthExchangeRequest → 200 AuthResult + Set-Cookie | `identity.ExchangeOAuthCode` | — | 401, 422, 429 |
| 7 | `GET /auth/oauth/{provider}/callback` | publique | — → 302 redirection | `identity.CompleteOAuth` | — | 404, 429 |
| 8 | `GET /auth/oauth/{provider}/start` | publique | — → 302 redirection + Set-Cookie | `identity.StartOAuth` | — | 404, 422, 429 |
| 9 | `POST /auth/password/forgot` | publique | ForgotPasswordRequest → 202 vide | `identity.ForgotPassword` | — | 422, 429 |
| 10 | `POST /auth/password/reset` | publique | ResetPasswordRequest → 204 vide | `identity.ResetPassword` | — | 422, 429 |
| 11 | `POST /auth/password/verify-code` | publique | VerifyCodeRequest → 200 VerifyCodeResult | `identity.VerifyPasswordCode` | — | 422, 429 |
| 12 | `POST /auth/refresh` | cookie de rafraîchissement + CSRF | — → 200 AuthTokens + Set-Cookie | `identity.RefreshTokens` | — | 401, 403, 429 |
| 13 | `POST /auth/register` | publique | RegisterRequest → 202 Accepted | `identity.Register` | — | 422, 429 |
| 14 | `POST /consents` | publique | ConsentInput → 201 ConsentRecord | `identity.RecordConsent` | — | 422, 429 |
| 15 | `DELETE /me` | client | EraseAccountRequest → 202 DataRequest | `identity.RequestErasure` | — | 401, 403, 409, 422 |
| 16 | `GET /me` | client | — → 200 Me | `identity.GetUser` | — | 401, 403 |
| 17 | `PATCH /me` | client | MeUpdate → 200 Me | `identity.UpdateUser` | — | 401, 403, 422 |
| 18 | `GET /me/company` | client | — → 200 Company | `identity.GetCompany` | — | 401, 403 |
| 19 | `PUT /me/company` | client | CompanyInput → 200 Company | `identity.PutCompany` | — | 401, 403, 409, 422 |
| 20 | `GET /me/consents` | client | — → 200 ConsentRecord | `identity.GetConsents` | — | 401, 403 |
| 21 | `PUT /me/consents` | client | ConsentInput → 200 ConsentRecord | `identity.RecordConsent` | — | 401, 403, 422 |
| 22 | `POST /me/data-export` | client | — → 202 DataRequest | `identity.RequestDataExport` | — | 401, 403, 409 |
| 23 | `GET /me/data-requests` | client | — → 200 [DataRequest] | `identity.ListMyDataRequests` | — | 401, 403 |
| 24 | `POST /me/email` | client | EmailChangeRequest → 202 Accepted | `identity.RequestEmailChange` | — | 401, 403, 409, 422, 429 |
| 25 | `POST /me/email/confirm` | publique | VerifyEmailRequest → 204 vide | `identity.ConfirmEmailChange` | — | 422, 429 |
| 26 | `POST /me/password` | client | ChangePasswordRequest → 204 vide | `identity.ChangePassword` | — | 401, 403, 409, 422, 429 |
| 26b | `GET` · `PUT /me/preferences` | client | — → 200 Preferences · Preferences → 200 Preferences | `identity.GetPreferences` · `identity.PutPreferences` | — | 401, 403, 400, 422 |

### Technique (gateway seul)

| # | Route | Accès | Corps → réponse | RPC | Permission | Erreurs |
|---|---|---|---|---|---|---|
| 27 | `GET /healthz` | publique | — → 200 HealthzResponse | `gateway` | — | 429 |
| 28 | `GET /readyz` | publique | — → 200 ReadyzResponse | `gateway` | — | 429 |

### Admin · Authentification (identity)

| # | Route | Accès | Corps → réponse | RPC | Permission | Erreurs |
|---|---|---|---|---|---|---|
| 29 | `POST /admin/auth/2fa/enroll` | jeton temporaire 2FA | — → 200 TwoFactorEnrollment | `identity.StaffEnrollTwoFactor` | — | 401, 409, 429 |
| 30 | `POST /admin/auth/2fa/enroll/confirm` | jeton temporaire 2FA | TwoFactorCodeRequest → 200 StaffTokens | `identity.StaffConfirmTwoFactor` | — | 401, 409, 422, 429 |
| 31 | `POST /admin/auth/2fa/verify` | jeton temporaire 2FA | TwoFactorCodeRequest → 200 StaffTokens | `identity.StaffVerifyTwoFactor` | — | 401, 409, 422, 429 |
| 32 | `POST /admin/auth/invitation/accept` | publique | AcceptInvitationRequest → 200 StaffLoginResult | `identity.AcceptStaffInvitation` | — | 422, 429 |
| 33 | `POST /admin/auth/login` | publique | StaffLoginRequest → 200 StaffLoginResult | `identity.StaffLogin` | — | 401, 422, 429 |
| 34 | `POST /admin/auth/logout` | publique | LogoutRequest → 204 vide | `identity.StaffLogout` | — | 422, 429 |
| 35 | `POST /admin/auth/logout-all` | personnel | — → 204 vide | `identity.StaffLogoutAll` | — | 401, 403, 409, 429 |
| 36 | `POST /admin/auth/refresh` | publique | RefreshRequest → 200 StaffTokens | `identity.StaffRefresh` | — | 401, 422, 429 |
| 37 | `GET /admin/me` | personnel | — → 200 StaffMe | `identity.GetStaffProfile` | — | 401, 403 |
| 38 | `POST /admin/me/password` | personnel | ChangePasswordRequest → 204 vide | `identity.ChangeStaffPassword` | — | 401, 403, 409, 422, 429 |
| 38b | `GET` · `PUT /admin/me/preferences` | personnel | — → 200 Preferences · Preferences → 200 Preferences | `identity.GetPreferences` · `identity.PutPreferences` | — | 401, 403, 400, 422 |

### Admin · Administration (identity)

| # | Route | Accès | Corps → réponse | RPC | Permission | Erreurs |
|---|---|---|---|---|---|---|
| 39 | `GET /admin/audit-log` | personnel | — → 200 AuditEntryCursorPage | `identity.ListAudit` | `audit:read` | 401, 403 |
| 40 | `GET /admin/identity-settings` | personnel | — → 200 IdentitySettings | `identity.GetSettings` | `admin:read` | 401, 403 |
| 41 | `PUT /admin/identity-settings` | personnel | IdentitySettings → 200 IdentitySettings | `identity.PutSettings` | `admin:write` | 401, 403, 422 |
| 42 | `GET /admin/permissions` | personnel | — → 200 [Permission] | `identity.ListPermissions` | `admin:read` | 401, 403 |
| 43 | `GET /admin/roles` | personnel | — → 200 [Role] | `identity.ListRoles` | `admin:read` | 401, 403 |
| 44 | `POST /admin/roles` | personnel | RoleInput → 201 Role | `identity.CreateRole` | `admin:write` | 401, 403, 409, 422 |
| 45 | `DELETE /admin/roles/{id}` | personnel | — → 204 vide | `identity.DeleteRole` | `admin:write` | 401, 403, 404, 409 |
| 46 | `GET /admin/roles/{id}` | personnel | — → 200 Role | `identity.GetRole` | `admin:read` | 401, 403, 404 |
| 47 | `PUT /admin/roles/{id}` | personnel | RoleInput → 200 Role | `identity.UpdateRole` | `admin:write` | 401, 403, 404, 409, 422 |
| 48 | `GET /admin/staff-users` | personnel | — → 200 StaffUserPage | `identity.ListStaffUsers` | `admin:read` | 401, 403 |
| 49 | `POST /admin/staff-users` | personnel | StaffUserInput → 201 StaffUser | `identity.CreateStaffUser` | `admin:write` | 401, 403, 409, 422 |
| 50 | `GET /admin/staff-users/{id}` | personnel | — → 200 StaffUser | `identity.GetStaffUser` | `admin:read` | 401, 403, 404 |
| 51 | `PUT /admin/staff-users/{id}` | personnel | StaffUserInput → 200 StaffUser | `identity.UpdateStaffUser` | `admin:write` | 401, 403, 404, 409, 422 |
| 52 | `POST /admin/staff-users/{id}/deactivate` | personnel | — → 200 StaffUser | `identity.DeactivateStaffUser` | `admin:write` | 401, 403, 404, 409, 422 |
| 53 | `POST /admin/staff-users/{id}/reactivate` | personnel | — → 200 StaffUser | `identity.ReactivateStaffUser` | `admin:write` | 401, 403, 404, 409, 422 |
| 54 | `POST /admin/staff-users/{id}/resend-invitation` | personnel | — → 204 vide | `identity.ResendStaffInvitation` | `admin:write` | 401, 403, 404, 409, 422 |
| 55 | `POST /admin/staff-users/{id}/reset-two-factor` | personnel | — → 204 vide | `identity.ResetStaffTwoFactor` | `admin:write` | 401, 403, 404 |

### Admin · Clients et entreprises (identity)

| # | Route | Accès | Corps → réponse | RPC | Permission | Erreurs |
|---|---|---|---|---|---|---|
| 56 | `GET /admin/companies` | personnel | — → 200 CompanyPage | `identity.ListCompanies` | `customers:read` | 401, 403 |
| 57 | `GET /admin/companies/{id}` | personnel | — → 200 Company | `identity.GetCompanyById` | `customers:read` | 401, 403, 404 |
| 58 | `POST /admin/companies/{id}/approve` | personnel | — → 200 Company | `identity.ApproveCompany` | `companies:approve` | 401, 403, 404, 409, 422 |
| 59 | `POST /admin/companies/{id}/reject` | personnel | RejectCompanyRequest → 200 Company | `identity.RejectCompany` | `companies:approve` | 401, 403, 404, 409, 422 |
| 60 | `GET /admin/customers` | personnel | — → 200 CustomerAdminPage | `identity.ListCustomers` | `customers:read` | 401, 403 |
| 61 | `GET /admin/customers/{id}` | personnel | — → 200 CustomerAdmin | `identity.GetCustomer` | `customers:read` | 401, 403, 404 |
| 62 | `PATCH /admin/customers/{id}` | personnel | CustomerStatusUpdate → 200 CustomerAdmin | `identity.SetCustomerStatus` | `customers:write` | 401, 403, 404, 409, 422 |
| 63 | `GET /admin/data-requests` | personnel | — → 200 DataRequestPage | `identity.ListDataRequests` | `customers:read` | 401, 403 |
| 64 | `POST /admin/data-requests/{id}/process` | personnel | — → 200 DataRequestAdmin | `identity.ProcessDataRequest` | `customers:privacy` | 401, 403, 404, 409, 422 |

---

## 5. RPC `identity` sans route publique (internes)

| RPC | Usage |
|---|---|
| `GetJwks` | Le gateway récupère les clés publiques de vérification des JWT (audiences `customer`, `staff`, `staff-challenge`) |
| `CheckPermission` | Revérification sans cache des actions sensibles (G4) |
| `CountPendingCompanies` | Compteur du tableau de bord (`/admin/dashboard/overview`, tag en brouillon) |

`AppendAudit` n'existe plus (correction n° 7). `RecordConsent` sert **deux** routes (`PUT /me/consents`, `POST /consents`) : voulu. `identity` appelle `pricing.ValidateTaxId` (brouillon) pour `taxIdVerified`, sans bloquer l'inscription.

---

## 6. Décisions prises sur les points soumis

| Sujet | Décision | Application |
|---|---|---|
| Domaines | Site, API et backoffice = sous-domaines d'un même domaine (**nom : pas encore choisi**) ; `SameSite=Strict` validé | `COOKIE_DOMAIN` (gateway) : vide = cookie lié à l'hôte de l'API ; sinon domaine parent (ex. `exemple.ci`). Le même `Domain` sert à **effacer** le cookie. |
| Parcours Google avec `Strict` | À vérifier | **Vérifié dans Chromium** (voir ci-dessous). `Lax` n'est **pas** nécessaire. |
| Mots de passe compromis | Service par préfixe de condensat (k-anonymat), délai court, repli sur une liste locale, jamais bloquant ; derrière une interface | `identity/internal/adapters/breach` : seuls 5 caractères du SHA-1 quittent le serveur (+ `Add-Padding`) ; délai 1,5 s (`BREACH_TIMEOUT_MS`) ; repli local sans erreur ; aucun mot de passe, préfixe ni condensat dans les logs. `BREACH_CHECK=pwned\|local\|disabled` (`disabled` interdit hors dev). |
| Valeurs | Toutes validées, toutes en réglages | Défauts dans `identity-settings` (contrat) et `domain.DefaultSettings` : ralentissement 3 / 1 s / 900 s par (compte, IP), alerte après 5 échecs, jeton 2FA 5 min, invitation 72 h, vérification d'e-mail 48 h, changement d'e-mail 24 h, connexion récente 10 min. |

### Parcours Google avec `SameSite=Strict`

Aucun cookie `Strict` n'est lu pendant le retour depuis `accounts.google.com`. Seul le cookie de liaison `ff_oauth` (`SameSite=Lax`, limité à `/v1/auth/oauth`) doit voyager lors de cette navigation inter-sites ; le cookie `ff_refresh` (`Strict`) n'est posé qu'à l'étape suivante, par l'appel d'échange fait depuis l'application (même site).

Test réel dans **Chromium** (3 « sites » locaux : application et API sur le même site, fournisseur sur un autre) :
- retour du fournisseur (navigation inter-sites) : `ff_oauth` envoyé au callback ✔ ;
- échange puis rafraîchissement depuis l'application : `ff_refresh` (`Strict`, `HttpOnly`, `Path=/v1/auth`) posé et renvoyé ✔ ;
- formulaire `POST` d'un **autre site** vers `/auth/refresh` : le cookie `Strict` n'est **pas** envoyé ✔.

Non vérifié : Firefox et Safari (non installés ici) et le vrai Google. À valider sur une préproduction avec le vrai domaine.

---

## 7. État d'implémentation (mesuré, pas déclaré)

Règle : « écrit » n'est pas « validé ». Une ligne n'est marquée **exécuté** que si un test l'a réellement fait tourner (mémoire **et** PostgreSQL réel pour identity). Dernière mesure : **61 RPC sur 66** implémentés dans `grpcserver`.

| Élément | État | Preuve |
|---|---|---|
| `pkg/` (erreurs, RFC 9457, JWT EdDSA + JWKS, ralentissement, événements, logs expurgés) | écrit, **exécuté** | tests `pkg/*` |
| identity : domaine, cas d'usage | **61 / 66 RPC** (comptes, sessions, rotation et rejeu, mot de passe oublié, OAuth, personnel + 2FA, invitations, rôles, clients, entreprises, audit, réglages, préférences, profil, changement d'e-mail, entreprise du compte, consentements, contrôle de permission) | suite partagée exécutée sur **mémoire et PostgreSQL réel** |
| identity : **5 RPC manquantes** (toutes RGPD) | `RequestErasure`, `RequestDataExport`, `ListMyDataRequests`, `ListDataRequests`, `ProcessDataRequest` — en attente de vos décisions (§9) | répondent `UNIMPLEMENTED` |
| gateway : serveur généré (ogen) + CORS, CSRF, débit, en-têtes, JWT locaux | **exécuté** (tests CSRF, cookie, rotation, débit connexion et inscription, permissions du personnel) | `gateway/internal/server/*_test.go` |
| gateway : **5 routes** RGPD (`DELETE /me`, `/me/data-export`, `/me/data-requests`, `/admin/data-requests` ×2) | répondent **501** | `api.UnimplementedHandler` |
| gateway + identity : profil (`PATCH /me`), e-mail (`POST /me/email`, `/me/email/confirm`), entreprise (`/me/company` ×2), consentements (`/me/consents` ×2, `POST /consents`) | **exécuté** : suite partagée **mémoire + PostgreSQL 17 réel** (course sur une même adresse, lien unique, expiration, revalidation de l'entreprise, journal append-only), 3 langues pour les e-mails, tests gateway, et **parcours HTTP réel** (gateway + identity + PostgreSQL + NATS + Redis) | `application/profile_test.go`, `gateway/.../profile_test.go` |
| PostgreSQL (identity) : migrations goose, unité de travail, contraintes en base | **exécuté** (PostgreSQL 17 réel) | `adapters/postgres`, suite partagée |
| Outbox + relais NATS JetStream | écrit, **exécuté** (PostgreSQL + NATS réels) | `pkg/outbox`, `pkg/natsbus` |
| **Redis** | **écrit et testé sur miniredis (Lua réel)** : `RedisLimiter` (fenêtre glissante atomique, horloge Redis, partage entre instances testé, concurrence : exactement `limit` autorisées sur 60 appels). `REDIS_URL` **obligatoire hors dev** (démarrage refusé sinon). Panne : repli par instance + journal d'erreur + `/readyz` échoue si Redis est injoignable (G14) ; sonde « redis » dans la santé admin. **Non exécuté contre un vrai serveur Redis** (aucun Redis/Docker sur ce poste) | `services/gateway/internal/server/ratelimit_redis{,_test}.go`, `cmd/gateway/main.go` |
| Idempotence `Idempotency-Key` (G7) | **écrite et testée** (mémoire + miniredis, `-race`) : rejeu à l'identique, `409 IDEMPOTENCY_KEY_REUSED` si le corps change, `409 CONFLICT` si l'appel est encore en cours, 20 appels simultanés → 1 seule exécution, erreur → clé libérée, portée (acteur, méthode, chemin, clé), table des routes vérifiée contre `openapi.yaml`. Magasin HS → **503, appel refusé** (jamais d'exécution sans garantie). **Aucune des 7 routes concernées n'est encore servie** (commandes/paiements/remboursements = 501) : le mécanisme est prêt, pas exercé sur un vrai handler. Non exécuté contre un vrai Redis | `services/gateway/internal/server/idempotency{,_test}.go` |
| OAuth Google | exécuté avec un **faux fournisseur** ; vrai Google **non testé** | `oauth_test.go` |
| Docker Compose, Dockerfile | écrits, **non exécutés** (Docker absent) | — |
| Navigateurs réels (Firefox, Safari), préproduction avec le vrai domaine | **non testés** | voir §6 |

## 8. Préférences d'interface (`/me/preferences`, `/admin/me/preferences`)

Préférences d'affichage d'un compte, synchronisées entre ses appareils (aujourd'hui : le guide visuel — activé, animations, visites déjà faites).

- **Le compte est toujours celui du jeton**, jamais un identifiant fourni par l'appelant : on ne peut ni lire ni écrire celles d'un autre. Le corps refuse les champs inconnus (`additionalProperties: false`).
- **Deux mondes séparés** : personnel (`/admin/me/preferences`, jeton du personnel, aucune permission requise) et clients (`/me/preferences`, jeton client). Un jeton de l'un n'ouvre pas la route de l'autre.
- **`PUT` remplace l'ensemble** (dernière écriture gagnante). Un compte qui n'a rien enregistré reçoit les défauts : guide actif, animations `auto`, aucune visite faite. `seen` vaut `[]`, jamais `null`.
- **Bornes** : au plus 200 visites, identifiants `^[a-z0-9-]{1,64}$` sans doublon (vérifiés au gateway, dans le domaine et par `CHECK` en base pour le nombre et les valeurs de `animations`).
- **Stockage** : table `user_preferences` (migration 00007), un propriétaire exactement (client OU membre du personnel, clés étrangères `ON DELETE CASCADE` : supprimer le compte efface ses préférences).
- **Pas d'entrée d'audit** : ce sont des réglages d'affichage personnels, sans effet sur la sécurité ni sur les données de l'entreprise.

## 9. Profil, e-mail, entreprise, consentements — décisions prises et écarts à confirmer

Implémenté sans changer le contrat (OpenAPI, proto, table des routes). Points où j'ai dû **choisir** : à confirmer.

1. **Compte sans mot de passe (Google) : changement d'e-mail REFUSÉ (403).** Le contrat exige alors une « connexion récente », mais `RequestEmailChangeRequest` ne porte aucune preuve de cela. Pour l'autoriser il faut un champ proto additif (ex. `recent_login`) rempli par le gateway depuis `iat` du jeton : **changement de contrat, non fait**.
2. **Le jeton d'accès déjà émis reste valable jusqu'à 15 min** après un changement d'e-mail (JWT vérifié localement, G2). Les jetons de rafraîchissement sont tous révoqués. Même limite que le changement de mot de passe.
3. **`GET /me/company` d'un particulier : 404 ; `GET /me/consents` sans consentement : 404** — le contrat ne déclare pas ce code pour ces routes (seulement 401/403). Je n'invente pas de réponse vide. Alternative : l'ajouter au contrat.
4. **`PUT /me/company` d'un particulier : 409** (déclaré au contrat). Seules les données **légales** (raison sociale, identifiant fiscal, pays) repassent le statut à `pending` ; la fonction (`jobTitle`) n'y touche pas.
5. **Consentements** : journal **append-only** (aucune mise à jour). `necessary` doit être `true` (sinon 422). Version de politique : `^[A-Za-z0-9._-]{1,32}$`, libre. L'adresse IP est **tronquée** (IPv4 /24, IPv6 /48) avant écriture. Un visiteur sans `anonymousId` en reçoit un généré. **Durée de conservation de la preuve : non fixée** ; si le compte est supprimé, la preuve est conservée détachée (`user_id → NULL`). À décider avec le RGPD.
6. **Nouveaux champs** (additifs, déjà prévus au contrat) : `users.phone` (E.164), `companies.job_title`, `one_time_tokens.payload` (nouvelle adresse), table `consents` — migration `00008`.
7. **Réponse uniforme** à `POST /me/email` si la nouvelle adresse existe déjà : 202, aucun e-mail. Le temps de réponse n'est pas parfaitement égalisé (pas de jeton créé dans ce cas).
8. **E-mails** : `email_change_requested` part vers la **nouvelle** adresse, `email_change_confirmed` vers l'**ancienne** (avec la nouvelle adresse et la date), en fr/en/es. Le lien pointe vers `/confirm-email-change?token=…` du site : **cette page n'existe pas encore côté vitrine**.
9. **Inscription** : `phone` du formulaire d'inscription n'est pas encore transmis à la création du compte.
