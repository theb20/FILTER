# Module financier : analyse, architecture et décisions à trancher, avant tout code

Statut : **PROPOSITION, en attente de vos décisions.** Aucun code n'est écrit. Source : votre prompt « Conception d'un système financier centralisé » (10 modules A à J), et l'état réel du projet (contrat, services, données).

---

## 0. À lire d'abord : ce que cela représente, et trois points qui changent le plan

**Ampleur.** Les dix modules A à J forment un logiciel de gestion complet (comptabilité en partie double, facturation, rapprochement bancaire, fiscalité, budgets). C'est plusieurs mois de travail si on veut qu'il soit fiable. Je propose de le livrer **par phases validées** (§ 9), chacune utilisable seule, et de commencer par ce dont tout le reste dépend : **le grand livre**.

**Point 1 : un système comptable ne devient « légal » que validé par votre expert-comptable.** Je peux construire un grand livre rigoureux (écritures équilibrées, non modifiables, régularisations, clôtures) et des exports propres pour lui. Je ne peux pas, seul, garantir la conformité réglementaire (plan comptable, mentions des factures, déclaration de TVA, facturation électronique). Chaque règle de ce type ci-dessous est marquée **[À CONFIRMER : expert-comptable]**. Je n'en invente aucune.

**Point 2 : le module dépend de données qui n'existent pas encore.** Les **commandes, paiements, devis, remboursements** sont au stade « brouillon » du contrat et **aucun service ne les porte**. Sans eux, le chiffre d'affaires de la boutique ne peut pas alimenter la finance automatiquement. Deux conséquences :
- Les phases 1 à 3 fonctionnent avec de la **saisie manuelle** (revenus, dépenses, banque), sans attendre.
- L'alimentation automatique depuis les ventes en ligne exige de construire d'abord le service Commandes/Paiements (§ 9, phase 4).

**Point 3 : « aucune donnée fictive ».** Tant qu'aucune opération n'est saisie, chaque écran affiche un **état vide explicite** (« Aucune opération enregistrée »), jamais des chiffres d'exemple. Les KPI sans données affichent « — » et non zéro (un zéro ment : il dit « on a mesuré, c'est nul »).

---

## 1. Analyse fonctionnelle : ce que le dirigeant doit comprendre, et ce qui le permet

| Question du dirigeant (votre § 8) | Donnée nécessaire | Module |
|---|---|---|
| Combien l'entreprise gagne | produits d'exploitation, encaissements | B, F, H |
| Combien elle dépense | charges, décaissements | C, F |
| Où va l'argent | charges par catégorie, département, fournisseur | C, G |
| Ce qu'elle doit payer | factures fournisseurs échues et à échoir, taxes | C, E, I |
| Ce qu'elle doit encaisser | factures clients échues et à échoir | E |
| Quelle est sa rentabilité | marges par activité, seuil de rentabilité | H |
| Sa trésorerie | soldes bancaires, flux prévisionnels | D |
| Quelles décisions prendre | écarts budget/réel, alertes, scénarios | G, A |

**Principe directeur : une seule source de vérité, le grand livre.** Revenus, dépenses, trésorerie, résultat, bilan et TVA ne sont **pas** des tables séparées qu'il faudrait faire concorder : ce sont des **vues calculées** à partir d'écritures comptables. C'est ce qui garantit « les soldes sont réconciliables » et « les calculs financiers sont centralisés dans le backend » (votre § 6).

---

## 2. Cartographie des modules

```
                     ┌───────────────────────── J. Sécurité & audit (transverse) ─────────────────────────┐
                     │                                                                                     │
 Sources de données  │   ┌─────────────────────────────────────────────────────────────┐                  │
 ─────────────────   │   │                 F. COMPTABILITÉ (cœur)                      │                  │
 B. Revenus (saisie) ┼──►│  plan comptable · journaux · écritures · périodes · clôture │──► G. Budgets    │
 C. Dépenses (saisie)┼──►│  grand livre · balance · compte de résultat · bilan         │──► H. Rentabilité│
 D. Banque (relevés) ┼──►│  rapprochement bancaire · export comptable                  │──► I. Fiscalité  │
 E. Factures/devis   ┼──►│                                                             │──► A. Dashboard  │
 Ventes en ligne     ┼──►└─────────────────────────────────────────────────────────────┘     (KPI)        │
 (phase ultérieure)  │                                                                                     │
                     └─────────────────────────────────────────────────────────────────────────────────────┘
```

| Module | Rôle | Dépend de | Phase |
|---|---|---|---|
| **F. Comptabilité** | Grand livre, journaux, périodes, états | rien (fondation) | **1** |
| **C. Dépenses** + **B. Revenus** | Saisie qualifiée qui génère des écritures | F | **2** |
| **D. Trésorerie** | Comptes, mouvements, rapprochement | F, B, C | **3** |
| **E. Facturation** | Devis, factures, avoirs, relances, PDF | F, clients, produits | **4** |
| **I. Fiscalité** | TVA collectée/déductible, déclarations | F, E | **5** |
| **G. Budgets** | Budgets, écarts, scénarios | F | **6** |
| **H. Rentabilité** + **A. Dashboard** | KPI, marges, seuil, CAC/LTV | F, G, ventes | **7** |
| **J. Sécurité & audit** | Transverse : dès la phase 1 | identité, audit existants | **toutes** |

---

## 3. Architecture technique

Elle prolonge celle qui existe déjà (contrat d'abord, services hexagonaux, passerelle REST, base par service, outbox + NATS, audit centralisé).

**Nouveau service `finance`** (comme `catalog` et `sourcing`) :
- Base PostgreSQL **dédiée** `filterfinder_finance` : les données financières sont isolées (sauvegardes, accès et chiffrement séparables).
- Communication interne en gRPC ; le gateway expose les routes `/admin/finance/...` avec **permissions vérifiées avant relais**.
- Journal d'audit : les actions passent par l'outbox vers le journal d'audit append-only **existant** (déjà en place pour sourcing et catalog), en plus du journal **comptable** propre à la finance (§ 4).
- **Aucun calcul dans l'interface** : le backoffice affiche ce que l'API renvoie (KPI, soldes, états).
- Alimentation depuis les autres services par **événements idempotents** (même mécanisme que l'audit) : `order.paid`, `refund.issued`, `quote.accepted`… (phase 4).

**Montants et devises.**
- Tous les montants en **entiers en unités mineures** (XOF : le franc, sans décimale ; EUR : le centime), type `bigint` + code devise. C'est déjà la convention du contrat (`Money`). **Aucun flottant, jamais.**
- Taux (change, TVA) en `numeric` exact, pas en flottant.
- Devise de **tenue de la comptabilité** : XOF (confirmée). Une opération en devise étrangère conserve son montant d'origine, son taux appliqué et sa date ; la contre-valeur XOF est figée dans l'écriture. Les écarts de change sont des écritures (jamais un recalcul silencieux).

**Sécurité.**
- Permissions dédiées et fines (§ 8) ; le joker `*` du super-admin reste le seul à tout couvrir.
- **Double validation** des opérations sensibles (§ 7, règle R9) : l'auteur ne valide jamais sa propre écriture.
- Chiffrement : base chiffrée au repos (infrastructure) ; **numéros de compte bancaire masqués** à l'affichage (4 derniers caractères), jamais dans les journaux ni l'audit.
- Aucune donnée financière dans les URL ni les journaux applicatifs.

**Tests.** Même exigence que le reste : mêmes suites sur PostgreSQL réel, contraintes vérifiées **en base** (écriture déséquilibrée refusée, modification d'une écriture validée refusée, période clôturée verrouillée, doublon refusé), concurrence (numérotation sans trou sous charge), **vérification d'équivalence des calculs** (le compte de résultat doit égaler la somme des écritures ; la balance doit être équilibrée ; un test de propriété génère des écritures aléatoires et vérifie les invariants).

---

## 4. Modèle de données (cœur)

Les noms sont indicatifs ; le détail sera figé dans le contrat de chaque phase.

**Référentiel**
- `accounts` : plan comptable. `code` (texte), `label`, `class` (1 à 9), `type` (actif, passif, charge, produit, tiers, trésorerie), `parent_id`, `active`. Code unique ; compte utilisé non supprimable (désactivable).
- `journals` : `code` (VT ventes, AC achats, BQ banque, CA caisse, OD opérations diverses), `label`, `type`.
- `fiscal_years` / `fiscal_periods` : exercice, périodes mensuelles, état `open` → `closed` (verrou).
- `currencies`, `fx_rates` : réutilise l'existant du contrat « Prix » (taux datés).

**Écritures (le cœur, non modifiable)**
- `entries` (en-tête) : `id`, `journal_id`, `number` (séquence **sans trou** par journal et exercice), `date`, `period_id`, `label`, `status` (`draft` → `posted`), `source_type` / `source_id` (facture, dépense, relevé…), `reverses_entry_id` (régularisation), `created_by`, `posted_by`, `posted_at`.
- `entry_lines` : `entry_id`, `account_id`, `debit`, `credit` (entiers, un seul des deux non nul), `currency`, `amount_original`, `fx_rate`, `third_party_id`, `department_id`, `activity_id` (analytique), `memo`.
- **Contraintes en base** : somme des débits = somme des crédits (vérifiée à la validation) ; une écriture `posted` n'est **ni modifiable ni supprimable** (trigger, comme le journal d'audit) ; une période `closed` refuse toute nouvelle écriture ; clé d'unicité sur `(source_type, source_id, kind)` contre les doublons.
- **Correction = contre-passation** : on crée une nouvelle écriture inverse (liée par `reverses_entry_id`) puis la bonne écriture. L'historique reste lisible.

**Tiers et opérations**
- `third_parties` : clients et fournisseurs (le client existe déjà côté identité ; la finance en garde une **référence**, pas une copie des données personnelles).
- `expenses`, `incomes` : opérations saisies, qualifiées (catégorie, département, fixe/variable, récurrente), qui **produisent** leurs écritures.
- `recurring_templates` : charges et revenus récurrents.
- `documents` : pièces justificatives (facture fournisseur scannée…) : nécessite le stockage S3 (en attente de votre décision).

**Trésorerie**
- `bank_accounts` (nom, banque, devise, IBAN/numéro **chiffré**, compte comptable lié, solde d'ouverture).
- `bank_statements`, `bank_statement_lines` : import de relevés (CSV) ; `reconciliations` (ligne de relevé ↔ écriture, avec écart).

**Facturation**
- `quotes`, `invoices`, `invoice_lines`, `credit_notes`, `payments`, `payment_allocations`, `dunning_events` (relances). Numérotation **séquentielle sans trou**, facture émise **non modifiable** (avoir pour corriger).

**Budgets**
- `budgets` (année, version, statut), `budget_lines` (compte × période × département), `scenarios`.

**Fiscalité**
- `tax_codes` (taux, compte de collecte, compte de déduction), `tax_returns` (période, montants calculés, statut, référence de dépôt).

**Audit financier**
- `finance_audit` (en plus de l'audit central) : avant/après, auteur, validateur, motif **obligatoire** pour toute correction.
- `anomalies` : règles déclenchées (§ 7, R11), statut (à examiner, justifiée, confirmée).

---

## 5. Indicateurs (KPI) : définition, formule, source

Tous calculés **par le backend** à partir des écritures validées (`posted`). Chaque KPI renvoie : valeur, **période**, **comparaison** (période précédente et même période N-1), **définition** et **formule** (affichées au survol), et la liste des comptes qui le composent (traçabilité jusqu'à l'écriture).

| KPI | Formule | Remarque |
|---|---|---|
| Chiffre d'affaires | Σ produits d'exploitation (classe 7, hors produits financiers et exceptionnels) | périmètre exact **[À CONFIRMER : expert-comptable]** |
| Résultat d'exploitation | produits d'exploitation − charges d'exploitation | |
| Résultat net | total produits − total charges (− impôt sur les résultats) | |
| Marge brute | (CA − coût d'achat des marchandises vendues) ÷ CA | exige le coût d'achat par vente |
| Marge nette | résultat net ÷ CA | « — » si CA = 0 |
| Taux de croissance | (CA période − CA période précédente) ÷ CA période précédente | « — » si base nulle |
| Ratio charges / revenus | total charges ÷ total produits | |
| Solde de trésorerie | Σ soldes des comptes de trésorerie | **réconciliable** avec les relevés |
| Cash-flow opérationnel | encaissements d'exploitation − décaissements d'exploitation | méthode directe |
| Seuil de rentabilité | charges fixes ÷ taux de marge sur coûts variables | exige la classification fixe/variable |
| Burn rate | (décaissements − encaissements) mensuels moyens, si négatif | uniquement si pertinent |
| Runway | trésorerie ÷ burn rate | uniquement si burn > 0 |
| BFR | créances clients + stocks − dettes fournisseurs | stocks : n'existent pas encore |
| ROI | (gain − coût) ÷ coût, par investissement déclaré | |
| CAC | dépenses d'acquisition ÷ nouveaux clients | exige le marquage des dépenses « acquisition » |
| LTV | marge moyenne par commande × commandes moyennes par client | « si pertinent » : uniquement avec assez d'historique |

---

## 6. Interface : maquettes fonctionnelles

**Navigation proposée (menu « Finance »)** : Tableau de bord · Revenus · Dépenses · Trésorerie · Facturation · Comptabilité · Budgets · Rentabilité · Fiscalité · Contrôles. Chaque entrée n'apparaît que si le membre a la permission.

**Tableau de bord financier (module A)**

```
┌ Finance · Tableau de bord ─────────────── Période : [Ce mois ▾]  Comparer à : [Mois précédent ▾]  [Exporter ▾] ┐
│ ┌─ Chiffre d'affaires ─┐ ┌─ Résultat net ──────┐ ┌─ Marge nette ───┐ ┌─ Trésorerie ────────┐              │
│ │ 12 480 000 F CFA     │ │  1 930 000 F CFA    │ │ 15,5 %          │ │ 8 210 000 F CFA     │              │
│ │ ▲ +8,2 % vs préc.    │ │ ▼ −3,1 % vs préc.   │ │ ▼ −1,7 pt       │ │ Runway : — (positif)│              │
│ │ (i) définition       │ │ (i) formule         │ │                 │ │ rapproché au 14/11  │              │
│ └──────────────────────┘ └─────────────────────┘ └─────────────────┘ └─────────────────────┘              │
│ ┌─ Revenus / dépenses / résultat (12 mois) ──────────────┐ ┌─ Où va l'argent ─────────────────────┐       │
│ │  barres + courbe, survol = valeur, clic = écritures     │ │  charges par catégorie (anneau)      │       │
│ └─────────────────────────────────────────────────────────┘ └──────────────────────────────────────┘       │
│ ┌─ À encaisser ───────────────┐ ┌─ À payer ──────────────────┐ ┌─ Alertes ───────────────────────────┐    │
│ │ 6 factures échues · 3,2 M   │ │ 4 échéances sous 7 jours   │ │ ⚠ Budget Marketing dépassé de 12 %  │    │
│ │ [Voir]                      │ │ [Voir]                     │ │ ⚠ Trésorerie < seuil dans 21 jours  │    │
│ └─────────────────────────────┘ └────────────────────────────┘ └─────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```
(Les chiffres sont des **exemples de mise en page**, pas des données ; l'écran réel affichera « Aucune opération enregistrée » tant que rien n'est saisi.)

**Saisie d'une dépense (module C)** : formulaire à gauche (date, fournisseur, montant TTC/HT/TVA, catégorie, département, fixe/variable, mode de paiement, pièce jointe) ; à droite, **l'écriture comptable générée** affichée en direct (compte débité / crédité) avant validation. Les écritures ne sont pas cachées à l'utilisateur.

**Écriture comptable (module F)** : tableau lignes (compte, libellé, débit, crédit), totaux, **indicateur d'équilibre**, statut, historique, bouton « Contre-passer » (motif obligatoire) plutôt que « Supprimer ».

**Rapprochement bancaire (module D)** : deux colonnes, lignes du relevé importé à gauche, écritures non rapprochées à droite ; suggestions automatiques (même montant, date proche, libellé proche) à **confirmer manuellement** ; l'écart restant s'affiche en permanence et doit atteindre zéro.

---

## 7. Règles de gestion

| N° | Règle |
|---|---|
| R1 | Toute opération financière produit des **écritures** ; aucun montant n'existe hors du grand livre. |
| R2 | Une écriture n'est validée que si **débits = crédits**, à la devise et à l'unité mineure près. |
| R3 | Une écriture **validée** n'est ni modifiable ni supprimable (base). Seule la **contre-passation** corrige. |
| R4 | Un **brouillon** peut être modifié ou supprimé ; la suppression est tracée. |
| R5 | La numérotation des écritures et des factures est **continue, sans trou**, par séquence (journal, exercice). |
| R6 | Une période **clôturée** refuse toute écriture (y compris contre-passations datées dedans) ; la réouverture exige la permission de clôture, un motif et laisse une trace. |
| R7 | Les montants sont des **entiers en unités mineures** ; les taux sont exacts ; l'arrondi suit une règle unique, documentée, appliquée par ligne (règle à valider avec l'expert-comptable). |
| R8 | Une facture **émise** est figée ; une erreur se corrige par un **avoir** et une nouvelle facture. |
| R9 | **Double validation** : l'auteur d'une opération sensible ne la valide pas lui-même. Opérations sensibles proposées : contre-passation, réouverture de période, rapprochement forcé, écriture manuelle, montant au-dessus d'un seuil **[À REMPLIR : seuil en F CFA]**. |
| R10 | **Anti-doublon** : une même source (facture, relevé, paiement) ne produit qu'**une** écriture (contrainte d'unicité) ; une dépense avec même fournisseur + même numéro de pièce + même montant est signalée avant enregistrement. |
| R11 | **Détection d'anomalies** (règles explicites, pas de « magie ») : montant très supérieur à l'historique de la catégorie, écriture un week-end/jour férié, dépense sans pièce au-dessus d'un seuil, rupture de séquence, solde négatif inattendu, ajustement répété par le même auteur. Une anomalie **n'interdit rien** : elle demande un examen et une justification tracée. |
| R12 | Les **soldes de trésorerie sont réconciliables** : solde du grand livre = solde du relevé ± éléments en rapprochement identifiés ; un écart non expliqué est une alerte. |
| R13 | Un **compte comptable utilisé** ne se supprime pas (désactivation). Un tiers lié à des écritures non plus. |
| R14 | **Aucune donnée fictive** : pas d'amorçage de chiffres ; le plan comptable livré est un **modèle**, activé après votre validation. |
| R15 | Tout accès respecte les **permissions** ; lecture et écriture, validation, clôture et export sont des droits distincts. |
| R16 | Les corrections et exports sont **tracés** (qui, quand, pourquoi, quoi). |

---

## 8. Permissions proposées

`finance:read` · `finance:write` (saisir, brouillons) · `finance:validate` (valider, contre-passer) · `finance:close` (clôturer/rouvrir les périodes) · `finance:invoice` (émettre factures et avoirs) · `finance:budget` · `finance:tax` · `finance:export` · `finance:admin` (plan comptable, paramètres). Une lecture seule `finance:read` suffit à un dirigeant ; l'expert-comptable externe n'a besoin que de `finance:read` + `finance:export`.

---

## 9. Plan de réalisation par phases (chacune validée avant la suivante)

| Phase | Contenu | Livre |
|---|---|---|
| **1. Fondation comptable** | Plan comptable configurable, journaux, exercices et périodes, écritures (brouillon → validée), contre-passation, grand livre, balance, audit, double validation | un grand livre fiable, saisie d'OD manuelle |
| **2. Dépenses et revenus** | Saisie qualifiée, catégories, récurrences, tiers, génération des écritures, anti-doublon | « combien je dépense / gagne » en saisie |
| **3. Trésorerie** | Comptes bancaires, import de relevés CSV, rapprochement, prévisions simples, alertes | soldes réconciliés |
| **4. Ventes et facturation** | Service Commandes/Paiements + devis, factures, avoirs, PDF, échéances, relances, alimentation automatique de la finance | CA réel automatique |
| **5. Fiscalité** | Codes de taxe, TVA collectée/déductible, préparation déclarative, exports | données pour l'expert-comptable |
| **6. Budgets** | Budgets annuels/mensuels, départements, écarts, dépassements, scénarios | prévisionnel vs réel |
| **7. Rentabilité et tableau de bord** | KPI complets, marges par activité, seuil, ROI, CAC/LTV, comparaison de périodes, alertes | la vue du dirigeant |

Je recommande de **ne pas construire le tableau de bord en premier** (le contraire de la tentation habituelle) : un graphique sur des données non fiables est pire que pas de graphique.

---

## 10. API nécessaires (aperçu, détail par phase)

Préfixe `/admin/finance/…`, JSON, montants en entiers + devise, pagination et filtres communs (date, catégorie, statut).

| Domaine | Routes principales | Permission |
|---|---|---|
| Plan comptable | `GET/POST /accounts`, `GET/PUT /accounts/{id}`, désactiver | read / admin |
| Journaux, exercices | `GET/POST /journals`, `GET/POST /fiscal-years`, `POST /fiscal-periods/{id}/close\|reopen` | read / admin / close |
| Écritures | `GET/POST /entries`, `GET /entries/{id}`, `POST /entries/{id}/post`, `POST /entries/{id}/reverse` | read / write / validate |
| États | `GET /reports/general-ledger`, `/trial-balance`, `/income-statement`, `/balance-sheet` (+ `?format=csv\|xlsx\|pdf`) | read / export |
| Dépenses, revenus | `GET/POST /expenses`, `/incomes`, `/recurring`, catégories | read / write |
| Tiers | `GET/POST /third-parties` | read / write |
| Trésorerie | `GET/POST /bank-accounts`, `POST /bank-statements/import`, `GET/POST /reconciliations`, `GET /cash-forecast` | read / write |
| Facturation | `GET/POST /quotes`, `/invoices`, `/credit-notes`, `POST /invoices/{id}/issue\|send\|remind`, `GET /invoices/{id}/pdf` | read / invoice |
| Budgets | `GET/POST /budgets`, `GET /budgets/{id}/variance`, `POST /scenarios` | read / budget |
| Fiscalité | `GET/POST /tax-codes`, `GET /tax/vat-summary`, `GET/POST /tax-returns` | read / tax |
| KPI | `GET /kpis?period=&compare=`, `GET /kpis/{code}` (définition, formule, composition) | read |
| Contrôles | `GET /anomalies`, `POST /anomalies/{id}/resolve`, `GET /audit` | read / validate |

---

## 11. Décisions que vous devez prendre (avec ma recommandation)

| N° | Question | Ma recommandation |
|---|---|---|
| 1 | **Rôle du système** : remplace-t-il le logiciel comptable de votre expert-comptable, ou le **nourrit-il** ? | **Le nourrir** : grand livre interne rigoureux + exports ; l'expert-comptable reste le garant légal. Remplacer entièrement est possible plus tard, après son aval. **[À décider : vous]** |
| 2 | **Référentiel comptable** (plan comptable) | Le plan du **SYSCOHADA révisé** (zone OHADA dont fait partie la Côte d'Ivoire). **[À CONFIRMER : expert-comptable]** ; il fournira son plan et ses journaux. |
| 3 | **Régime fiscal et TVA** : régime d'imposition, assujettissement, **taux applicables** | Je n'en suppose aucun ; le taux et le régime sont des **paramètres** saisis, jamais écrits en dur. **[À REMPLIR : expert-comptable]** |
| 4 | **Exercice comptable** | Année civile (1er janvier au 31 décembre) ? **[À CONFIRMER]** |
| 5 | **Devises** | XOF de référence (confirmé). EUR/USD dès la phase 2, ou plus tard ? **[À décider]** |
| 6 | **Facturation** : numérotation (ex. `FA-2026-000001`), mentions légales obligatoires, conditions de paiement, logo, et **obligation éventuelle de facturation électronique normalisée en Côte d'Ivoire** | À faire valider avant la phase 4. **[À CONFIRMER : expert-comptable]** |
| 7 | **Paiements en ligne** : quels moyens (cartes, Mobile Money, virement) et quel prestataire ? | Nécessaire pour la phase 4 (les webhooks de paiement du contrat sont génériques). **[À décider]** |
| 8 | **Banques** : combien de comptes ? Format des relevés fournis (CSV, Excel, PDF) ? | Phase 3 par **import de fichier** ; connexion directe aux banques seulement si elles offrent une API (rare). |
| 9 | **Départements et activités** pour budgets et rentabilité (ex. vente en ligne, négoce, services) | **[À REMPLIR : vous]** ; je ne les invente pas. |
| 10 | **Salaires** | Enregistrés comme **charges** (montants), sans calcul de paie (cotisations sociales, impôts sur salaires : domaine légal à part). Un logiciel de paie dédié est préférable. **[À confirmer]** |
| 11 | **Double validation** : seuil en F CFA et qui valide | **[À REMPLIR : seuil et rôles]** |
| 12 | **Reprise de l'existant** : soldes d'ouverture au démarrage et historique à importer ? | Soldes d'ouverture saisis par écriture d'à-nouveaux validée par l'expert-comptable ; historique ancien non repris au début. **[À décider]** |
| 13 | **Durée de conservation** des pièces et écritures | Durée légale **[À CONFIRMER : expert-comptable]** ; le système rend les écritures non supprimables dans tous les cas. |
| 14 | **Qui accède à quoi** (dirigeant, comptable interne, expert externe) | Rôles proposés en § 8 ; à ajuster. |
| 15 | **Stockage des pièces justificatives et PDF** | Même stockage S3 que les images produit, **à fixer** (déjà en attente). |
| 16 | **Faut-il commencer la phase 1 maintenant**, ou d'abord finir le catalogue (import CSV, site branché) ? | Les deux sont indépendants ; je recommande de **finir le catalogue** (le site vide empêche de tester la vente), puis la phase 1 financière avec l'avis de l'expert-comptable. **[À décider : vous]** |

---

## 12. Ce que je ne ferai pas sans votre accord explicite

- Écrire un taux de TVA, un seuil, un plan comptable ou une règle fiscale « par défaut ».
- Remplir un écran avec des chiffres d'exemple.
- Permettre de modifier ou supprimer une écriture validée.
- Déclarer la conformité légale du système (c'est à votre expert-comptable de le faire).
