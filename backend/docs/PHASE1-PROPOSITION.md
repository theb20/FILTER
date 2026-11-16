# FilterFinder : Phase 1 — audit du frontend et proposition d'architecture

Statut : **Phase 1 validée sous réserve** — voir [DECISIONS-PHASE2.md](DECISIONS-PHASE2.md) pour les décisions prises, le changement de périmètre (produits industriels génériques) et les écarts avec ce document (pagination par pages, `fulfillment`, remises volume, statut des comptes pro).
Périmètre : lecture de tout `src/`, inventaire des données fictives et des actions non connectées,
architecture, schémas de données, contrat d'API (version tableau ; le `openapi.yaml` sera généré
en phase 2 après validation).

---

## 0. État du dépôt (à trancher avant la phase 2)

- Le frontend n'est **pas** dans un dossier `frontend/` : il est à la **racine** du dépôt
  (`src/`, `public/`, `package.json`, `vite.config.js`, `index.html`…) sur la branche `feat/redesign-b2b`.
- Branche actuelle : `init-backend`. Elle contient déjà tout le frontend à la racine, plus un
  `backend/` non suivi (`go.work`, `pkg/`).
- Pour obtenir les 3 dossiers demandés, il faut **déplacer** le frontend dans `frontend/`
  (`git mv`, un seul commit « chore: move frontend to frontend/ »). Conséquences : chemins des
  workflows CI/déploiement, `.gitignore`, alias `@` (`jsconfig.json` + `vite.config.js`), `node_modules`
  à réinstaller. Aucun code n'est modifié.
- Chemin de module Go retenu : `github.com/theb20/FILTER/backend/<service>` (fichier `go.work`).

---

## 1. Audit du frontend

### 1.1 Routes

| Route | Contenu actuel | État |
|---|---|---|
| `/` | Accueil : recherche, 9 secteurs, marques/partenaires, avis, histoire | Données en dur (`home.jsx`) |
| `/category?cat=&q=` | Liste produits, filtres catégorie / marque / en stock, recherche texte | Filtrage **côté client** sur 8 produits |
| `/card?id=` | Fiche produit détaillée (galerie, prix, paliers, stock, avis, FAQ, associés) | Mocks |
| `/brands` | Page « Toutes les marques » | **Vide** (`InfoPage` « à compléter ») |
| `/basket` | Panier complet (lignes, promo, zone, TVA, devis) | `localStorage` + mocks |
| `/sign`, `/reset` | Connexion, inscription (particulier/entreprise), mot de passe oublié (3 étapes) | Formulaires sans backend |
| `/profil` | Espace client (infos + commandes) | `USER` et `ORDERS` en dur |
| `/quote` | Demande de devis | Formulaire local, `setSubmitted(true)` |
| `/contact` | Formulaire de contact (entreprise/particulier, pièce jointe, RGPD) | Formulaire local |
| `/blog`, `/blog/:slug` | Blog complet (filtres, pagination, article) | `blogMocks.js` |
| `/terms`, `/refund` | Pages légales | **Vides** (« à compléter ») |
| `/privacy` | Politique de confidentialité (texte complet, FR seulement) | Texte en dur |

### 1.2 Inventaire des données fictives → service cible

| Donnée fictive (fichier) | Contenu | Service cible |
|---|---|---|
| `data/demoProducts.js` | 8 produits : id, marque, référence, catégorie, prix XOF, `inStock`, `crossRefs` ; 5 catégories | `catalog` |
| Clés i18n `prod.<id>`, `cat.<slug>`, `stock.*` (`locales/*.js`) | **Noms de produits et de catégories stockés comme traductions du frontend** | `catalog` (traductions) |
| `data/cartMocks.js` → `CART_PRODUCT_MOCKS` | stock, délai d'expédition (24 h / 48 h / sur commande), 3 caractéristiques (dimensions, filetage, micronnage), compatibilités, paliers de remise, image | `inventory`, `catalog`, `pricing` |
| `cartMocks.js` → `SHIPPING_ZONES` | 3 zones (Abidjan, intérieur, international), frais, délais en jours ouvrés, éligibilité livraison gratuite | `shipping` |
| `cartMocks.js` → `PROMO_CODES` | 3 codes (%, montant fixe + minimum, expiré) | `pricing` |
| `cartMocks.js` → `VAT_RATE`, `FREE_SHIPPING_THRESHOLD`, `QUOTE_THRESHOLDS`, `LOW_STOCK_THRESHOLD`, `DEMO_INITIAL_CART` | TVA 18 %, seuils fictifs | `pricing`, `shipping`, `quote`, `inventory` |
| `config/currency.js` | XOF (base), EUR (parité fixe 655,957), **USD taux fictif 600** | `pricing` |
| `data/blogMocks.js` | 14 articles, 8 étiquettes, sélections éditoriales (à la une, sujets, guides, dossiers, plus lus, produits cités), biographie d'auteur, `fetchBlogArticles()` (point d'entrée API déjà isolé) | `content` |
| `data/productMocks.js` | 6 avis fictifs (identiques pour tous les produits) | `content` (avis) |
| `config/reviews.js` | `REVIEWS = []`, résumé `null` (règle : avis réels uniquement) | `content` |
| `config/site.js` | nom, éditeur, téléphone, e-mail, adresse, LinkedIn/Facebook/YouTube, `reviewsUrl`, `regionalSite` ; **`CLAIMS`** : « +5000 références croisées », « 1000 de stock », « fondée en 2011 » | `content` (réglages du site) ; les chiffres devraient être **calculés** depuis `catalog`/`inventory` |
| `pages/home.jsx` → `SECTORS` | 9 secteurs + image | `catalog` |
| `pages/profil.jsx` → `USER`, `ORDERS` | profil et 3 commandes fictives | `identity`, `order` |
| Marques « Brand A à D » | dérivées des produits ; page `/brands` vide | `catalog` |
| Textes légaux, histoire de l'entreprise | `/privacy` en dur (FR) ; CGV, remboursement, histoire = « à compléter » | `content` |

### 1.3 Actions « Mode démonstration : cette action n'est pas encore connectée »

| # | Action | Fichier | Cible |
|---|---|---|---|
| 1 | Création de compte (particulier / entreprise) | `AccountForm.jsx` | `identity` |
| 2 | Connexion e-mail + mot de passe | `sign.jsx` | `identity` |
| 3 | Connexion Google | `sign.jsx` | `identity` (OAuth) |
| 4 | Mot de passe oublié (e-mail → code → nouveau mot de passe) | `reset.jsx` | `identity` + `notification` |
| 5 | Inscription à la newsletter | `Newsletter.jsx` | `notification` |
| 6 | Chat « Discuter avec nous » | `TopBar.jsx` | `support` |
| 7 | « Commander » | `basket.jsx` | `order` + `payment` |
| 8 | « Télécharger le devis (PDF) » | `basket.jsx` | `quote` |
| 9 | « Modifier » le profil | `profil.jsx` | `identity` |

Formulaires **sans message de démo mais purement locaux** (ils affichent un succès sans rien envoyer) :
`/quote` (devis) et `/contact` (contact) → à connecter à `quote` et `support`.
La recherche (`SearchBar`, filtres de `/category`) est **côté client** → à passer sur `search`.
Le bandeau cookies stocke le consentement dans `ff_cookie_consent` → à journaliser côté serveur (RGPD).
`GeoBanner` (site régional) dépend de la géolocalisation : `regionalSite` est `null` aujourd'hui.
La bulle d'aide `HelpBubble` n'a pas été auditée en détail (aucun appel réseau trouvé).

### 1.4 Stockage navigateur actuel

| Clé | Contenu | Devient |
|---|---|---|
| `ff_cart_v1` | `{ items:[{id, quantity, priceAtAdd}], saved:[], promoCode, zone, note }` | panier serveur ; fusion à la connexion (§4.3) |
| `ff_lang` | fr / en / es | préférence (profil si connecté) |
| `ff_currency` | XOF / EUR / USD | préférence (profil si connecté) |
| `ff_cookie_consent` | préférences cookies | `consents` côté serveur + local |

### 1.5 Règles de calcul déjà codées dans le frontend

À reproduire **côté serveur** (le serveur ne doit jamais faire confiance aux montants du navigateur).
Les valeurs numériques sont, pour la plupart, des **valeurs de démonstration** : voir §8.

- Quantité : entier ≥ 1, plafonné au stock (min 1 même en rupture).
- Remise volume : palier le plus élevé atteint ; prix unitaire remisé arrondi au franc.
- Sous-total = Σ (prix unitaire remisé × quantité) ; économie = prix plein − remisé.
- Code promo : ordre de contrôle **inconnu → expiré (fin du jour UTC) → minimum d'achat → valide** ;
  remise en % arrondie, ou montant fixe plafonné au sous-total.
- Livraison : gratuite si la zone est éligible **et** (sous-total − remise) ≥ seuil ; sinon frais de zone ; 0 si panier vide.
- TVA = arrondi(HT × taux) ; TTC = HT + TVA ; les prix catalogue sont **HT**.
- Stock « faible » si ≤ 5 ; commande bloquée si un article est en rupture.
- Prix modifié depuis l'ajout : comparaison `priceAtAdd` ≠ prix actuel.
- Livraison estimée : jours ouvrés (lun.–ven.) ajoutés à la date du jour.
- Devis accessible si **≥ 10 articles OU ≥ 200 000 F CFA HT** (règle que tu as indiquée).

---

## 2. Architecture proposée

```
                       ┌───────────────┐          ┌──────────────────┐
  frontend (Vite SPA) ─►│               │          │ backoffice       │
                       │    gateway    │◄─────────┤ (Next.js, BFF)   │
  mobile/partenaires ──►│  REST /v1     │  JWT staff└──────────────────┘
                       └──────┬────────┘
              gRPC (mTLS)     │
  ┌────────┬────────┬────────┬─┴──────┬────────┬────────┬────────┬──────────┐
  │identity│catalog │ search │pricing │  cart  │ order  │ quote  │ payment  │
  └────────┴────────┴────────┴────────┴────────┴────────┴────────┴──────────┘
  ┌───────────┬────────┬──────────────┬─────────┐
  │fulfillment│content │ notification │ support │
  └───────────┴────────┴──────────────┴─────────┘
       │ Postgres par service │ Redis (cache, sessions, rate limit)
       │ NATS JetStream (événements, outbox) │ S3-compatible │ Meilisearch
       └── OpenTelemetry (traces, métriques) ─► collecteur ; logs JSON
```

### 2.1 Découpage en services (13 déployables)

Je garde **toutes les responsabilités** demandées. Seul regroupement proposé :
**`inventory` + `shipping` → `fulfillment`**. Justification : les deux reposent sur le même modèle
d'entrepôt et de zone ; le délai promis au client (« expédié sous 24 h » + durée de livraison)
combine stock et transport ; les séparer imposerait un appel entre services à chaque calcul de panier.
Les deux restent des **modules distincts** (domain/application/adapters) et peuvent être scindés plus tard.
Si tu préfères 14 services séparés, c'est possible, mais plus coûteux à exploiter.

| Service | Responsabilité | Données propres |
|---|---|---|
| `gateway` | Entrée unique, routage, JWT, rate limiting (Redis), CORS, `/v1`, idempotence, agrégation | aucune |
| `identity` | Comptes clients et **personnel** (pools séparés), sessions, OAuth Google, reset, RBAC, entreprises, consentements, journal d'audit | PostgreSQL |
| `catalog` | Produits, attributs techniques typés, marques, catégories, secteurs, machines/compatibilités, références croisées, médias, traductions | PostgreSQL |
| `search` | Index (SKU, référence, modèle, réf. croisée), facettes, tolérance aux fautes, journal des requêtes (recherches sans résultat) | Meilisearch + PostgreSQL (journal) |
| `pricing` | Devises, taux de change, prix par devise, remises volume, codes promo, règles de taxe, validation N° TVA/SIRET | PostgreSQL |
| `cart` | Panier serveur, « plus tard », zone, promo, note, fusion invité → connecté | PostgreSQL + Redis |
| `order` | Commandes, statuts, historique, factures (numérotation), remboursements, prix figés | PostgreSQL |
| `quote` | Demandes de devis, éligibilité, devis commerciaux (PDF), conversion devis → commande | PostgreSQL |
| `payment` | Abstraction multi-prestataires, intentions de paiement, webhooks signés, remboursements | PostgreSQL |
| `fulfillment` | Entrepôts, stock multi-entrepôts, réservations ; zones, tarifs, transporteurs, estimation, suivi | PostgreSQL |
| `content` | Blog, avis (modération), pages légales, histoire, FAQ, réglages du site, traductions | PostgreSQL |
| `notification` | E-mails/SMS transactionnels, newsletter (double opt-in), modèles multilingues, liste de suppression | PostgreSQL |
| `support` | Contact, pièces jointes, chat, tickets | PostgreSQL + S3 |

### 2.2 Choix techniques et justification

- **Hexagonal** par service : `internal/domain`, `internal/application`, `internal/adapters/{http,grpc,postgres,nats,…}`, `cmd/<service>/main.go`.
- **Communication** : REST + OpenAPI uniquement au gateway ; **gRPC** (protobuf, `buf`) entre services ; **événements** sur **NATS JetStream**.
  NATS plutôt que Kafka : opérations beaucoup plus simples, suffisant à ce volume ; l'abstraction `EventBus` dans `pkg/`
  permet de migrer vers Kafka si le volume l'exige.
- **Outbox** (table `outbox` dans la même transaction) pour `order`, `payment`, `inventory`, `quote` ; consommateurs **idempotents** (table `processed_events`).
- **Saga** de commande : `order.placed` → `fulfillment` réserve le stock → `payment` crée l'intention → `payment.succeeded` → `order.paid` → `fulfillment` confirme / `notification` envoie. Échec ou expiration → compensation (libération du stock, annulation).
- **Recherche** : **Meilisearch** (typos, facettes, simple à exploiter, bon pour ~5 000+ références). OpenSearch si tu as besoin d'analytique avancée.
- **Migrations** : `goose` (SQL versionné), une base **par service**, aucun accès croisé.
- **Cache / sessions** : Redis (panier chaud, rate limiting, jetons).
- **Fichiers** : stockage objet compatible S3 (MinIO en local), URL pré-signées, antivirus avant mise à disposition.
- **Tests** : unitaires + intégration (`testcontainers-go`) ; seed reproduisant les 8 produits, 14 articles, 3 codes promo, 3 zones, taux démo.
- **Observabilité** : OpenTelemetry, logs JSON (`slog`), `/healthz` et `/readyz` par service.
- **Déploiement** : Docker Compose (local), manifests Kubernetes (prod, `kustomize`), CI (lint `golangci-lint`, tests, build, contrôle OpenAPI/proto).

### 2.3 Sécurité

- Clients : JWT d'accès **10 à 15 min** + jeton de rafraîchissement à **rotation** (famille de jetons, détection de rejeu).
- Mots de passe : **Argon2id**. Personnel : pool et émetteur séparés, **2FA TOTP** obligatoire (WebAuthn en option).
- RBAC par permissions fines (`orders:refund`, `catalog:write`…), vérifié au gateway **et** dans chaque service.
- Idempotence : en-tête `Idempotency-Key` obligatoire sur `POST /v1/orders` et `POST /v1/payments`.
- Validation stricte des entrées (schéma OpenAPI au gateway), limites de taille, secrets hors code (variables d'environnement / gestionnaire de secrets).
- Journal d'audit **append-only** : chaque service publie des événements `audit.*` ; `identity` les stocke et les expose au backoffice.

---

## 3. Données

### 3.1 Conventions communes

- Identifiants : **UUID v7** (triables). Dates : `timestamptz` en **UTC**. Pays : ISO 3166-1 alpha-2. Téléphones : **E.164**.
- **Argent** : `amount_minor BIGINT` + `currency CHAR(3)` (ISO 4217). Jamais de `float`.
  XOF et XAF ont 0 décimale (1 unité mineure = 1 franc) ; EUR/USD ont 2.
  Dans l'API, le montant est une **chaîne** : `{ "amount": "77000", "currency": "XOF" }`.
- Textes traduisibles : tables `*_i18n(entity_id, locale, …)` (`locale` BCP 47 : `fr`, `en`, `es`…), avec repli sur la langue par défaut.
- Chaque table : `created_at`, `updated_at`, `version` (concurrence optimiste) ; suppression logique là où la loi l'impose.

### 3.2 Schémas par service (tables principales)

**identity**
`users(id, email citext unique, password_hash, status, locale, preferred_currency, email_verified_at)` ·
`profiles(user_id, kind[individual|company], first_name, last_name, phone_e164, company_id)` ·
`companies(id, legal_name, tax_id, tax_id_status, country, billing_address jsonb)` ·
`oauth_identities(provider, subject, user_id)` ·
`refresh_tokens(id, user_id, family_id, token_hash, expires_at, revoked_at, ip, user_agent)` ·
`password_resets(user_id, code_hash, expires_at, attempts, used_at)` ·
`staff_users(id, email, password_hash, totp_secret_enc, status)` · `roles`, `permissions`, `role_permissions`, `staff_roles` ·
`consents(subject_id, purpose, granted, policy_version, recorded_at, ip)` ·
`data_requests(user_id, kind[export|erasure], status)` · `audit_log(actor_id, action, entity, entity_id, before jsonb, after jsonb, ip, at)`.

**catalog**
`products(id, sku unique, brand_id, category_id, status)` · `product_i18n(product_id, locale, name, description)` ·
`attribute_defs(key, value_type, unit)` + `attribute_def_i18n` · `product_attributes(product_id, key, value_num, value_text)`
(dimensions Ø/longueur, filetage, micronnage…) ·
`brands(id, slug, name, logo_key)` · `categories(id, slug, parent_id)` + `category_i18n` · `sectors(id, slug, image_key)` + `sector_i18n` ·
`product_sectors` · `machines(id, make, model)` · `product_compatibilities(product_id, machine_id)` ·
`cross_references(product_id, ref_brand, ref_number, ref_number_normalized, kind[oem|equivalent])` ·
`media(product_id, object_key, sort, alt)` + `media_i18n`.

**search** : index `products` (SKU, noms par langue, références croisées normalisées, marque, catégorie, secteurs, machines, attributs, en stock) ;
`search_queries(id, q, locale, results_count, user_id nullable, at)` (alimente « recherches sans résultat »).

**pricing**
`currencies(code, minor_exponent, enabled)` · `fx_rates(base, quote, rate numeric(20,10), source, fetched_at)` ·
`prices(product_id, currency, amount_minor, valid_from, valid_to)` ·
`volume_discounts(scope[product|category], scope_id, min_qty, percent_bp, valid_from, valid_to)` ·
`promo_codes(code unique, kind[percent|fixed], value, currency nullable, min_subtotal_minor, starts_at, expires_at, max_uses, per_user_limit, active)` + `promo_redemptions` ·
`tax_rules(country, customer_kind, rate_bp, valid_from, valid_to, exemption_reason, reverse_charge)` ·
`tax_id_checks(tax_id, country, valid, checked_at, raw jsonb)`.

**cart**
`carts(id, user_id nullable, guest_token nullable, currency, zone_code, promo_code, note, status, updated_at)` ·
`cart_items(cart_id, product_id, quantity, price_at_add_minor, currency)` · `saved_items(cart_id, product_id, quantity, price_at_add_minor)`.

**order**
`orders(id, number unique, user_id, company_id, status, currency, subtotal_minor, discount_minor, shipping_minor, tax_minor, total_minor, fx_rate_id, customer_reference, note, billing_address jsonb, shipping_address jsonb, idempotency_key unique, placed_at)` ·
`order_items(order_id, product_id, sku, name_snapshot, quantity, unit_minor, discount_minor, tax_rate_bp)` ·
`order_status_history(order_id, from, to, actor, at)` · `invoices(number unique, order_id, issued_at, pdf_key)` · `refunds(order_id, amount_minor, reason, status)` · `outbox`.

**quote**
`quote_requests(id, user_id nullable, company, contact_name, email, phone_e164, notes, status, source[form|cart])` ·
`quote_request_lines(request_id, reference_text, description, quantity, product_id nullable)` ·
`quotes(id, request_id, number, currency, valid_until, subtotal_minor, tax_minor, total_minor, status, pdf_key)` + `quote_lines` ·
`quote_rules(min_items, min_amount_minor, currency, active)`.

**payment**
`payment_intents(id, order_id, provider, method, amount_minor, currency, status, idempotency_key unique, provider_ref)` ·
`provider_events(provider, event_id unique, payload jsonb, processed_at)` · `refunds(…)` · `provider_configs(provider, country, enabled)`.

**fulfillment**
`warehouses(id, code, country, region)` · `stock_levels(product_id, warehouse_id, on_hand, reserved, CHECK reserved ≤ on_hand)` ·
`stock_movements(id, product_id, warehouse_id, delta, reason, ref, at)` · `reservations(id, owner_ref, product_id, warehouse_id, quantity, expires_at, status)` ·
`zones(id, code, countries[], region)` · `shipping_rates(zone_id, carrier_id, fee_minor, currency, free_threshold_minor, eta_min_days, eta_max_days)` ·
`carriers(id, name)` · `shipments(order_id, carrier_id, tracking_no, status)` + `tracking_events` · `pickup_points(id, name, address)`.

**content**
`authors(id, name)` + `author_i18n(bio)` · `articles(id, slug unique, category, kind, author_id, status, published_at, read_minutes, image_key)` ·
`article_i18n(article_id, locale, title, excerpt, body, seo_title, seo_description)` · `tags` + `tag_i18n` + `article_tags` ·
`curated_collections(key[mosaic|topics|guides|dossiers|popular|cited_products], position, article_id|product_id)` ·
`reviews(id, product_id, user_id nullable, order_item_id nullable, rating 1..5, title, body, status[pending|approved|rejected], created_at)` ·
`pages(slug, version, effective_at)` + `page_i18n(body)` (CGV, confidentialité, remboursement, histoire) ·
`site_settings(key, value jsonb)` (+ i18n) (contacts, réseaux sociaux, partenaires) · `faq_items` + `faq_i18n`.

**notification**
`templates(key, channel, locale, subject, body, version)` · `messages(id, channel, recipient, template_key, locale, payload, status, provider_ref, attempts)` ·
`newsletter_subscribers(email, locale, status[pending|confirmed|unsubscribed], consented_at, source, token_hash)` · `suppressions(address, reason)`.

**support**
`contact_messages(id, kind[company|individual], company, tax_id, name, email, phone_e164, message, consent_at, status)` ·
`attachments(id, object_key, mime, size, av_status)` · `tickets(id, contact_id, status, assignee_id)` + `ticket_messages` · `chat_sessions` + `chat_messages`.

---

## 4. Contrat d'API v1 (gateway)

### 4.1 Conventions

- Base : `/v1`. Auth : `Authorization: Bearer <JWT>` (clients) ; `/v1/admin/**` réservé au personnel (autre émetteur, autre audience).
- Langue : `Accept-Language` (ou `?lang=`) ; devise : `?currency=` (sinon préférence du profil, sinon XOF).
- Erreurs : **RFC 9457** `application/problem+json` (`type`, `title`, `status`, `detail`, `errors[]`).
- Pagination : `?limit=&cursor=` (curseur opaque) ; tri `?sort=`. En-têtes `RateLimit-*`.
- Idempotence : `Idempotency-Key` sur les `POST` sensibles ; `ETag`/`If-Match` sur les mises à jour.
- Montants : objet `Money` `{ "amount": "<entier mineur>", "currency": "XOF" }`.

### 4.2 Endpoints publics et clients

| Tag | Méthode et chemin | Rôle |
|---|---|---|
| Recherche | `GET /search?q&cat&brand&sector&inStock&page` | résultats + facettes + corrections |
| | `GET /search/suggest?q` | suggestions |
| Catalogue | `GET /products`, `GET /products/{idOrSku}` | liste filtrée / fiche complète (attributs, compat., réf. croisées, prix dans la devise, disponibilité) |
| | `GET /products/{id}/related` | produits associés |
| | `GET /categories`, `/brands`, `/brands/{slug}`, `/sectors` | référentiels |
| | `GET /machines/makes`, `/machines/models?make=` | « Trouver mon filtre » |
| Prix | `GET /currencies`, `GET /fx-rates` | devises et taux horodatés |
| Livraison | `GET /shipping/zones`, `POST /shipping/estimate` | zones, frais, date estimée |
| | `GET /products/{id}/availability?zone=` | stock et délai d'expédition |
| Panier | `GET /cart` | panier + **totaux calculés serveur** (HT, TVA, TTC, livraison gratuite restante, éligibilité devis, alertes prix/rupture) |
| | `POST /cart/items`, `PATCH /cart/items/{productId}`, `DELETE /cart/items/{productId}` | lignes |
| | `POST /cart/items/{productId}/save-for-later`, `POST /cart/saved/{productId}/move-to-cart`, `DELETE /cart/saved/{productId}` | « plus tard » |
| | `PUT /cart/zone`, `POST /cart/promo`, `DELETE /cart/promo`, `PUT /cart/note` | zone, promo, note |
| | `POST /cart/merge` | fusion du panier invité (§4.3) |
| Commande | `POST /orders` (Idempotency-Key) | passer commande depuis le panier |
| | `GET /orders`, `GET /orders/{id}`, `GET /orders/{id}/invoice`, `POST /orders/{id}/cancel` | espace client |
| Paiement | `GET /payment-methods?country`, `POST /payments`, `GET /payments/{id}` | initier / suivre |
| | `POST /webhooks/payments/{provider}` | webhooks signés (public) |
| Devis | `POST /quote-requests` | demande (formulaire `/quote`) |
| | `POST /cart/quote-request` | demande depuis le panier |
| | `GET /quote-requests`, `GET /quotes/{id}`, `GET /quotes/{id}/pdf`, `POST /quotes/{id}/accept` | suivi, PDF, conversion en commande |
| | `GET /quote-eligibility` | seuils actuels |
| Compte | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | |
| | `GET /auth/oauth/google/start`, `/auth/oauth/google/callback` | |
| | `POST /auth/password/forgot`, `/auth/password/verify-code`, `/auth/password/reset` | 3 étapes de `/reset` |
| | `GET`/`PATCH /me`, `GET`/`PUT /me/company`, `GET`/`PUT /me/consents` | profil |
| | `POST /me/data-export`, `DELETE /me` | RGPD (export, suppression) |
| Contenu | `GET /blog/home`, `GET /blog/articles`, `GET /blog/articles/{slug}`, `GET /blog/tags` | blog |
| | `GET /products/{id}/reviews`, `POST /products/{id}/reviews` | avis (modérés) |
| | `GET /pages/{slug}`, `GET /faq`, `GET /site-settings` | légal, histoire, FAQ, réglages |
| Newsletter | `POST /newsletter/subscribe`, `GET /newsletter/confirm`, `POST /newsletter/unsubscribe` | double opt-in |
| Support | `POST /contact` (multipart), `POST /support/uploads` | contact + pièce jointe |
| | `POST /chat/sessions`, `GET /chat/sessions/{id}/ws` (WebSocket) | chat |
| Consentement | `POST /consents` | bandeau cookies (journalisé) |
| Technique | `GET /healthz`, `GET /readyz` | sondes |

### 4.3 Migration du panier `ff_cart_v1` sans perte

1. Invité : le navigateur garde `ff_cart_v1`. Au premier appel panier, le serveur crée un panier invité (`guest_token` en cookie `HttpOnly`).
2. À la connexion, le frontend envoie `POST /cart/merge` avec `{ items:[{productId, quantity}], saved:[…], promoCode, zone, note }`.
3. Le serveur **ignore** `priceAtAdd` du navigateur, reprend le prix serveur, **additionne** les quantités avec le panier existant (plafonnées au stock) et renvoie le panier fusionné avec les alertes (`priceChanged`, `outOfStock`).
4. Seulement après la réponse `200`, le frontend vide `ff_cart_v1`. En cas d'échec réseau, la copie locale est conservée.

### 4.4 Backoffice (`/v1/admin/**`, personnel uniquement)

Groupes d'endpoints CRUD, tous paginés/filtrables/triables côté serveur, tous audités :
`products`, `attributes`, `cross-references` (import/export CSV), `brands`, `categories`, `sectors`, `inventory` (stock par entrepôt, mouvements),
`prices`, `discounts`, `promo-codes`, `currencies`, `fx-rates`, `tax-rules`, `shipping` (zones, tarifs, transporteurs),
`orders` (statuts, remboursement), `quotes` (réponse, conversion), `payments`, `invoices`,
`customers`, `companies`, `contact-messages`, `tickets`, `chat`,
`articles`, `reviews` (modération), `pages`, `site-settings`, `translations`,
`staff-users`, `roles`, `audit-log`, `settings`, et `GET /admin/dashboard/*` (CA par devise/pays/période, commandes, devis en attente, conversion, ruptures, paiements en échec, top produits, **top recherches sans résultat**).

---

## 5. Raccordement du frontend (aperçu pour la phase 7)

- Un client API **typé généré depuis l'OpenAPI** remplace les mocks fichier par fichier :
  `fetchBlogArticles()` (déjà isolé), `DEMO_PRODUCTS`, `CART_PRODUCT_MOCKS`, `PROMO_CODES`, `SHIPPING_ZONES`, `DEMO_REVIEWS`, `SITE`/`CLAIMS`…
- Les calculs de `lib/cartMath.js` et `lib/cartTotals.js` deviennent un **affichage** : le serveur fait foi (le panier renvoie les totaux).
- Les noms de produits ne seront plus des clés `prod.<id>` du frontend mais viendront de `catalog` selon la langue.
- UI inchangée ; les messages « Mode démonstration » sont retirés **action par action**, une fois chacune réellement connectée.

---

## 6. Phasage (rappel)

1. **Ce document** (audit, architecture, schémas, contrat).
2. Socle : `pkg/`, gateway, identity, Docker Compose, CI.
3. catalog, search, fulfillment (stock), pricing.
4. cart, order, quote, fulfillment (livraison), payment.
5. content, notification, support.
6. Backoffice.
7. Raccordement du frontend, tests de bout en bout, README par service.

---

## 7. Hypothèses prises (à corriger si besoin)

1. Prix catalogue **HT**, stockés en XOF (devise de base), les autres devises étant dérivées des taux.
2. Un seul pays de TVA au lancement : Côte d'Ivoire à 18 %. Les autres pays sont modélisés mais **non configurés**.
3. NATS JetStream pour les événements, Meilisearch pour la recherche, `goose` pour les migrations, `buf` pour protobuf.
4. Les avis ne sont affichés qu'après **modération** ; « achat vérifié » seulement si rattaché à une ligne de commande réelle.
5. Les nombres affichés sur le site (« +5000 références », « 1000 de stock », « fondée en 2011 ») deviendront des valeurs **calculées** ou des réglages saisis, jamais codés en dur.
6. Le personnel et les clients ont des **pools de comptes distincts** dans le même service `identity`.
7. Le journal d'audit est hébergé par `identity` (consommateur des événements `audit.*`).

---

## 8. Questions bloquantes (je n'invente aucune règle métier)

**Structure et infrastructure**
1. Valides-tu le déplacement du frontend dans `frontend/` (§0) ?
2. Cible de déploiement production (cloud, région initiale, Kubernetes managé ou non) ? Budget d'infrastructure ?
3. Valides-tu les choix : **NATS** (et non Kafka), **Meilisearch** (et non OpenSearch), et le regroupement `fulfillment` (13 services) ?

**Paiement et prestataires**
4. Quels prestataires de paiement pour la Côte d'Ivoire : agrégateur (CinetPay, Paystack, Flutterwave…) ou intégration directe Orange Money / MTN MoMo / Wave / Moov ? Cartes bancaires via quel prestataire ? Virement bancaire accepté ? Paiement à la livraison ? Paiement différé pour comptes pro ?
5. Prestataires e-mail et SMS, et WhatsApp Business API ou simple lien `wa.me` ?
6. Source des taux de change (BCE pour l'EUR ; fournisseur pour l'USD) ?

**Règles commerciales** (les valeurs actuelles du frontend sont fictives, sauf TVA 18 % et seuil de devis que tu as indiqués)
7. Marchés ouverts au lancement et règles de TVA par pays (exonérations B2B, autoliquidation) ; validation du N° TVA/SIRET via quel service ?
8. Les prix sont-ils saisis par devise, ou toujours dérivés du XOF ? Arrondi dans chaque devise ?
9. Vraies zones et tarifs de livraison, seuil de livraison gratuite, transporteurs, retrait sur place à Abidjan ?
10. Codes promo : cumulables ? limites d'usage ? Remises volume : par produit, par catégorie, ou les deux ?
11. Réservation de stock : durée avant libération ? Liste des entrepôts ? Les commandes en rupture sont-elles autorisées (« sur commande ») ?
12. Statuts de commande souhaités, numérotation légale des factures (format, remise à zéro annuelle), mentions obligatoires ?
13. Devis : durée de validité, circuit de réponse commerciale, qui peut le convertir en commande ?
14. Inscription « entreprise » : validation manuelle par ton équipe avant accès aux conditions pro ?

**Données et conformité**
15. Langues au lancement (fr / en / es ?) et qui traduit le catalogue et le blog ?
16. Durées de conservation des données personnelles, commandes, factures et journaux ; hébergement dans quelle région (loi ivoirienne 2013-450 + RGPD) ?
17. Protection anti-spam des formulaires publics (captcha) ? Fournisseur ?
18. Rôles internes du backoffice (liste et permissions par rôle) ; 2FA : TOTP seul ou clés de sécurité aussi ?
19. Catalogue réel : format et source de l'import initial (fichier Excel/CSV, ERP existant), volume exact (références, références croisées, marques) ?
