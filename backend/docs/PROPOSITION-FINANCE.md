# Centre de pilotage financier de FilterFinder : proposition révisée (v2)

Statut : **PROPOSITION, rien n'est validé.** Aucun code, aucun contrat, aucune migration. Cette version **remplace** la v1 (conservée dans `PROPOSITION-FINANCE-v1.md`).

Méthode : avant d'écrire, j'ai relu le contrat (`api/openapi.yaml`), les protos (`order`, `payment`, `quote`, `pricing`, `events`), les services réellement présents, les conventions d'événements (`pkg/events`, `pkg/outbox`, `pkg/natsbus`) et les permissions existantes.

Légende : **[EC]** = à valider par votre expert-comptable ; **[VOUS]** = décision métier qui vous appartient ; **[PRESTATAIRE]** = dépend de la documentation officielle du prestataire de paiement, que je n'ai pas.

---

## 1. Diagnostic critique de ma proposition précédente

| # | Défaut de la v1 | Conséquence | Correction en v2 |
|---|---|---|---|
| 1 | **Elle mélangeait finance interne et flux de la plateforme** (paiements, remboursements, litiges) dans un même module. | Des responsabilités floues ; un incident de paiement aurait pu bloquer la comptabilité, et inversement. | Deux domaines distincts, trois services aux rôles séparés (§ 3, 5). |
| 2 | **Elle plaçait la facturation et les devis dans la finance.** Or le contrat provisoire les attribue déjà à d'autres services : factures, avoirs et remboursements au service **Commandes** ; devis au service **Devis** ; paiements, webhooks, rapprochement manuel au service **Paiements**. | Doublon de données, deux sources de vérité pour une facture. | La finance **observe** (événements) ; elle n'émet ni facture ni devis (§ 3). |
| 3 | **Elle promettait une numérotation « sans trou »** sans en expliquer le prix. Je l'ai moi-même constaté dans le catalogue : une création refusée consomme un numéro de SKU (un compteur PostgreSQL ne revient jamais en arrière). | Promesse fausse pour une `SEQUENCE`, coûteuse ailleurs. | Quatre familles de numéros distinguées, garanties différentes, contraintes expliquées (§ 9.4). |
| 4 | **Les indicateurs confondaient** chiffre d'affaires, facturation, paiement et encaissement. | Un chiffre « encaissé » qui n'est en réalité que « payé chez le prestataire » (l'argent n'est pas encore en banque). | Cinq états distincts d'une vente et deux notions d'encaissement (§ 8, 11). |
| 5 | **Événements décrits en une ligne** (`order.paid`…), sans structure, sans traitement d'erreur, sans reprise, sans gestion de l'ordre. | Risque de pertes silencieuses et d'écritures doublées. | Catalogue d'événements, boîte de réception idempotente, mise en quarantaine, rattrapage (§ 6, 13). |
| 6 | **Sécurité des paiements à peine abordée** (signature des webhooks, rejeu, protection des données bancaires). | Faille potentielle sur le point le plus exposé. | § 10 et § 14. |
| 7 | **Aucun traitement des frais de transaction, des règlements du prestataire, des litiges, ni des remboursements liés à la transaction d'origine.** | Le solde bancaire ne se rapproche jamais du chiffre des ventes (frais, délais, litiges). | Modèle de flux de plateforme complet (§ 7, 8). |
| 8 | **Les alertes pouvaient sembler bloquantes** (R9, R10, R11 mêlaient intégrité, soupçon et blocage). | Opérations légitimes empêchées sans décision de votre part. | Séparer « contrôle d'intégrité » (technique, bloquant) et « alerte métier » (jamais bloquante sans règle que **vous** validez) (§ 9.3). |
| 9 | **Marge brute et seuil de rentabilité promis sans donnée de coût.** Le catalogue n'a pas de prix d'achat, et il n'existe ni stock ni valorisation. | Des KPI impossibles à calculer honnêtement. | Marqués « indisponibles » tant que la donnée n'existe pas ; options en décision (§ 11, 17). |
| 10 | **KPI définis par numéros de comptes écrits en dur** (« classe 7 »). | Fragile, et c'est une règle comptable que je n'ai pas à décider. | Tables de correspondance paramétrables, validées par l'expert-comptable (§ 11). |
| 11 | **La phase 4 de la v1 (commandes + paiements + factures + finance) était trop grosse** et ses dépendances mal ordonnées. | Phase intestable et non validable. | Les services de vente sont leurs **propres phases** validées avant l'intégration financière (§ 15). |
| 12 | **Permissions `finance:*` créées sans les rapprocher des existantes** (`payments:*`, `invoices:read`, `orders:refund`…). | Chevauchements et trous. | Matrice unifiée (§ 10). |
| 13 | **Doublon avec le tableau de bord existant** (`/admin/dashboard/revenue`, alimenté par le service Commandes). | Deux « chiffres d'affaires » qui divergent sans explication. | Distinction explicite entre pilotage commercial et pilotage comptable, avec écran de réconciliation (§ 12). |

---

## 2. Ce qui existe réellement (état au 17 nov. 2026)

| Élément | État réel |
|---|---|
| Services **construits** | `gateway`, `identity` (clients, entreprises, personnel, rôles, journal d'audit), `sourcing`, `catalog` (taxonomie, produits). Un dossier `content` existe sans code exploitable : je ne m'appuie pas dessus. |
| Services **en contrat uniquement** (protos et routes provisoires, **aucun code**) | `order`, `payment`, `quote`, `pricing` (taux de change, règles de taxe), `fulfillment` (stock, expéditions), `cart`, `notification`, `support`. |
| Ce que le contrat provisoire prévoit pour la vente | Commande (`placed → preparing → shipped → delivered / cancelled`) ; statut de paiement (`pending, paid, partially_refunded, refunded, failed, voided`) ; **facture** émise par le service Commandes (moment configurable : commande, expédition ou livraison) ; **avoirs** liés aux remboursements ; remboursements avec **canal** (moyen d'origine, virement, espèces, mobile money, avoir) et **raison** ; devis (brouillon → envoyé → accepté → converti). |
| Paiement (contrat provisoire) | Moyens : `mobile_money`, `card`, `bank_transfer`, `cash_on_delivery`, `deferred_invoice`. **Un seul prestataire prévu au lancement : GeniusPay** (adaptateur remplaçable). Les deux derniers moyens sont **désactivés au lancement**. Statuts : `requires_action, processing, succeeded, failed, cancelled, expired, refunded, partially_refunded`. Journal des notifications du prestataire (`provider_events`) idempotent. Rapprochement **manuel** d'un paiement (`mark-paid`, permission `payments:reconcile`). Conditions de crédit par entreprise (plafond, délai). |
| Taxe | Règles dans le service `pricing` (taux en points de base par pays et type de client, exonération, autoliquidation, dates de validité). Le total d'une commande **fige** le taux et le montant appliqués. |
| Événements existants (`events.v1`) | `OrderPlaced`, `OrderStatusChanged`, `OrderPaymentStatusChanged`, `OrderPaid`, `OrderCancelled`, `RefundRequested/Succeeded/Failed`, `PaymentSucceeded/Failed/Refunded`, … avec enveloppe (`event_id`, type, date, agrégat, trace, auteur). |
| Mécanique d'événements | Écriture **dans la même transaction** que la donnée (outbox), relais vers NATS JetStream, consommateurs idempotents (`processed_events`), journal d'audit append-only déjà branché (flux `AUDIT`). |
| **Absent du contrat** | Aucune notion de **commission, reversement, vendeur tiers ou place de marché**. Aucun **règlement du prestataire** (versement vers votre banque), **frais de transaction** ni **litige** (contestation de paiement). Aucun **prix d'achat** produit, aucun stock valorisé. |
| Site | Le tunnel de commande n'est pas branché (message « démo »). Aucune vente réelle n'existe. |

Conséquence : **aucune donnée financière réelle n'existe encore**, et la chaîne de vente n'est pas construite. La finance ne peut pas être alimentée automatiquement avant que Commandes et Paiements existent.

---

## 3. Périmètre fonctionnel définitif proposé

**Le modèle économique, tel que je le lis dans vos documents :** FilterFinder vend **son propre catalogue** (prix de vente direct, hors taxes, en F CFA), à des particuliers et à des entreprises, avec commande en ligne et devis. Les « offres externes » prévues en phase 2 du moteur sont des **liens de renvoi** vers des sites tiers : aucun argent n'y transite par la plateforme.

Donc, **je ne propose PAS** : commissions, reversements aux fournisseurs, comptes séquestres, partage de revenus. Ils sont hors périmètre tant que vous ne confirmez pas un modèle de place de marché (décision D1). Le modèle de données laisse la porte ouverte (tiers, dettes) sans rien construire.

### 3.1 Domaine A : finance interne de l'entreprise (service `finance`)
Comptabilité en partie double ; dépenses et factures fournisseurs ; autres revenus ; trésorerie et banques ; rapprochement bancaire ; budgets et prévisions ; fiscalité (préparation et exports) ; rentabilité ; indicateurs.

### 3.2 Domaine B : flux financiers de la plateforme (service `payment`, avec `order` pour les documents de vente)
Paiements, tentatives, statuts ; webhooks du prestataire ; remboursements et annulations ; frais de transaction ; règlements du prestataire ; litiges ; paiements en attente ou échoués ; conditions de crédit B2B ; rapprochement **transaction ↔ prestataire**. Les **factures et avoirs** sont des documents de vente (service `order`).

### 3.3 Ce que la finance fait des flux (sans les posséder)
Elle **reçoit** leurs événements, les **traduit en écritures** par des règles validées, et **contrôle** la cohérence entre les trois mondes (transactions, règlements du prestataire, banque). Elle ne modifie jamais un paiement, une commande ou une facture.

---

## 4. Distinction entre finance interne et flux de la plateforme

| | **Flux de la plateforme** (B) | **Finance interne** (A) |
|---|---|---|
| Question | « Qu'est-ce qui s'est passé avec cet argent client ? » | « Où en est l'entreprise ? » |
| Nature | **Opérationnel**, temps réel, idempotent | **Comptable**, immuable, daté, périodique |
| Vérité sur | l'état d'un paiement, d'un remboursement, d'un litige | les écritures, les soldes, les états financiers |
| Disponibilité | **critique** (le paiement du client en dépend) | différée acceptable (rattrapage possible) |
| Propriétaire | `payment` (+ `order` pour factures/avoirs) | `finance` |
| Données de carte | jamais stockées (page de paiement hébergée par le prestataire) | aucune |
| Relation | émet des événements | les consomme ; **ne les rappelle jamais** |

**Règle d'or : pas de dépendance circulaire.** Le sens est unique : `order`, `payment`, `pricing` → (événements) → `finance`. `finance` peut **lire** (gRPC en lecture seule) pour contrôler ou rattraper, mais n'écrit jamais chez eux et eux ne l'appellent jamais. Le checkout ne dépend donc pas de la disponibilité de la comptabilité.

**Où vit chaque décision ?** *Là où l'état vit.* L'approbation d'un remboursement se fait dans `order`/`payment` (permission `orders:refund`, déjà au contrat). L'approbation d'une écriture se fait dans `finance`. Un seul endroit par décision, pas de double état à synchroniser.

---

## 5. Architecture technique globale

```
 Client ─► gateway ─┬─► order ───────┐ (commandes, factures, avoirs, remboursements)      ┌──► audit (identity)
                    ├─► quote ───────┤                                                    │
                    ├─► payment ─────┼── outbox ─► NATS JetStream ─► finance (inbox) ─────┤
                    │      ▲         │                │                                   │
                    │      │ webhooks│                └─ rattrapage (lecture gRPC) ◄─────┘
  Prestataire ──────┘      │         │
  (GeniusPay)  ◄───────────┘         └─► pricing (taxes) ─ lu par order, figé dans les totaux
                    └─► finance ──► écritures · trésorerie · budgets · fiscalité · KPI
```

Principes conservés de la v1 : service indépendant, **base PostgreSQL dédiée** par service, gRPC interne, routes exposées par le gateway avec **permission vérifiée avant relais**, outbox + NATS, **journal d'audit append-only**, calculs **uniquement** dans le backend, montants en **entiers d'unités mineures** (jamais de flottant), idempotence.

Améliorations :
- **Boîte de réception (inbox)** côté finance : tout événement reçu est d'abord **stocké brut** (identifiant unique) avant d'être interprété. On peut donc **rejouer** sans redemander à l'émetteur.
- **Règles de comptabilisation versionnées** : la traduction « événement → écritures » est une **table de règles** (donnée, pas du code), versionnée, validée par l'expert-comptable **[EC]**. Une règle a un état (`brouillon`, `approuvée`) ; tant qu'elle n'est pas approuvée, ses écritures restent en **brouillon** (à valider par un humain) et ne comptent dans aucun état financier.
- **Séparation des responsabilités à l'intérieur de `finance`** (modules internes, un seul service au départ ; extractibles plus tard) :

| Module interne | Responsabilité | N'a pas le droit de |
|---|---|---|
| `ledger` (comptabilité) | plan, journaux, écritures, périodes, contre-passation, grand livre, balance | connaître un paiement ou une facture (il ne voit que des écritures) |
| `operations` | factures fournisseurs, dépenses, autres revenus, tiers, récurrences | écrire directement dans le grand livre (il demande une écriture à `ledger`) |
| `ingest` | inbox, règles de comptabilisation, quarantaine, rattrapage | modifier une écriture validée |
| `treasury` | comptes bancaires, relevés, règlements du prestataire, rapprochement, prévisions | modifier un relevé importé |
| `controls` | contrôles d'intégrité, alertes, anomalies, approbations | bloquer une opération hors règle validée |
| `reporting` | états financiers, KPI, exports | écrire (lecture seule sur `ledger`) |

Les calculs (KPI, états) lisent `ledger` ; **aucun module ne recalcule un solde d'une autre façon**.

---

## 6. Responsabilités de chaque service

| Service | Possède | Émet | Consomme | État |
|---|---|---|---|---|
| `order` | commandes, **factures**, **avoirs**, demandes de remboursement, numérotation des documents de vente | `OrderPlaced`, `invoice.issued`, `credit_note.issued`, `OrderCancelled`, `RefundRequested` | `PaymentSucceeded/Failed/Refunded`, `RefundSucceeded/Failed` | contrat provisoire, **pas de code** |
| `payment` | paiements, tentatives, `provider_events`, remboursements exécutés, **frais**, **règlements du prestataire**, **litiges**, conditions de crédit | `PaymentSucceeded/Failed/Refunded`, `RefundSucceeded/Failed`, `payment.fee_recorded`, `provider.settlement_received`, `payment.dispute_*` | `RefundRequested`, `OrderCancelled` | contrat provisoire, **pas de code**, **à étendre** (frais, règlements, litiges) |
| `quote` | devis | `quote.sent/accepted/converted` | | contrat provisoire |
| `pricing` | taux de change, règles de taxe | `price.changed` | | contrat provisoire |
| **`finance`** | grand livre, opérations internes, trésorerie, budgets, fiscalité, KPI | `finance.period_closed`, `finance.anomaly_raised` | tous les événements ci-dessus | **à créer** (aucun code) |
| `identity` | clients, personnel, rôles, **audit** | `audit.recorded` | `audit.recorded` des autres | construit |

**Un point d'attention** : `payment` doit être **étendu** (frais, règlements, litiges) : ce sont des ajouts à son contrat provisoire, à valider **avant** que `finance` en dépende.

---

## 7. Cycle de vie et échanges : une vente de bout en bout

Une vente traverse **cinq états distincts**, chacun avec sa source et sa date. Les confondre est la source n° 1 d'erreurs.

| # | État | Source de vérité | Événement | Effet comptable (principe) **[EC]** | Argent en banque ? |
|---|---|---|---|---|---|
| 1 | **Commande** passée | `order` | `OrderPlaced` | aucune écriture de produit (engagement, pas encore un revenu) ; selon l'expert, un enregistrement hors bilan | non |
| 2 | **Facturation** (le revenu est constaté au moment défini par `invoice_issuance_moment`) | `order` | `invoice.issued` | produit + créance client + TVA collectée | non |
| 3 | **Paiement confirmé** (le prestataire dit « réussi ») | `payment` | `PaymentSucceeded` | la créance client est soldée **contre un compte de transit prestataire** (l'argent est chez le prestataire, pas à la banque) | **non** |
| 4 | **Règlement du prestataire** (il vous verse, **net de frais**) | `payment` (rapport prestataire) puis banque | `provider.settlement_received`, `payment.fee_recorded` | transit → banque ; frais en charge | **oui** |
| 5 | **Rapprochement bancaire** | `finance` / `treasury` | (action interne) | la ligne du relevé est liée au règlement ; écart conservé s'il y en a | oui, **confirmé** |

États alternatifs (traités explicitement, jamais « ignorés ») :

| Cas | Traitement |
|---|---|
| Paiement **en attente** (`requires_action`, `processing`) | aucun effet comptable ; compté dans « paiements en attente » (KPI), avec ancienneté |
| Paiement **échoué / annulé / expiré** | aucune écriture de produit ; conservé comme transaction (KPI « échecs », motif) ; la facture reste une créance si elle existe |
| **Annulation avant paiement** | si facture émise : **avoir** (événement `credit_note.issued`) qui annule la créance |
| **Annulation / remboursement après paiement** | `RefundRequested` → exécution chez `payment` → `RefundSucceeded` (lié à la transaction d'origine par `payment_id`, `order_id`, `refund_id`) → avoir + écriture de remboursement. Pendant l'attente (`refund_pending`), le montant est un **passif à rembourser** (KPI) |
| **Remboursement échoué** | reste en file (déjà au contrat : `retry`, `complete` manuel) ; aucune écriture de sortie d'argent tant qu'il n'a pas réussi |
| **Remboursement manuel** (virement, espèces, mobile money) | écriture à la **confirmation** de l'opérateur (`complete`), avec pièce **[VOUS]** |
| **Litige / contestation** | cas ouvert → montant **provisionné ou bloqué selon l'expert [EC]** → clôture gagnée/perdue → écriture |
| **Écart de règlement** (le prestataire verse moins que prévu) | l'écart est **conservé** dans le rapprochement (jamais absorbé), puis résolu par une écriture explicite |

---

## 8. Modèle de données détaillé

Convention commune : clés `uuid` (v7), montants **`bigint` en unités mineures + code devise** (XOF : le franc), dates `timestamptz` (UTC), contraintes **en base**, suppression logique interdite pour tout ce qui est comptable.

### 8.1 Service `finance`

**Référentiel**
- `accounts` : plan comptable (`code`, `label`, `class`, `kind`, `parent_id`, `active`, `normal_side`). Code unique ; compte utilisé non supprimable.
- `journals` : `code`, `label`, `kind` (ventes, achats, banque, caisse, opérations diverses…).
- `fiscal_years`, `fiscal_periods` : `status` ∈ `open | closed`, `closed_at`, `closed_by`.
- `kpi_account_map` : `kpi_code` → ensemble de comptes (et sens). **Paramétrable [EC]**, c'est ici que « chiffre d'affaires » est défini, pas dans le code.
- `finance_settings` : devise de tenue (XOF), seuils de double validation, jours de relance, etc.

**Grand livre (cœur, immuable)**
- `entries` : `id`, `journal_id`, `status` ∈ `draft | posted`, `entry_number` (**NULL tant que brouillon**, attribué à la validation), `accounting_date` (date comptable), `booked_at` (date d'enregistrement), `period_id`, `label`, `origin` ∈ `manual | rule | operation | reversal`, `source_type`, `source_id`, `source_event_id`, `rule_id` + `rule_version`, `reverses_entry_id`, `reversal_reason`, `created_by`, `posted_by`, `approved_by`.
  Contraintes : `source_type, source_id, kind` **unique** (anti-doublon) ; une écriture `posted` est **non modifiable et non supprimable** (trigger, comme le journal d'audit) ; une période `closed` refuse toute validation ; `posted_by ≠ created_by` pour les origines à double validation.
- `entry_lines` : `entry_id`, `account_id`, `debit`, `credit` (un seul non nul, ≥ 0), `currency`, `amount_original`, `fx_rate` (`numeric`), `third_party_id`, `activity_id`, `department_id`, `memo`.
  Contrainte de validation : Σ débits = Σ crédits **par écriture**.
- `entry_number_counters` : `(journal_id, fiscal_year_id)` → `last_number` (compteur **verrouillé** à la validation, voir § 9.4).

**Intégration (flux entrants)**
- `inbox_events` : `event_id` (**unique**), `type`, `occurred_at`, `aggregate_id`, `aggregate_version`, `payload` (brut), `received_at`, `status` ∈ `received | processed | failed | quarantined | ignored`, `attempts`, `next_attempt_at`, `last_error`, `waiting_for` (dépendance manquante).
- `posting_rules` : `event_type`, `version`, `status` ∈ `draft | approved | retired`, `conditions`, `template` (lignes à générer), `auto_post` (**faux** par défaut), `approved_by` **[EC]**.
- `reconciliation_runs` / `reconciliation_gaps` : contrôles de complétude entre l'émetteur et `finance` (§ 13.3).

**Opérations internes**
- `third_parties` : **référence** (`kind`, `external_ref`, `display_name`), **pas** de copie des données personnelles des clients (elles restent dans `identity`).
- `supplier_bills` : fournisseur, numéro de facture fournisseur, dates (émission, échéance), montants HT/TVA/TTC, statut ∈ `draft | approved | paid | disputed`, pièce jointe. **Unique** `(fournisseur, numéro, montant)` côté contrôle d'intégrité.
- `expenses` : dépense directe (catégorie, nature fixe/variable, département, moyen de paiement).
- `other_incomes` : revenus hors vente en ligne (subvention, produit exceptionnel…) **[VOUS]**.
- `recurring_templates` : modèles de charges récurrentes (génèrent des **brouillons**, jamais des écritures validées).
- `attachments` : pièces (référence de stockage S3, empreinte, taille) : dépend de la décision de stockage.

**Trésorerie**
- `bank_accounts` : nom, banque, devise, **numéro chiffré** (AES-GCM, clé hors base), `last4`, compte comptable lié, solde d'ouverture.
- `bank_statements` + `bank_statement_lines` : import **immuable** (empreinte du fichier unique → ré-import refusé), montant signé, date de valeur, libellé, référence.
- `settlement_records` : copie **de lecture** des règlements du prestataire (reçus par événement), `gross`, `fees`, `net`, période couverte, `provider_reference`.
- `reconciliations` : `statement_line_id` ↔ (`settlement_record_id` | `entry_line_id`), `matched_by` (`auto`/`manual`), `status`, `difference` (**conservée**), `resolved_by`, `resolution_entry_id`.
- `cash_forecast_runs` : hypothèses et résultat horodatés (jamais écrasés).

**Budgets**
- `budgets` (année, version, statut), `budget_lines` (compte × mois × département), `scenarios` (hypothèses nommées, jamais confondues avec le réel).

**Fiscalité**
- `tax_codes` (taux en points de base, comptes de collecte/déduction, validité **[EC]**), `tax_periods`, `tax_returns` (montants **calculés**, statut, référence de dépôt saisie).

**Contrôles**
- `approval_requests` : `operation_type`, `payload_hash`, `requested_by`, `approved_by`, `status`, `reason`, `expires_at`.
- `control_rules` : règles d'alerte (code, seuil, `severity`, **`blocking`**, `enabled`) ; `blocking = false` par défaut.
- `alerts` : alerte déclenchée (règle, objet, statut `open | justified | confirmed`, justification, par qui).
- `finance_audit_notes` : motif **obligatoire** des corrections ; l'audit détaillé passe par `audit.recorded` (append-only, existant).

### 8.2 Extensions nécessaires au service `payment` (à ajouter à son contrat provisoire)
- `payments`, `payment_attempts` (déjà prévus) : ajouter `net_expected`, `fee_amount` lorsque connus.
- `provider_events` (prévu) : `event_id` unique, empreinte du corps, **signature vérifiée (oui/non)**, `received_at`, issue.
- `provider_fees` : frais par transaction ou par règlement (**source : rapport du prestataire [PRESTATAIRE]**).
- `provider_settlements` : règlement (montant brut, frais, net, date, référence) **[PRESTATAIRE]**.
- `payment_disputes` : ouverture, motif, montant, échéance de réponse, issue (gagné/perdu) ; utile surtout aux **cartes** **[PRESTATAIRE]**.
- `refunds` (prévu côté `order`) : garder `payment_id` d'origine **obligatoire** quand un moyen d'origine existe.

### 8.3 Extensions nécessaires au service `order`
- `invoices` : numéro, série, date d'émission, **échéance**, totaux HT/TVA/TTC **figés**, statut ∈ `issued | void` (une facture émise n'est jamais modifiée ni supprimée), `order_id`.
- `credit_notes` : numéro, `invoice_id` **d'origine obligatoire**, motif, montants.
- `document_number_counters` : par `(série, exercice)`.

---

## 9. Règles de gestion et contrôles

### 9.1 Principes comptables structurants
| N° | Règle |
|---|---|
| R1 | **Séparation CA / facturation / paiement / encaissement** (§ 7). Chaque notion a sa source, sa date, son KPI. |
| R2 | Une facture émise n'est ni modifiée ni supprimée ; elle est **corrigée par un avoir**. Elle n'est « payée » que lorsque des paiements confirmés la soldent. |
| R3 | Tout **paiement** garde son **état** (en attente, confirmé, échoué, annulé, expiré, remboursé) ; un état « confirmé » ne vaut pas « en banque ». |
| R4 | Tout **remboursement** garde le lien `payment_id` / `order_id` / `refund_id` vers la transaction d'origine et vers son avoir. |
| R5 | Une écriture **validée** est immuable (base). Toute correction = **contre-passation** + nouvelle écriture, avec motif, auteur, date, tracées. |
| R6 | Un **brouillon** est modifiable ; sa suppression est tracée. Une écriture automatique issue d'une règle non approuvée **reste brouillon**. |
| R7 | Une période **clôturée** refuse toute validation ; la réouverture exige une permission dédiée, un **motif** et une double validation. |
| R8 | Les **écarts de rapprochement ne sont jamais absorbés** : ils restent visibles, datés, assignés, jusqu'à une écriture de résolution explicite. |
| R9 | **Aucune donnée fictive** : pas d'amorçage de chiffres ; les écrans vides disent « aucune donnée » ; un KPI sans base affiche « — », pas 0. |
| R10 | Les corrections de **taxe** et le **plan comptable** sont des paramètres validés par l'expert **[EC]**, jamais des constantes dans le code. |

### 9.2 Intégrité technique (bloquante, parce qu'incontournable)
Contraintes **en base** qui refusent l'opération sans appel à une décision métier : écriture déséquilibrée ; modification/suppression d'une écriture validée ; validation dans une période clôturée ; clé d'unicité de source déjà utilisée (anti-doublon d'événement) ; événement rejoué (idempotence) ; ligne de relevé déjà rapprochée ; import de relevé déjà importé (empreinte).

### 9.3 Alertes métier (jamais bloquantes sans votre validation)
Une alerte **signale et demande un examen**, elle n'empêche rien. Pour chaque règle, vous décidez : actif ou non, seuil, gravité, **bloquante ou non** (par défaut : **non**).

| Règle proposée | Signale | Bloquante par défaut |
|---|---|---|
| Doublon probable (mêmes fournisseur + n° + montant) | saisie en double d'une facture fournisseur | non (une confirmation explicite « ce n'est pas un doublon » est tracée) |
| Montant atypique (très supérieur à l'historique de la catégorie) | erreur de saisie probable | non |
| Dépense au-dessus d'un seuil sans pièce | justificatif manquant | non |
| Écriture manuelle sur compte sensible | contournement possible | non |
| Rupture de la numérotation attendue | document manquant ou supprimé | non |
| Écart de rapprochement non résolu depuis N jours | argent non expliqué | non |
| Paiements « confirmés » sans règlement du prestataire depuis N jours | règlement manquant | non |
| Remboursements anormalement fréquents pour un même auteur/client | abus possible | non |
| Solde de trésorerie prévisionnel sous un seuil | tension de trésorerie | non (information) |
| Webhook rejeté (signature invalide) ou rejoué | tentative d'intrusion ou incident prestataire | non (alerte immédiate) |

### 9.4 Numérotation : quatre familles, quatre garanties
La **numérotation continue sans trou n'est pas gratuite** : elle impose de **verrouiller un compteur** dans la même transaction que l'émission (sinon un échec crée un trou, comme avec une `SEQUENCE` PostgreSQL). Cela **sérialise** les émissions d'une série (un seul document à la fois par série) : acceptable pour des factures (quelques dizaines par jour), inutile ailleurs.

| Famille | Exemple | Garantie proposée | Mécanisme | Pourquoi |
|---|---|---|---|---|
| **Numéros de factures et d'avoirs** | `FA-2026-000123` | **Continue, sans trou, par série et exercice** *si votre expert l'exige* **[EC]** | compteur verrouillé, attribué **à l'émission** (jamais au brouillon), dans la même transaction que la facture ; une facture émise n'est jamais supprimée (annulée par avoir) | exigence usuelle de la facturation ; je ne l'invente pas, à confirmer |
| **Numéros d'écritures comptables** | `VT-2026-000456` | **Continue par journal et exercice, attribuée à la validation** | idem compteur verrouillé ; **les brouillons n'ont pas de numéro** | un brouillon abandonné ne doit pas créer de trou |
| **Références de transactions** | `pay_…`, `ref_…`, référence prestataire | **Unique, opaque, sans exigence de continuité** | uuid + référence du prestataire (unique) | l'ordre n'a pas de sens comptable ; les trous sont normaux |
| **Numéros de commandes et de devis** | `CMD-…`, `DV-…` | **Unique, lisible ; trous tolérés** | compteur ou séquence | document commercial, pas comptable ; **[EC]** pour confirmer que les devis n'ont pas d'exigence |

Contraintes à connaître : le verrou de compteur ralentit l'émission sous forte concurrence (acceptable ici) ; la continuité suppose **zéro suppression** (donc annulation par avoir) ; un changement d'exercice **remet** le compteur à zéro (série annuelle) ; la reprise après incident ne **réutilise jamais** un numéro attribué.

### 9.5 Protections supplémentaires
- **Idempotence** de bout en bout (§ 13.2). **Anti-doublon** par clés d'unicité de source, pas par heuristique.
- **Maker-checker** appliqué par le code, **y compris au super-administrateur** : l'auteur d'une opération sensible ne peut jamais l'approuver lui-même (sinon la double validation ne protège rien). Opérations sensibles proposées : validation d'écriture manuelle, contre-passation, réouverture de période, rapprochement forcé avec écart, règle de comptabilisation approuvée, export complet, modification des seuils **[VOUS : liste et seuil en F CFA]**.
- **Corrections** : jamais d'écrasement ; toujours un **motif** obligatoire.

---

## 10. Système de permissions

Les permissions existantes sont **conservées** et mises en cohérence ; seules les manquantes sont ajoutées (même format `domaine:action`, un mot par action).

| Domaine | Existantes (contrat) | Proposées en plus |
|---|---|---|
| Commandes / remboursements | `orders:read`, `orders:write`, `orders:refund` | (aucune) |
| Paiements | `payments:read`, `payments:reconcile` (mark-paid, remboursement manuel), `payments:settings` | `payments:disputes` (traiter un litige) |
| Factures | `invoices:read` | `invoices:void` (annuler par avoir) |
| Finance | (aucune) | `finance:read`, `finance:write` (brouillons, saisies), `finance:approve` (valider une écriture, 2ᵉ signature), `finance:close` (clôture/réouverture), `finance:admin` (plan, règles, paramètres), `finance:export` |
| Trésorerie | (aucune) | `treasury:read`, `treasury:write` (importer, saisir), `treasury:reconcile` (rapprocher) |
| Budgets, fiscalité | (aucune) | `budget:read`, `budget:write`, `tax:read`, `tax:write` |

**Séparation des rôles recommandée** (matrice indicative, à ajuster **[VOUS]**) :

| Rôle | Peut | Ne peut pas |
|---|---|---|
| **Dirigeant** | lire tout (`finance:read`, `treasury:read`, `budget:read`, `tax:read`, `orders:read`, `payments:read`), `finance:export` | saisir, valider |
| **Comptable interne** | `finance:write`, `treasury:write`, `treasury:reconcile`, `budget:write` | `finance:approve` sur ses propres écritures, `finance:close` |
| **Responsable financier (approbateur)** | `finance:approve`, `finance:close` | saisir et approuver la **même** opération |
| **Gestionnaire paiements** | `payments:read`, `payments:reconcile`, `orders:refund` | écrire en comptabilité |
| **Expert-comptable externe** | `finance:read`, `finance:export`, `tax:read` (lecture seule) | tout le reste |
| **Auditeur** | lecture + journal d'audit | écrire |

**Principes** : les données bancaires complètes ne sont visibles de personne ; seuls les 4 derniers caractères sont affichés. La permission `finance:admin` (plan comptable, règles de comptabilisation) est **rare** et **tracée**. Le gateway refuse une route sans permission (déjà en place, échec fermé).

---

## 11. Indicateurs financiers : définitions et formules

**Règles communes.** Tous calculés **par le backend**. Chaque réponse contient : valeur, **période**, **base de calcul** (`engagement` ou `trésorerie`), **comparaison** (période précédente, même période l'an passé), **date de calcul**, et un **indicateur de complétude** : nombre d'événements **en attente d'écriture** ou **en brouillon** qui n'y sont pas encore (jamais de chiffre « propre » qui masque du retard). Chaque valeur est **traçable** : indicateur → comptes → écritures → document source (facture, paiement, événement). Aucun KPI sans donnée n'affiche 0 : il affiche « — ». **« Comptes » = table `kpi_account_map` validée [EC].**

Deux bases à ne jamais mélanger : **base engagement** (le revenu est constaté à la facturation) et **base trésorerie** (l'argent a réellement bougé).

| # | KPI | Définition | Formule | Sources | Période | Limites |
|---|---|---|---|---|---|---|
| 1 | **Chiffre d'affaires** | Produits d'exploitation constatés (facturés, nets d'avoirs) | Σ crédits − Σ débits des comptes du KPI « CA » sur la période | écritures validées issues de `invoice.issued` et `credit_note.issued` | jour/mois/trim./année | périmètre exact **[EC]** ; hors produits financiers/exceptionnels |
| 2 | **Revenus facturés** | Montant HT facturé | Σ factures émises − Σ avoirs émis (HT) | événements `invoice.issued`, `credit_note.issued` | idem | ne dit **rien** du paiement |
| 3 | **Paiements confirmés** | Montants confirmés réussis par le prestataire (argent chez le prestataire, pas forcément en banque) | Σ `PaymentSucceeded` − Σ remboursements réussis | `payment` | idem | **ce n'est pas un encaissement bancaire** |
| 4 | **Encaissements bancaires** | Argent réellement reçu en banque, rapproché | Σ lignes de relevé créditrices rapprochées (clients, règlements du prestataire) | relevés + rapprochements | idem | dépend des relevés importés ; non rapproché = non compté |
| 5 | **Dépenses** | Charges constatées | Σ débits − Σ crédits des comptes « charges » | écritures (factures fournisseurs, dépenses, frais de paiement) | idem | classement des charges **[EC]** |
| 6 | **Résultat d'exploitation** | Rentabilité de l'activité courante | produits d'exploitation − charges d'exploitation | écritures | idem | périmètre **[EC]** |
| 7 | **Résultat net** | Gain après toutes charges et impôts | total produits − total charges | écritures | idem | impôt sur les résultats : saisi/estimé **[EC]**, jamais inventé |
| 8 | **Marge brute** | Marge après coût d'achat des marchandises | (CA − coût d'achat des ventes) ÷ CA | CA + **coût d'achat par produit** | idem | **INDISPONIBLE** : aucun prix d'achat ni stock valorisé n'existe (décision D8) |
| 9 | **Marge nette** | Part du CA restant en résultat | résultat net ÷ CA | #7, #1 | idem | « — » si CA = 0 |
| 10 | **Créances clients** | Montant facturé non encore soldé | Σ factures émises − Σ paiements confirmés alloués − avoirs | factures, paiements, allocations | **instantané** à une date | exige l'**allocation** paiement→facture (prévue) |
| 11 | **Dettes fournisseurs** | Factures fournisseurs approuvées non payées | Σ `supplier_bills` approuvées non payées | `finance.operations` | instantané | saisie manuelle : exhaustivité non garantie |
| 12 | **Solde de trésorerie** | Argent disponible rapproché | Σ soldes des comptes bancaires (relevé le plus récent), avec **écart de rapprochement affiché** | relevés + rapprochements | instantané | « à la date du dernier relevé » toujours affiché |
| 13 | **Paiements en attente** | Paiements initiés non conclus | nombre et Σ des paiements `requires_action`/`processing` (avec ancienneté) | `payment` | instantané | dépend de la fraîcheur du prestataire |
| 14 | **Transactions échouées** | Paiements `failed`/`expired`/`cancelled` | nombre, Σ, **taux** = échecs ÷ tentatives, ventilé par motif et moyen | `payment` | période | un échec n'est pas une perte de CA |
| 15 | **Remboursements** | Montants remboursés et à rembourser | Σ remboursements réussis ; **à rembourser** = Σ `refund_pending` | `order`, `payment` | période / instantané | délai entre demande et exécution |
| 16 | **Frais de paiement** | Coût des moyens de paiement | Σ frais du prestataire ; **taux** = frais ÷ montants confirmés | `payment.fee_recorded`, règlements | période | **[PRESTATAIRE]** : les frais ne sont connus que si le prestataire les fournit |
| 17 | **Commissions** | — | **non applicable** (aucune place de marché) | — | — | à réactiver seulement si D1 change |
| 18 | **Rentabilité par activité** | Résultat par activité | (produits − charges) filtrés par axe `activity_id` | écritures analytiques | période | **nécessite** la définition des activités **[VOUS]** et l'affectation des charges communes |
| 19 | **Cash-flow opérationnel** | Flux de trésorerie d'exploitation | encaissements d'exploitation − décaissements d'exploitation (méthode directe) | relevés rapprochés | période | exige des relevés complets |
| 20 | **Prévision de trésorerie** | Solde futur estimé | solde actuel + encaissements attendus (factures échues/à échoir selon conditions de crédit) − décaissements connus (factures fournisseurs, récurrences, budget) | factures, factures fournisseurs, récurrences | 30/60/90 jours | **uniquement des flux connus** ; pas de statistique ni d'IA ; hypothèses **affichées** et versionnées |
| 21 | **Taux de croissance** | Évolution du CA | (CA − CA précédent) ÷ CA précédent | #1 | comparaison | « — » si la base précédente est nulle |
| 22 | **Ratio charges / revenus** | Poids des charges | total charges ÷ total produits | #5, #1 | période | |
| 23 | **Seuil de rentabilité** | CA minimal pour couvrir les charges fixes | charges fixes ÷ taux de marge sur coûts variables | charges classées **fixes/variables** + marge brute | période | **INDISPONIBLE** tant que #8 l'est |
| 24 | **Burn rate / runway** | Consommation de trésorerie | burn = décaissements nets mensuels si négatifs ; runway = trésorerie ÷ burn | #12, #19 | mensuel | affichés **uniquement** si burn > 0 |
| 25 | **CAC / LTV** | Coût d'acquisition ; valeur d'un client | CAC = dépenses « acquisition » ÷ nouveaux clients ; LTV = marge moyenne × commandes moyennes par client | dépenses marquées, clients (`identity`), commandes | période | LTV exige la marge (#8) et un historique suffisant ; sinon « — » |

---

## 12. Interfaces du backoffice

Principe : **distinguer clairement le pilotage commercial (existant) du pilotage comptable (nouveau).** Les deux ne montrent pas le même « chiffre d'affaires » (commandes passées vs revenus comptabilisés) : un écran **Réconciliation commercial ↔ comptable** explique l'écart, ligne par ligne, plutôt que de le laisser surprendre.

**Menu proposé**
- **Ventes** (existant au contrat) : Commandes, Devis, **Paiements**, **Remboursements**, **Événements du prestataire**, Factures. Ici vivent les flux de la plateforme.
- **Finance** (nouveau) : Comptabilité (écritures, journaux, grand livre, balance, périodes) · Opérations (factures fournisseurs, dépenses, autres revenus) · Trésorerie (banques, relevés, rapprochement, prévisions) · Contrôles (alertes, événements en quarantaine, approbations) · Budgets · Fiscalité · **Pilotage** (KPI, états) · Paramètres (règles de comptabilisation, plan, permissions d'alertes).

**Écrans clés (maquettes fonctionnelles)**

*Flux plateforme (écran Paiements, dans Ventes)* :
```
┌ Paiements ───────────── [Aujourd'hui ▾] [Statut ▾] [Moyen ▾] [Prestataire ▾]  [Exporter CSV] ┐
│ En attente : 12 · 1,9 M F CFA      Échoués (24 h) : 7 · taux 4,1 %      À rembourser : 3 · 0,4 M │
│ ──────────────────────────────────────────────────────────────────────────────────────────── │
│ Date | Commande | Client | Moyen | Montant | Statut | Prestataire (réf.) | Règlement | Écart   │
│ clic → détail : chronologie (initié → confirmé → règlement → rapproché), événements bruts,    │
│ remboursements liés, audit                                                                      │
└────────────────────────────────────────────────────────────────────────────────────────────────┘
```
(Disposition seulement : aucun chiffre réel avant les premières ventes.)

*Rapprochement à trois niveaux (Finance › Trésorerie)* :
```
 TRANSACTIONS (payment)        RÈGLEMENTS PRESTATAIRE           BANQUE (relevé importé)
 120 confirmées  14,2 M   ──►  4 règlements  13,9 M net   ──►  4 virements reçus 13,9 M
 écart brut/net = frais 0,3 M (expliqué)                        écart non résolu : 0   ✔
 [Non rapprochés : 2 paiements, 1 règlement]   [Écarts ouverts : 1] → assigner · justifier · écrire
```

*Écriture comptable* : lignes (compte, libellé, débit, crédit), totaux et indicateur d'équilibre, **source cliquable** (facture, paiement, événement), règle appliquée et sa version, historique, boutons « Valider » (approbateur ≠ auteur) et « Contre-passer » (motif obligatoire). **Pas de bouton « Supprimer »** sur une écriture validée.

*Contrôles* : événements **en quarantaine** (cause, dernier essai, bouton « Rejouer » après correction), brouillons d'écritures automatiques à approuver, alertes ouvertes, rattrapages en cours.

*Pilotage* : **construit en dernier** (§ 15). Avant, un écran **« Qualité des données »** affiche ce qui est comptabilisé, en attente, rapproché ou non (c'est lui qui conditionne la confiance aux KPI).

---

## 13. API et événements nécessaires

### 13.1 Événements : enveloppe commune (existante)
`event_id` (unique), `type` (versionné, ex. `invoice.issued.v1`), `occurred_at`, `aggregate_id`, `trace_id`, `actor_id/type`, **`aggregate_version`** (à ajouter : ordre par agrégat), `payload`. Montants : `int64` unités mineures + devise. Un champ **ajouté** reste compatible ; un champ retiré/renommé impose un nouveau `vN` (contrôle `buf breaking`, déjà en place).

| Événement | Émetteur | Contenu minimal du `payload` | Existe ? |
|---|---|---|---|
| `order.placed.v1` | order | order_id, number, user_id/company_id, currency, totaux HT/TVA/TTC, taux figé | oui (à compléter) |
| `invoice.issued.v1` | order | invoice_id, number, series, order_id, issued_at, **due_at**, tiers, HT, taxe(s), TTC, devise | **non** |
| `credit_note.issued.v1` | order | credit_note_id, number, **invoice_id d'origine**, refund_id, motif, montants | **non** |
| `order.cancelled.v1` | order | order_id, motif, `refund_pending` | oui |
| `payment.succeeded.v1` | payment | payment_id, order_id, montant, devise, prestataire, **réf. prestataire**, source (`provider`/`manual`), `received_at` | oui (à compléter) |
| `payment.failed.v1` | payment | payment_id, order_id, motif, moyen | oui |
| `payment.refunded.v1` | payment | payment_id, order_id, refund_id, montant | oui |
| `refund.succeeded.v1` / `.failed.v1` / `.requested.v1` | order/payment | refund_id, order_id, **payment_id d'origine**, canal, motif, montant | oui |
| `payment.fee_recorded.v1` | payment | payment_id ou settlement_id, montant des frais, devise, base de calcul | **non [PRESTATAIRE]** |
| `provider.settlement_received.v1` | payment | settlement_id, période couverte, brut, frais, **net**, date de valeur, réf. | **non [PRESTATAIRE]** |
| `payment.dispute_opened.v1` / `.closed.v1` | payment | dispute_id, payment_id, montant, motif, issue | **non [PRESTATAIRE]** |
| `quote.accepted.v1` | quote | quote_id, order_id | oui (par `converted`) |
| `finance.period_closed.v1` | finance | période, par, date | **non** |
| `finance.alert_raised.v1` | finance | règle, objet, gravité | **non** |

### 13.2 Idempotence, erreurs, reprise
1. **Idempotence** : `inbox_events.event_id` unique ; un événement déjà traité est **accusé sans effet**. Les écritures générées portent `(source_type, source_id, kind)` unique : même un bug de rejeu ne peut pas doubler une écriture (double verrou : événement **et** écriture).
2. **Statuts d'un événement** : `received → processed`. En cas d'échec : `failed` avec nouvelle tentative (attente exponentielle plafonnée) ; au-delà de N essais : **`quarantined`** (visible, jamais perdu, jamais silencieux), alerte `finance.alert_raised`.
3. **Ordre** : JetStream ne garantit pas l'ordre entre agrégats ni entre sujets. Les gestionnaires sont **tolérants à l'ordre** : si une dépendance manque (remboursement reçu avant le paiement), l'événement passe en `waiting_for` et est **repris automatiquement** à l'arrivée de la dépendance, ou à l'échéance du rattrapage. `aggregate_version` permet de détecter un trou.
4. **Événement empoisonné** (invalide) : quarantaine immédiate avec cause ; correction de la règle ou de la donnée, puis **rejeu manuel** tracé.
5. **Rejeu volontaire** : outil réservé (`finance:admin`) pour retraiter une plage d'événements depuis l'inbox, après correction d'une règle ; les écritures déjà générées ne sont **jamais écrasées** (une règle nouvelle produit, au besoin, des **contre-passations** plus de nouvelles écritures).

### 13.3 Rattrapage et contrôle de complétude
Une tâche périodique compare, **par jour et par type**, le **nombre et le total** émis par `order`/`payment` (lecture gRPC) à ce que `finance` a reçu : tout écart crée un **`reconciliation_gap`** et déclenche une **demande de réémission** (l'outbox peut republier à partir d'un point) ou une **reprise par lecture** de l'agrégat manquant. C'est la protection contre un événement perdu que personne n'aurait remarqué.

### 13.4 API REST `finance` (préfixe `/admin/finance/…`, permissions entre crochets)
| Domaine | Routes principales |
|---|---|
| Plan, journaux, périodes | `GET/POST /accounts`, `PUT /accounts/{id}`, `POST /accounts/{id}/deactivate` [read/admin] ; `GET/POST /journals`, `GET/POST /fiscal-years`, `POST /fiscal-periods/{id}/close\|reopen` [read/admin/close] |
| Écritures | `GET/POST /entries`, `GET /entries/{id}`, `POST /entries/{id}/submit`, `POST /entries/{id}/approve`, `POST /entries/{id}/reverse` [read/write/approve] |
| États | `GET /reports/general-ledger`, `/trial-balance`, `/income-statement`, `/balance-sheet`, `?format=csv\|xlsx\|pdf` [read/export] |
| Opérations | `GET/POST /supplier-bills`, `/expenses`, `/other-incomes`, `/recurring-templates`, `/third-parties` [read/write] |
| Intégration | `GET /inbox?status=`, `POST /inbox/{id}/retry`, `GET/POST /posting-rules`, `POST /posting-rules/{id}/approve` [read/admin/approve] |
| Trésorerie | `GET/POST /bank-accounts`, `POST /bank-statements/import`, `GET /reconciliations`, `POST /reconciliations`, `POST /reconciliations/{id}/resolve`, `GET /cash-forecast` [treasury:*] |
| Contrôles | `GET /alerts`, `POST /alerts/{id}/resolve`, `GET /control-rules`, `PUT /control-rules/{id}`, `GET /approvals` [read/approve/admin] |
| Budgets, fiscalité | `GET/POST /budgets`, `GET /budgets/{id}/variance`, `GET /tax/summary`, `GET/POST /tax-returns` [budget:*, tax:*] |
| KPI | `GET /kpis?period=&compare=&basis=`, `GET /kpis/{code}` (définition, formule, composition, complétude), `GET /kpis/{code}/trace` [read] |

**À ajouter aux services existants** (contrats provisoires) : `payment` : `GET /admin/provider-settlements`, `/provider-fees`, `/disputes` ; `order` : événements de facture/avoir, numérotation, échéance de facture.

---

## 14. Stratégie de tests et de sécurité

### 14.1 Tests
| Niveau | Contenu |
|---|---|
| **Contraintes en base** (PostgreSQL réel, comme le reste du projet) | écriture déséquilibrée refusée ; écriture validée immuable (y compris en SQL direct) ; période clôturée verrouillée ; unicité de source ; empreinte de relevé |
| **Propriétés** | pour des suites d'opérations aléatoires : balance toujours équilibrée ; le résultat = Σ produits − Σ charges ; contre-passation ⇒ solde net nul sur l'écriture |
| **Règles de comptabilisation** | jeux « événement → écritures attendues » (« golden ») **relus et signés par l'expert-comptable** ; toute modification de règle fait échouer le test jusqu'à nouvelle validation |
| **Événements** | doublon, **ordre inversé**, retard, perte puis rattrapage, événement invalide (quarantaine), rejeu complet ⇒ **mêmes écritures qu'au premier passage** |
| **Concurrence** | validation simultanée de deux écritures (numéros uniques et continus), approbation simultanée, rapprochement simultané de la même ligne |
| **Rapprochement** | injection d'écarts : ils restent visibles et ne disparaissent pas à la clôture |
| **KPI** | recalcul indépendant depuis les écritures (requête de référence) comparé à l'API ; traçabilité jusqu'à la source |
| **Permissions** | matrice complète permission × route (refus par défaut) ; **maker-checker** (y compris `*`) |
| **Contrats** | `buf breaking` sur les événements ; tests de compatibilité entre émetteur et consommateur |
| **Reprise** | restauration de sauvegarde puis contrôle de cohérence (soldes = Σ écritures) |

### 14.2 Sécurité des paiements et des données
| Menace | Parade |
|---|---|
| **Faux webhook** | **signature vérifiée** avec le secret du prestataire, comparaison en temps constant ; corps brut conservé (empreinte) ; rejet **avant** tout effet, avec journalisation (sans le secret) **[PRESTATAIRE : format exact]** |
| **Rejeu d'un webhook** | identifiant d'événement **unique** (`provider_events`) + fenêtre de validité sur l'horodatage ; un rejeu est accusé sans effet |
| **Montant falsifié** | on ne se fie pas au seul corps du webhook : le **statut et le montant sont re-vérifiés** auprès du prestataire avant de confirmer **[PRESTATAIRE : si son API le permet]** ; le montant confirmé doit égaler le dû |
| **Réponse trop précoce** | le webhook n'est accusé (2xx) **qu'après** écriture durable dans `provider_events` ; sinon le prestataire redélivre |
| **Secrets** | uniquement par l'environnement ou un coffre (comme la clé Brave), jamais en base ni en journal, rotation possible |
| **Données de carte** | **jamais** stockées ni transitant par nos serveurs (page de paiement du prestataire) ; seuls des jetons/références |
| **Données bancaires** | chiffrées (AES-GCM, clé hors base, comme la clé TOTP), masquées à l'affichage, absentes des journaux et de l'audit |
| **Fraude interne** | maker-checker, audit append-only, alertes de comportement, accès en lecture seule pour l'expert externe, exports tracés |
| **Altération du passé** | immuabilité en base, empreintes de relevés, clôtures de périodes |
| **Déni de service sur webhook** | limitation de débit par prestataire, validation de taille, mise en file avant traitement |
| **Perte de données** | sauvegardes chiffrées et **testées** (restauration réelle) ; journal d'événements conservé |

Toute alerte de sécurité est **non bloquante pour la comptabilité** (elle informe) ; seule la **signature invalide** bloque, car un webhook non authentifié n'est pas un événement.

---

## 15. Plan de réalisation par phases

Chaque phase : **autonome, testée, documentée, sécurisée, validée par vous avant la suivante.** Aucune ne commence sans votre accord.

### 15.1 Dépendances

```
 Catalogue (FAIT) ──► Import CSV ──► Site branché (catalogue réel)
        │
        ├──► [Pricing/Taxe] ──► [Panier] ──► [Commandes + Factures + Avoirs] ──► [Paiements] ──┐
        │                                                                                       ▼
        └─────────────────────────────────────────────────────────────────────────►  Finance F4 (intégration)
 Finance F1 (grand livre) ──► F2 (opérations) ──► F3 (trésorerie) ──────────────────────────────┘
                                                                              F4 ──► F5 fiscalité ──► F6 budgets ──► F7 KPI/pilotage
```

**Les phases F1 à F3 ne dépendent pas de la vente** (saisie manuelle) : elles peuvent avancer en parallèle de la construction des services de vente.

### 15.2 Phases

| Phase | Contenu | Prérequis | Livre | Critère de fin |
|---|---|---|---|---|
| **F0** | Décisions § 17, atelier avec l'expert-comptable (plan, journaux, principes d'écriture, règles de numérotation) | vous + expert | un cahier validé | décisions signées |
| **F1 Grand livre** | plan comptable, journaux, exercices/périodes, écritures (brouillon → validée), contre-passation, **maker-checker**, numérotation par compteur, grand livre, balance, audit, permissions `finance:*` | F0 | un grand livre fiable alimenté en **saisie manuelle** | tests de contraintes et de propriétés ; balance équilibrée |
| **F2 Opérations** | tiers, factures fournisseurs, dépenses, autres revenus, récurrences, génération d'**écritures** par `operations`, anti-doublon, pièces (si S3) | F1 | « combien je dépense » en saisie | écritures traçables jusqu'à la saisie |
| **F3 Trésorerie** | comptes bancaires (chiffrés), import de relevés CSV, rapprochement avec écarts conservés, prévisions v1 (flux connus) | F1, F2 | soldes réconciliés | écarts visibles ; relevés immuables |
| **S-A Prix/Taxes + Panier** *(hors finance)* | règles de taxe, taux de change, panier | catalogue, décision TVA **[EC]** | totaux figés | à planifier à part |
| **S-B Commandes + Factures + Avoirs** *(hors finance)* | commandes, **numérotation continue des factures**, avoirs, événements `invoice.issued`/`credit_note.issued` | S-A, F0 (règles de numérotation) | ventes réelles possibles | à planifier à part |
| **S-C Paiements** *(hors finance)* | adaptateur du prestataire, **webhooks sécurisés**, remboursements, rapprochement manuel, frais, règlements, litiges | S-B, documentation du prestataire | paiements réels | à planifier à part |
| **F4 Intégration des flux** | inbox, quarantaine, rattrapage, règles de comptabilisation (approuvées par l'expert), écritures automatiques de ventes/paiements/remboursements/frais/règlements, **rapprochement à trois niveaux** | F1–F3, S-B, S-C | comptabilité alimentée **sans ressaisie** | rejeu identique ; écarts expliqués |
| **F5 Fiscalité** | codes de taxe, TVA collectée/déductible calculée **d'après les écritures**, préparation déclarative, exports pour l'expert | F4, règles **[EC]** | données déclaratives | validées par l'expert |
| **F6 Budgets** | budgets, écarts, scénarios | F1 (F4 pour le réel automatique) | prévisionnel vs réel | |
| **F7 Pilotage** | KPI complets, rentabilité, tableaux de bord définitifs, réconciliation commercial↔comptable | F4+ et données fiables | le centre de pilotage | KPI = requêtes de référence |

**Règle** : **aucun tableau de bord définitif avant F7.** D'ici là, seuls les écrans « Qualité des données » et de contrôle existent.

---

## 16. Risques techniques et fonctionnels

| # | Risque | Gravité | Parade |
|---|---|---|---|
| 1 | Règles comptables mal comprises → états faux | **Élevée** | validation de chaque règle par l'expert, jeux « golden » signés, brouillons tant que non approuvées |
| 2 | Prestataire : documentation, signature, frais, règlements **inconnus** de moi | **Élevée** | lire la documentation officielle avant S-C ; adaptateur isolé ; ne rien présumer des frais/règlements |
| 3 | Événement perdu ou en double → écart silencieux | Élevée | outbox, inbox idempotente, unicités, **contrôle de complétude** quotidien |
| 4 | Ordre des événements inversé | Moyenne | gestionnaires tolérants, `waiting_for`, `aggregate_version` |
| 5 | Absence de donnée de coût → marge brute impossible | Moyenne | KPI marqués indisponibles ; décision D8 |
| 6 | Réglementation de facturation (numérotation, mentions, **facturation électronique normalisée éventuelle**) | **Élevée** | validation de l'expert **avant S-B** ; rien n'est supposé |
| 7 | Vous attendez des chiffres avant d'avoir des données fiables | Moyenne | écrans de qualité des données d'abord ; pas de tableau de bord prématuré |
| 8 | Portée gigantesque → dérive | **Élevée** | phases validées, périmètre minimal, ce qui n'est pas décidé n'est pas construit |
| 9 | Fraude interne (rembourser ou saisir pour soi) | Moyenne | maker-checker pour tous, alertes, audit |
| 10 | Dépendance à un prestataire unique (GeniusPay) | Moyenne | abstraction multi-prestataires prévue ; continuité à décider |
| 11 | Écart entre « CA commercial » et « CA comptable » mal compris | Moyenne | écran de réconciliation expliquant chaque écart |
| 12 | Surcharge de la saisie manuelle en F2 (activité réelle faible au départ) | Faible | commencer simple ; importer en lot si besoin |
| 13 | Charge de l'exploitation (sauvegardes, clés, rotation) | Moyenne | procédure écrite, test de restauration |
| 14 | Pièces justificatives sans stockage | Moyenne | décision S3 ; sinon les pièces restent hors système (référence seulement) |

---

## 17. Décisions restant à prendre (rien n'est validé)

**Modèle d'affaires et périmètre**
- **D1.** Confirmez-vous que FilterFinder **vend son propre catalogue** et qu'il n'y aura **ni commissions, ni reversements, ni vendeurs tiers** ? *(Ma lecture : oui.)* Si un jour il y en a, c'est un projet distinct.
- **D2.** Rôle du système : **alimente-t-il** votre expert-comptable (grand livre interne rigoureux + exports ; l'expert reste garant légal), ou **remplace-t-il** son logiciel ? *(Recommandation : alimenter.)*

**Comptabilité et fiscalité** (à faire valider par l'expert-comptable **[EC]**)
- **D3.** Référentiel comptable et plan : **SYSCOHADA révisé** ? Journaux utilisés ?
- **D4.** Régime d'imposition, **TVA** (assujettissement, taux, règles sur les envois), exercice (**année civile** ?), moment de constatation du revenu (**`invoice_issuance_moment`** : commande, expédition ou livraison).
- **D5.** Facturation : **numérotation continue obligatoire ?** séries, mentions légales, **facturation électronique normalisée éventuelle en Côte d'Ivoire**, durée de conservation.
- **D6.** Traitement comptable du **compte de transit prestataire**, des **frais**, des **litiges** (provision ?), des **remboursements en attente**.

**Paiements et flux** (dépendent du prestataire **[PRESTATAIRE]**)
- **D7.** Le prestataire choisi (**GeniusPay**, au contrat provisoire) fournit-il : signature des webhooks, **frais par transaction**, **rapports de règlement**, API de vérification, gestion des litiges ? Pouvez-vous me donner **la documentation officielle** ? *Sans elle, S-C ne peut pas être conçue précisément.*
- **D8.** **Prix d'achat des produits** : acceptez-vous d'ajouter un **coût d'achat** (accessible seulement aux rôles finance) au catalogue, pour permettre la marge brute ? Sinon la marge brute et le seuil de rentabilité restent indisponibles.
- **D9.** Moyens de paiement au lancement (confirmé : `mobile_money`, `card`, `bank_transfer` ?) ; **paiement à la livraison** et **facture à terme B2B** : actifs un jour ? (Désactivés au lancement dans le contrat.)

**Organisation et sécurité**
- **D10.** Activités et départements pour la rentabilité et les budgets **[VOUS]** (je n'en invente aucun).
- **D11.** **Seuil de double validation** en F CFA et **liste des opérations sensibles** ; qui approuve ?
- **D12.** Règles d'alerte à activer et **lesquelles sont bloquantes** (par défaut aucune).
- **D13.** Rôles réels (dirigeant, comptable, responsable, expert externe, auditeur) : qui existe, combien de personnes ? Le **maker-checker s'applique-t-il aussi au super-administrateur** ? *(Recommandation : oui, sinon il ne protège rien.)*
- **D14.** **Stockage S3** des pièces justificatives et des PDF : où ? (déjà en attente pour les images produits.)

**Séquencement**
- **D15.** Ordre : finir le **catalogue** (import CSV, site branché), puis **S-A/S-B/S-C** (la vente) en parallèle de **F1 à F3** ? Je recommande de ne démarrer **F1** qu'après **F0** (atelier avec l'expert).
- **D16.** Reprise de l'existant : **soldes d'ouverture** et historique à importer ?

---

## 18. Recommandations et changements par rapport à la v1

**Ce que je modifie**
1. **Deux domaines, trois services** : la finance observe, elle ne possède plus factures, devis ni paiements.
2. **Un seul sens de dépendance** (`order`/`payment` → `finance`), pas de cycle ; la finance peut lire, jamais écrire chez les autres.
3. **Cinq états d'une vente** et **deux sortes d'encaissement** (paiement confirmé vs argent en banque).
4. **Règles de comptabilisation en données**, versionnées, **approuvées par l'expert** ; brouillons tant qu'elles ne le sont pas.
5. **Numérotation en quatre familles**, la continuité sans trou **seulement** là où elle est exigée, et avec ses contraintes expliquées.
6. **Intégrité (bloquante) séparée des alertes métier (non bloquantes sans votre décision).**
7. **Inbox idempotente, quarantaine, rattrapage** pour ne jamais perdre ni doubler un événement.
8. **Marge brute et seuil de rentabilité marqués indisponibles** tant que le coût d'achat n'existe pas.
9. **Plan de réalisation éclaté** : la vente (commandes, paiements) devient ses propres phases validées ; le tableau de bord définitif passe en dernier.

**Ce que je recommande**
- Commencer par **F0 (décisions et atelier avec l'expert-comptable)**, puis **F1 en parallèle** de la construction de la vente.
- **Finir le catalogue d'abord** (import CSV, site branché) : sans site réel, aucune vente ni donnée n'existe.
- **Obtenir la documentation du prestataire de paiement** avant toute conception détaillée des paiements.
- Garder le **périmètre minimal** : tant que D1 n'est pas changé, pas de commissions ni de reversements.

**Les décisions que j'attends de vous en priorité** : **D1** (modèle d'affaires), **D2** (alimenter ou remplacer l'expert), **D7** (documentation du prestataire), **D8** (coût d'achat produit), **D13** (maker-checker pour tous), **D15** (séquencement). Les points **[EC]** (D3 à D6) sont à transmettre à votre expert-comptable.

**Je n'écris aucun code, contrat ni migration avant votre validation explicite.**

---

## 19. Avancement réel (17 nov. 2026) : F1, F2 et F3 livrées

Votre validation reçue : « exécute F1, puis F2 et F3 ». Les trois phases sont **développées, testées et branchées au backoffice**. Rien n'est commité (en attente de votre autorisation). Ce qui suit décrit ce qui existe réellement dans le code, pas une intention.

### F1 — Socle comptable (livré)
- Service `backend/services/finance` (hexagonal, gRPC port 9005, base `filterfinder_finance`, migration `00001_ledger.sql`).
- Plan de comptes (import CSV, désactivation jamais suppression), journaux, exercices et **périodes mensuelles** (clôture refusée s'il reste des brouillons ; réouverture uniquement via une **demande approuvée par une autre personne**, consommée une seule fois).
- Écritures : brouillon → validée ; une écriture validée est **immuable par triggers en base** (équilibre débit/crédit, période ouverte, comptes et journal actifs) ; **double validation systématique** (`posted_by ≠ created_by`, seuil `four_eyes_min_amount`, 0 = toujours, super-admin inclus) ; **numérotation sans trou** par (journal, exercice) attribuée **au moment de la validation** sous verrou ; correction uniquement par **contre-passation** (écriture inverse liée, une seule par origine).
- Rapports calculés en SQL : grand livre (solde progressif), balance, **exports CSV** (BOM UTF-8, neutralisation des formules, jeton jamais exposé au navigateur).
- Backoffice : `/finance` (accueil), écritures (éditeur multi-lignes), comptes (+ import), journaux, périodes, approbations, rapports, réglages.

### F2 — Dépenses et revenus hors plateforme (livré)
- Migration `00002_operations.sql`. Tiers **par référence** (aucune copie de données personnelles), catégories d'opérations **configurables** (catégorie → compte), opérations dépense/revenu (montant HT + taxe, **compte de taxe choisi par vous**, jamais déduit d'une règle inventée), règlement immédiat ou en compte, **soupçon de doublon à confirmation explicite**, opération **figée par trigger** dès qu'une écriture lui est liée ; statut dérivé de l'état de l'écriture.
- Règlements partiels (écritures de règlement en brouillon, reste calculé), **modèles récurrents** (génération idempotente, une occurrence par mois au plus), **pièces jointes** (PDF/PNG/JPEG par détection de contenu, ≤ 5 Mio, dédoublonnage SHA-256, stockées en base derrière un port — déplaçables vers S3 quand l'emplacement sera fixé).
- Backoffice : opérations, tiers, catégories, modèles récurrents, téléchargement de pièce via le serveur.

### F3 — Trésorerie (livré)
- Migration `00003_treasury.sql`. Comptes bancaires : **numéro chiffré AES-256-GCM** (clé hors base, `FINANCE_BANK_KEY_FILE`), seuls les **4 derniers chiffres** sortent du service ; un compte du plan comptable lié par compte bancaire.
- Relevés **immuables** : import CSV tout-ou-rien avec **mode simulation**, même fichier refusé (empreinte), relevés qui se recouvrent → lignes déjà connues ignorées, lignes antérieures à la date d'ouverture refusées.
- Rapprochement : allocations de montant entre ligne de relevé et ligne d'écriture **validée** du compte (triggers : même sens, bon compte, pas de sur-allocation, verrous) ; **les suggestions ne sont jamais appliquées seules** ; retrait possible ; **l'écart n'est jamais absorbé** : synthèse avec écart d'ouverture et un contrôle de cohérence qui doit valoir 0.
- Prévision de trésorerie v1 : **flux connus uniquement** (opérations comptabilisées à échéance, retards et sans-échéance à part, occurrences récurrentes futures), brouillons exclus et signalés, horizons 30/60/90/180 j, hypothèses affichées.
- Backoffice : comptes bancaires (fiche avec explication de l'écart), import, mouvements, écran de rapprochement deux colonnes, prévisions.

### Permissions, contrat, tests
- Permissions ajoutées : `finance:read|write|approve|close|admin|export`, `treasury:read|write|reconcile` (catalogue identity = contrat, vérifié par test).
- Contrat : ~55 routes `/v1/admin/finance/...` (OpenAPI + proto `financeadmin/v1` + mapping), `check_contract.py` : **COHÉRENT** ; `buf lint` : propre.
- Tests exécutés : intégration PostgreSQL du service finance (immutabilité par SQL direct, double validation, numérotation concurrente, relevés, rapprochement), matrices de permissions au gateway, `tsc`/`eslint` propres, **E2E navigateur** F1/F2/F3 (deux utilisateurs, zéro erreur JS, numéro de compte jamais affiché).

### Décisions toujours ouvertes
D1 modèle d'affaires, D3–D6 **[EC]** (plan de comptes réel, TVA, règles de numérotation, dates d'exercice — valeurs de test en attendant), D7 documentation GeniusPay, D8 coût d'achat, emplacement S3, seuil de double validation (0 pour l'instant). **F4–F7 n'ont pas commencé** et attendent votre validation de F1–F3.
