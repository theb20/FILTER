# Phase 1 — Moteur de recherche : proposition de contrat

Statut : **VALIDÉE sur le fond (v3)** — clé de fusion phase 1 et rétention de 13 mois de `zero_result_queries` confirmées. Les sources et les budgets suivront. Contrat OpenAPI + protos rédigés (`engine:*`, tags `Admin · Moteur de recherche`, `proto/sourcing`, `proto/searchadmin`). Source : « Spécification complète du backoffice » (section 2) et feuille de route (phase 1). Modèle retenu : **moteur multi-sources ET vente directe** (le contrat commerce déjà validé reste intact).

Révision v2 : relecture critique (budget non atomique, résultats bruts sans table, absence de délai global, table sans-résultat, bascule en boucle, devise, données personnelles) et valeurs recommandées.

Révision v3 : (1) réservations de budget **à échéance**, libérées par une tâche ; (2) texte normalisé calculé **après masquage**, rétention de `zero_result_queries` ; (3) « sans clic » détecté par une **tâche différée** ; (4) une source **appelée une seule fois** par recherche ; (5) **durée maximale de cache par source** ; (6) requêtes communicables tant que `user_id` existe ; (7) `min_results_to_stop` compté **après fusion des doublons**.

## 1. Périmètre de la phase 1

Six onglets, section « Moteur de recherche » :

| Onglet | Ce que l'on peut faire | Ce que l'on mesure |
|---|---|---|
| Sources et connecteurs | Ajouter, activer/désactiver, régler priorité / délai / tentatives, **tester** une source | santé, temps de réponse, taux d'erreur, dernière synchronisation |
| Requêtes | Journal des recherches, détail (sources interrogées, résultats bruts, résultat fusionné), **rejouer** | requête, langue, pays, durée, nb de résultats, clic ou non |
| Sans résultat | Recherches sans résultat ou sans clic, marquer « traité » | fréquence, tendance, statut |
| Cache et indexation | Durée de vie par type de donnée, vider le cache, relancer une indexation, suivre la file | taux de réponse depuis le cache, taille de l'index, tâches en attente |
| Coûts et quotas | Budget par source, plafond journalier, alertes, bascule vers une source de secours | coût par recherche, coût mensuel, quota restant |
| Santé des services | État de chaque service Go, de la base et de l'index, historique des incidents | disponibilité, latence, taux d'erreur |

**Hors phase 1** : créer un synonyme ou une fiche depuis « Sans résultat » (dépend de Synonymes et Produits, phases 2-3) — le bouton reste **désactivé** avec l'explication ; classement, dédoublonnage, enrichissement IA, sites exclus.

## 2. Services

| Service | Rôle | Statut |
|---|---|---|
| `search` (existe en brouillon) | Recherche, journal des requêtes, sans-résultat, cache, indexation, réglages | à compléter |
| `sourcing` (**nouveau**) | Connecteurs vers les sources externes : appels, délais, tentatives, **coûts**, quotas, santé, bascule | à créer |

`sourcing` isole tout ce qui parle à l'extérieur (clés d'API, coûts, pannes) : une source qui tombe ne fait pas tomber `search`. Un **disjoncteur** (circuit breaker) par source coupe les appels après des échecs répétés et les reprend par essais espacés.

## 3. Modèle de données

**Sources et budget**
- `sources(id, kind, name UNIQUE, base_url, enabled, priority, timeout_ms, max_retries, secret_ref, cost_per_call_micros, currency, allows_raw_storage, terms_checked_at, max_cache_ttl_seconds NULL, created_at)`
  - `kind` : `web_search_api` | `manufacturer_api` | `distributor_feed`.
  - **Les clés d'API ne sont jamais stockées** : seulement `secret_ref` (référence vers une variable d'environnement ou un coffre).
  - `cost_per_call_micros` : **estimation du coût maximal d'un appel**, servant à la réservation (voir §5.2).
  - `currency` : code ISO 4217 de facturation de la source (souvent USD) ; **tous les montants d'une source sont dans cette devise**.
  - `allows_raw_storage` : vaut `true` seulement après vérification des conditions d'usage de l'API (`terms_checked_at` renseigné, par un membre habilité).
  - `max_cache_ttl_seconds` : **durée maximale pendant laquelle les données de cette source peuvent être gardées en cache**, selon ses conditions d'usage (`NULL` = non renseignée : **aucune mise en cache des données de cette source** tant qu'elle n'est pas vérifiée). Durée effective = `min(durée de la politique de cache du type de donnée, max_cache_ttl_seconds)`.
- `source_quotas(source_id PK, monthly_budget_micros, daily_cap_micros, alert_threshold_pct, fallback_source_id NULL)` — devise = celle de la source.
- `source_usage(source_id, period_kind, period_start, spent_micros, reserved_micros, PK(source_id, period_kind, period_start))` — compteurs jour et mois, mis à jour **atomiquement** (voir §5.2).
- `source_reservations(id, source_id, call_id, amount_micros, status ('held'|'settled'|'released'|'expired'), created_at, expires_at, settled_at)` — **une ligne par réservation**, avec une **date d'expiration** (`created_at` + délai global + marge de 30 s). Un index partiel sur `status = 'held'` et `expires_at` sert la tâche de libération (voir §5.2).
- `source_calls(id, source_id, query_id, started_at, duration_ms, status, http_status, attempts, cost_micros, currency, cached, error_kind)` — une ligne par appel ; rétention 90 jours.

**Requêtes**
- `search_queries(id, text_masked, text_normalized, language, country, started_at, duration_ms, result_count, session_id, user_id NULL)` — `text_masked` : texte après masquage des e-mails et numéros de téléphone (§5.9) ; **`text_normalized` est toujours calculé À PARTIR de `text_masked`** (jamais du texte brut : sinon un e-mail pourrait survivre dans la forme normalisée) ; le texte brut n'est **jamais** stocké ; rétention 13 mois ; `user_id` effacé à 90 jours.
- `search_query_clicks(query_id, clicked_at, position)` — alimente « clic ou non » et « sans clic ».
- `search_query_sources(query_id, source_id, status, raw_count, duration_ms, from_cache)` — quelles sources ont répondu.
- `search_query_raw_results(query_id, source_id, payload (jsonb compressé), stored_at, expires_at)` — **table dédiée, rétention propre** (30 jours, valeur proposée), alimentée **uniquement** si `sources.allows_raw_storage`. Sans elle, « relire » n'est pas proposé pour cette source.
- `search_followups(query_id PK, due_at, done_at)` — **tâche différée** de détection « sans clic » : une ligne créée à chaque recherche ayant servi des résultats, échéance = `started_at` + 30 minutes.
- `zero_result_queries(id, kind ('no_result'|'no_click'), text_normalized, language, country, occurrences, first_seen_at, last_seen_at, status ('open'|'handled'), handled_by → staff_users, handled_at, expires_at, UNIQUE(kind, text_normalized, language, country))` — **vraie table**, mise à jour par *upsert* ; index sur `text_normalized`. `text_normalized` provient du texte **déjà masqué**. **Rétention : `expires_at` = `last_seen_at` + 13 mois** (valeur proposée), purge automatique ; une ligne « traitée » reste soumise à la même durée.

**Cache**
- `cache_policies(data_type PK, ttl_seconds CHECK (> 0))`.

Contraintes en base : unicité du nom de source ; `priority ≥ 0`, `max_retries ≥ 0`, `timeout_ms > 0`, `cost_per_call_micros ≥ 0` ; `daily_cap_micros ≤ monthly_budget_micros` ; `spent_micros ≥ 0` et `reserved_micros ≥ 0` ; devises ISO ; **bascule limitée à un seul saut** (voir §5.5) ; `handled_by` et `handled_at` renseignés ensemble ; `max_cache_ttl_seconds > 0` quand il est renseigné ; `source_reservations.amount_micros > 0` et `expires_at > created_at`.

## 4. Routes d'administration proposées

Toutes derrière `staffAuth`, avec `x-required-permission` ; toute écriture est journalisée dans l'audit.

| Route | Permission |
|---|---|
| `GET/POST /admin/sources`, `GET/PUT /admin/sources/{id}` | `engine:read` / `engine:write` |
| `POST /admin/sources/{id}/enable`, `/disable` | `engine:write` |
| `POST /admin/sources/{id}/test` (**consomme du budget**, journalisé) | `engine:write` |
| `GET /admin/sources/{id}/health` | `engine:read` |
| `GET/PUT /admin/sources/{id}/quota` | `engine:read` / `engine:budget` |
| `PUT /admin/sources/{id}/raw-storage` (autoriser le stockage des résultats bruts après vérification des conditions d'usage) | `engine:budget` |
| `GET /admin/source-costs?from&to&sourceId` (agrégats **par devise**) | `engine:read` |
| `GET /admin/search-queries` (curseur, filtres) ; `GET /admin/search-queries/{id}` ; `POST /admin/search-queries/{id}/replay` | `engine:read` ; replay : `engine:write` |
| `GET /admin/search-zero-results` ; `POST /admin/search-zero-results/{id}/handle` | `engine:read` / `engine:write` |
| `GET/PUT /admin/search-cache-settings` ; `POST /admin/search-cache/purge` | `engine:read` / `engine:write` |
| `GET /admin/search-index/status` ; `POST /admin/search-index/reindex` (tâche asynchrone → `/admin/jobs`) | `engine:read` / `engine:write` |
| `GET /admin/service-health` | `engine:read` |
| Export CSV : `?format=csv` sur requêtes, sans-résultat, coûts | `engine:export` |

Nouvelles permissions : `engine:read`, `engine:write`, `engine:budget`, `engine:export`. La spécification demande lire / créer / modifier / supprimer / exporter par onglet ; je propose de **ne pas** éclater en cinq permissions par onglet (matrice ingérable, erreurs de configuration) mais de garder `read` / `write` / `export` + `budget` pour ce qui engage de l'argent. **À valider.**

## 5. Règles proposées

1. **Désactiver, jamais supprimer** une source (l'historique la référence).
2. **Réservation atomique du budget avant chaque appel, à échéance.** Avant d'appeler une source, **une seule transaction** (a) réserve le montant dans `source_usage` par une instruction conditionnelle `UPDATE … SET reserved = reserved + :c WHERE … AND spent + reserved + :c <= plafond (jour ET mois)` puis (b) insère **une ligne `source_reservations`** (`held`, `expires_at`). Si l'`UPDATE` ne touche aucune ligne, l'appel n'a **pas lieu**. Après l'appel, dans une transaction : la réservation passe à `settled`, `spent += coût réel`, `reserved -= montant réservé` ; si l'appel échoue sans facturation, elle passe à `released`. **Tâche de libération** (toutes les minutes) : toute réservation `held` dont `expires_at` est dépassé passe à `expired`, et son montant est retiré de `reserved` — un plantage du service ne bloque donc jamais une source à tort. Une **tâche de réconciliation** (toutes les heures) recalcule `reserved` à partir des lignes `held` non expirées et corrige toute dérive. Plusieurs appels simultanés ne peuvent pas dépasser ensemble le plafond. **Dépassement résiduel assumé et documenté** : il vient de l'écart entre coût réel et estimation `cost_per_call_micros` (API facturant à la page), des facturations tardives de l'API et des réservations expirées dont l'appel a malgré tout été facturé ; il est borné par (appels en vol × écart maximal). À **80 %** du budget mensuel : alerte au backoffice ; au plafond : la source est suspendue et la bascule s'applique (règle 5). Un récapitulatif mensuel compare dépense interne et facture du fournisseur.
3. **Délai global par recherche : 4 500 ms** (valeur proposée, 4 à 5 s). Les sources sont interrogées par **niveaux de priorité** (valeur basse d'abord) ; les sources d'un même niveau en parallèle. Le niveau suivant n'est lancé que si le **nombre minimal de résultats** (réglage `min_results_to_stop`) n'est pas atteint. **Ce nombre est compté APRÈS fusion des doublons** : trois sources renvoyant la même pièce comptent pour un résultat. Dès que le seuil est atteint, on **arrête** les niveaux suivants. En phase 1, la fusion applique une **clé minimale** (référence normalisée + marque normalisée identiques) ; le dédoublonnage fin (similarité, choix de la source de référence par champ) est l'onglet Dédoublonnage de la phase 2. À l'expiration du délai global, on sert les résultats déjà reçus et on annule les appels en cours (leur réservation est libérée ou convertie au coût estimé si l'API a pu facturer).
4. **Délai par appel : 3 000 ms ; 2 tentatives**, **uniquement** sur erreurs réseau et réponses 5xx (jamais sur 4xx ni sur dépassement de quota de l'API). Les tentatives comptent dans le coût, restent dans le délai global et dans la réservation.
5. **Une source n'est appelée qu'UNE SEULE FOIS par recherche.** La recherche tient l'ensemble des sources déjà appelées ou planifiées ; la bascule vers `A.fallback` est **ignorée si cette source a déjà été appelée** (ou l'est à un autre niveau, ou est le secours d'une autre source déjà basculée) : on ne double ni l'appel, ni la réservation, ni le coût. Les résultats d'une source appelée au titre de secours sont fusionnés comme les autres.
   **Bascule : un seul saut, jamais en chaîne.** Si la source A est suspendue ou en panne, ses requêtes vont vers `A.fallback` ; **si cette source de secours est elle-même suspendue, hors budget ou en panne, on ne bascule pas plus loin** : la recherche continue avec les autres sources du niveau. La source de secours **réserve son propre budget** comme toute autre. En base : `fallback_source_id <> source_id` ; une source **déjà désignée comme secours** ne peut pas avoir de secours elle-même, et inversement (déclencheur qui refuse A→B puis B→A et toute chaîne A→B→C).
6. **Test d'une source** : coûte un appel réel (réservé et compté comme les autres), journalisé avec l'auteur, limité à 10 tests par heure et par membre.
7. **Rejouer une requête** : deux modes. « Relire » (gratuit) ne lit que `search_query_raw_results` et n'est proposé que si le stockage brut était autorisé pour la source et la rétention non dépassée ; « Refaire les appels » coûte (réservation habituelle) et produit une **nouvelle** ligne de journal, l'originale n'est pas modifiée.
8. **Cache** : durée de vie par type de donnée ; purge totale ou par type ; la purge est auditée.
9. **Données personnelles.** Une requête est un texte libre : on **ne peut pas garantir** qu'il ne contient rien d'identifiant. Mesures : masquage automatique à l'enregistrement des **adresses e-mail** et des **numéros de téléphone** (remplacés par `[e-mail]` / `[tel]`) **avant toute autre opération** (la forme normalisée en dérive) ; le texte brut n'est jamais stocké ; pas d'adresse IP ; session anonyme ; `user_id` effacé à 90 jours. **Communicabilité** : tant que `user_id` existe (90 jours), les requêtes d'un client sont des données personnelles rattachables : elles sont **incluses** dans l'export et l'effacement d'un client ; **après 90 jours elles sont anonymes** (non rattachables à une personne) et ne sont plus communiquées. Ces mesures réduisent le risque sans l'annuler ; la mention figurera dans la politique de confidentialité.
10. **Sans résultat / sans clic.** *Sans résultat* (`kind = no_result`) : détecté **à la recherche**, si `result_count = 0`. *Sans clic* (`kind = no_click`) : détecté par une **tâche différée** qui traite `search_followups` échus (30 minutes après la recherche) : s'il n'y a eu **aucun clic dans la même session**, la requête est ajoutée (upsert) à `zero_result_queries`. La tâche est idempotente (`done_at`), reprend après un arrêt, et ne traite une recherche qu'une fois.
11. **Devise** : chaque source facture dans sa devise ; budgets, plafonds, coûts et écrans sont **par devise** (pas de somme entre devises). Un total consolidé en devise de référence est un affichage calculé avec les taux du contrat `fx-rates`, étiqueté « estimation ».

## 6. Valeurs proposées (retenues sur avis du relecteur) et valeurs à fournir

| Sujet | Valeur |
|---|---|
| Plafond journalier par source | **budget mensuel ÷ 30 × 2** (une journée chargée ne coupe pas la source) |
| Seuil d'alerte | **80 %** |
| Délai maximal par appel | **3 000 ms** |
| Délai global par recherche | **4 500 ms** |
| Tentatives | **2**, uniquement sur erreurs réseau et 5xx |
| Délai « sans clic » | **30 minutes**, lié à la session |
| Rétention requêtes / appels | **13 mois / 90 jours**, `user_id` effacé à 90 jours |
| Rétention des résultats bruts | **30 jours** (si autorisé par les conditions d'usage de la source) |
| Cache : disponibilité | **2 h** |
| Cache : prix | **12 h** |
| Cache : résultats de recherche | **24 h** |
| Cache : caractéristiques | **30 jours** |
| Durée maximale de cache **par source** | **[À REMPLIR — vous]** : selon les conditions d'usage de chaque API (vide = aucune mise en cache de cette source) |
| Échéance d'une réservation de budget | délai global + **30 s** (proposition) ; libération toutes les minutes, réconciliation toutes les heures |
| Rétention de `zero_result_queries` | **13 mois** après la dernière occurrence (proposition) |
| Tests d'une source | 10 par heure et par membre |
| **Sources retenues** | **[À REMPLIR — vous]** : nom, type (API de recherche web, API fabricant, flux distributeur), documentation, conditions d'usage (stockage des résultats autorisé ?), tarification |
| **Budget mensuel maximal par source** | **[À REMPLIR — vous]** : montant et devise, par source |
| Coût maximal estimé d'un appel, par source | **[À REMPLIR]** : tiré de la tarification de chaque API |
| Source de secours (un seul saut) | **[À REMPLIR]** pour chaque source |
| Nombre minimal de résultats **après fusion** avant d'arrêter les niveaux suivants | **[À REMPLIR]** (proposition : 10) |
| Clé minimale de fusion des doublons (phase 1) | référence normalisée + marque normalisée (proposition) — **à valider** |

## 7. Cohabitation avec le contrat commerce

- Le catalogue commerce (produits, prix, stock, commandes, paiements) **ne change pas**. Les fiches issues du moteur deviendront des produits du catalogue en phase 2.
- La phase 2 devra trancher comment une **offre d'un fournisseur externe** (lien de renvoi) et une **offre de vente directe** coexistent sur une même fiche : question centrale du modèle « les deux », posée avec le contrat du catalogue.
- Le tableau de bord recevra les indicateurs du moteur quand ils existeront ; jusque-là il reste vide, sans chiffres inventés.

## 8. Ordre de réalisation une fois validé

1. Contrat OpenAPI + protos `sourcing` et `search` (validation avec vous) ; permissions `engine:*`.
2. Service `sourcing` : sources, quotas, **réservation atomique du budget**, appels journalisés, disjoncteur, délai global, bascule à un saut ; **un premier connecteur réel** (celui que vous choisirez) + un connecteur de test contrôlé pour les tests automatisés.
3. Service `search` : journal des requêtes avec masquage, sans-résultat (table), cache, résultats bruts conditionnés, réindexation (tâches).
4. Gateway + écrans du backoffice (6 onglets), états vide / chargement / erreur, export CSV.
5. Tests : intégration PostgreSQL ; **concurrence du budget** (N appels simultanés, jamais plus que le plafond) ; **réservations expirées libérées après un plantage simulé** et réconciliation ; bascule (pas de boucle, pas de saut double, **une source jamais appelée deux fois** par recherche) ; délai global et arrêt anticipé ; **comptage après fusion des doublons** ; masquage des données personnelles (e-mail et téléphone absents du texte brut ET de la forme normalisée) ; **détection « sans clic » différée** (idempotente, reprise après arrêt) ; durée de cache plafonnée par source ; purge des rétentions.

## 9. Démarrage à coût zéro : Brave Search API (décision du 16 nov. 2026)

**Décisions :**
- **Source web unique au lancement : Brave Search API**, limitée au **crédit gratuit mensuel**.
- **Budget mensuel de la source = montant du crédit gratuit.** Au plafond, la source est **suspendue** et la recherche continue sur le **catalogue interne seul** (vide tant que la phase 2 n'est pas livrée : la recherche ne renvoie alors rien, jamais de données inventées).
- **Aucune source de secours payante** : `fallback_source_id` reste vide.

**Faits relevés sur les pages officielles de Brave (à confirmer dans votre tableau de bord avant le premier appel) :**

| Sujet | Information relevée |
|---|---|
| Plan « Search » | 5 $ pour 1 000 requêtes, soit **0,005 $ par requête** |
| Crédit gratuit | **5 $ chaque mois** (≈ 1 000 requêtes), appliqués automatiquement ; la page d'accueil annonce aussi « jusqu'à 20 $ » sans expliquer la différence → **à vérifier dans votre compte** |
| Débit | 50 requêtes par seconde (anciens guides : 1 par seconde → à vérifier) |
| Dépassement | réponse **429** ; Brave facture au-delà du crédit si un moyen de paiement est enregistré → **ne pas enregistrer de carte**, ou fixer une limite de dépense dans le compte |
| **Stockage des résultats** | **Interdit sur le plan standard** : la FAQ de Brave n'autorise le stockage ou l'entraînement qu'avec **un plan sur mesure** |
| Endpoint | `GET https://api.search.brave.com/res/v1/web/search`, en-tête `X-Subscription-Token`, paramètres `q`, `country`, `search_lang`, `count`, `offset` ; résultats dans `web.results` |

Sources : [brave.com/search/api](https://brave.com/search/api/), [documentation Brave](https://api-dashboard.search.brave.com/documentation/pricing), [guide de limitation de débit](https://api-dashboard.search.brave.com/documentation/guides/rate-limiting).

**Conséquences sur le contrat (valeurs de la source) :**

| Champ | Valeur |
|---|---|
| `kind` / `base_url` | `web_search_api` / `https://api.search.brave.com/res/v1/web/search` |
| `currency` | `USD` |
| `monthly_budget_micros` | **5 000 000** (5 $) — à ajuster si votre compte affiche un autre crédit |
| `cost_per_call_micros` | **5 000** (0,005 $) |
| `daily_cap_micros` | **333 333** (≈ 66 requêtes par jour = budget ÷ 30 × 2) |
| `alert_threshold_pct` | 80 |
| `fallback_source_id` | **aucune** |
| `allows_raw_storage` | **faux** — jamais, tant que le plan n'est pas sur mesure : pas de résultats bruts conservés, « Relire » indisponible pour cette source |
| `max_cache_ttl_seconds` | **vide = aucune mise en cache** des données de Brave (le stockage étant interdit, la mise en cache l'est a fortiori tant que ce n'est pas confirmé par écrit) |
| `timeout_ms` / `max_retries` / `priority` | 3 000 / 2 (réseau et 5xx seulement) / 0 |
| `secret_ref` | `BRAVE_SEARCH_API_KEY` (variable d'environnement ; la clé n'entre jamais en base) |

**Effet réel à connaître :** sans mise en cache ni stockage, **chaque recherche d'un visiteur consomme une requête Brave** (0,005 $). Avec le crédit gratuit, cela fait **environ 1 000 recherches par mois** au total, puis la source est suspendue et la recherche externe s'arrête jusqu'au mois suivant. C'est suffisant pour tester ; ce ne l'est pas pour un site public. Deux protections s'imposent donc dès le lancement : la **limitation du débit par visiteur** sur le site, et l'option de n'appeler la source externe que sur action explicite (bouton « chercher aussi sur le web »). À trancher avec vous avant d'ouvrir la recherche au public.

**Ce que j'ai vérifié et ce que je n'ai pas pu vérifier :** les pages ci-dessus concordent sur les 5 $ de crédit, le prix de 0,005 $ la requête, les 50 requêtes par seconde et l'interdiction de stocker sur le plan standard. Elles ne concordent pas sur le « jusqu'à 20 $ » ni sur l'ancien débit de 1 par seconde. Je n'ai pas lu les conditions d'utilisation complètes (page de conditions non retrouvée) : **à lire par vous avant la mise en production**, notamment l'obligation éventuelle d'afficher « Brave » comme source des résultats.

## 10. Décisions du 16 nov. 2026 (suite) : protections de la recherche web et ordre des travaux

**Recherche web : deux protections obligatoires dès l'ouverture au public.**
1. **Aucun appel automatique.** La source externe n'est jamais interrogée par une recherche ordinaire : le visiteur déclenche l'appel avec un bouton **« Chercher aussi sur le web »**, affiché sous les résultats du catalogue interne (ou à leur place s'il n'y en a aucun). La recherche ordinaire ne coûte donc rien.
2. **Limite par visiteur.** Nombre maximal de recherches web par visiteur (session anonyme) et par période. **Valeur proposée : 3 par jour et par visiteur, 10 par jour et par adresse réseau** (deux plafonds, pour qu'ouvrir une nouvelle session ne contourne pas la limite). Au-delà : message clair (« Limite atteinte, revenez demain ») et aucun appel. Ces plafonds s'ajoutent à celui du budget de la source (≈ 66 appels par jour). **Valeurs à valider.**

**Ordre des travaux validé :** (1) gateway + audit des écritures de `sourcing` *(fait)* ; (2) **CRUD Produits** : ajout manuel, prioritaire car le catalogue interne est la base de la recherche ; (3) service `search` ; (4) écrans du moteur dans le backoffice.

**Clé Brave :** lue **uniquement dans l'environnement** du processus (`BRAVE_SEARCH_API_KEY`) ; `./dev.sh up` ne l'écrit dans aucun fichier ni journal et ne la passe jamais en argument. Pour la renseigner : `export BRAVE_SEARCH_API_KEY=…` dans le terminal, puis `./dev.sh up`. Limite connue : un processus du même utilisateur peut lire l'environnement d'un autre ; en production, la clé viendra d'un coffre de secrets.

**Audit de `sourcing` :** chaque écriture publie un événement `audit.recorded.v1` dans l'outbox (même transaction que l'écriture : une action refusée ne laisse aucune trace) ; identity l'inscrit dans son journal append-only par un consommateur idempotent. La suspension automatique d'une source pour budget épuisé est auditée une seule fois, avec l'auteur « système ».
