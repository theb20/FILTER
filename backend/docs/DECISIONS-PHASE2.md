# Décisions et état du contrat — phase 2 (livrable 1)

Statut : **contrat partiellement validé.** Aucun code de service n'est écrit.

## 1. Périmètre de validation

| Élément | Statut |
|---|---|
| Conventions communes (en-tête OpenAPI), `proto/common`, `proto/events` | **validé** (ajouts à revalider : voir [IDENTITY-GATEWAY-MAPPING.md](IDENTITY-GATEWAY-MAPPING.md) §5) |
| Gateway et service `identity` (59 routes) | **soumis à validation** : [IDENTITY-GATEWAY-MAPPING.md](IDENTITY-GATEWAY-MAPPING.md) |
| Statuts de commande et de paiement, remboursements | **validé** : [PROPOSITION-STATUTS.md](PROPOSITION-STATUTS.md) |
| Tous les autres services et routes (292 routes) | **brouillon** : relus route par route, avec la correspondance exacte route ↔ RPC, au début de leur phase |

Le statut est inscrit dans les fichiers : `x-status: draft | validated` sur chaque **tag** de `openapi.yaml` et sur les schémas validés ; `// STATUS: draft | validated` en tête de chaque `.proto`. `buf breaking` ne protège que les contrats validés (`proto/buf.yaml`).

## 2. Livrables

| Fichier | Contenu |
|---|---|
| [`../api/openapi.yaml`](../api/openapi.yaml) | OpenAPI 3.1 : 94 routes client + 257 routes d'administration, 297 schémas |
| [`../api/route-rpc-map.yaml`](../api/route-rpc-map.yaml) | Correspondance route ↔ RPC du périmètre validé |
| [`../proto/`](../proto) | 12 services + `common` + `events` : 360 RPC |
| [`../tools/`](../tools) | `check_contract.py`, `check_order_states.py`, `check_generated_go.sh` |
| [`../Makefile`](../Makefile), [`../../.github/workflows/contract.yml`](../../.github/workflows/contract.yml) | `make contract-check` en local et en CI |

Les fichiers ont été produits par un script jetable : ils font désormais foi, on les modifie directement.

## 3. Contrôles qualité (exécutés)

| Contrôle | Résultat |
|---|---|
| `buf lint` (règles STANDARD, **sans exception**) | passe |
| `buf breaking` | opérationnel ; limité aux contrats validés (testé : il détecte bien les ruptures) |
| OpenAPI 3.1 (`openapi-spec-validator`) | valide |
| `tools/check_contract.py` | opérationIds uniques, aucun schéma inutilisé, `x-status` sur chaque tag, permission sur chaque route d'administration, statuts identiques dans OpenAPI / proto / modèle d'états, route ↔ RPC du périmètre validé |
| `tools/check_order_states.py` | machine à états exhaustive, 0 erreur, test de mutation concluant |
| `tools/check_generated_go.sh` | les `.proto` génèrent 26 fichiers Go qui **compilent** (`go build`, `go vet`) |

Limite : le workflow GitHub `contract.yml` **n'a pas encore tourné** sur GitHub ; les mêmes commandes ont été exécutées en local. La comparaison `buf breaking` contre `main` n'a pas pu être testée de bout en bout (`main` ne contient pas encore de contrat).

## 4. Génération (règle)

Le code serveur et le client du frontend sont **générés** depuis le contrat, jamais écrits à la main en parallèle : Go depuis les `.proto` (`make generate`, plugins locaux : aucun envoi du contrat à un service externe) ; le gateway Go (types et squelette) depuis `openapi.yaml` ; le client TypeScript du frontend depuis `openapi.yaml` (phase 7). Seule la génération Go des `.proto` est déjà vérifiée ; les deux autres seront branchées avec le code correspondant.

## 5. Décisions validées

| # | Décision |
|---|---|
| A | Module Go : `github.com/theb20/FILTER/backend/<service>` ; code généré dans `backend/pkg/gen/<service>/v1` |
| B | Frontend dans `frontend/` (`1892fa3`) ; Docker Compose seul (pas de Kubernetes avant que la cible soit fixée) ; NATS JetStream, Meilisearch, `fulfillment` (13 services) ; mTLS plus tard |
| C | Paiement : agrégateur **GeniusPay**, premier et seul adaptateur derrière l'abstraction `payment`. Documentation fournie par toi avant la phase 4. Remboursements via le même adaptateur |
| D | Produits industriels **génériques** : arbre de catégories libre, attributs par catégorie, facettes dérivées, références croisées optionnelles, rien ne s'appelle « filter » |
| E | Un code promo par panier ; remises volume par produit **et** catégorie, la plus avantageuse retenue (pas d'addition) ; commande en rupture seulement pour les produits « sur commande » (erreur par ligne sinon) |
| F | Devis → commande par le client **et** par le commercial, tous deux audités |
| G | Comptes entreprise : validation manuelle (`pending`/`approved`/`rejected`), traité comme un particulier tant qu'en attente |
| H | Statut de commande (5) + statut de paiement (6) + `refundPending` ; remboursement partiel par ligne et par montant ; annulation d'une commande payée = remboursement automatique ; expiration automatique des prépaiements seulement |
| I | Pagination par **pages** ; **curseur** pour les listes qui grossissent sans fin : journal d'audit, mouvements de stock, notifications du prestataire, journal des messages |
| J | **Réglages par service**, jamais centralisés (voir §6) |
| K | TVA sur (HT + livraison) : **règle configurable** (`pricing-settings.taxBase`), à confirmer par ton comptable |

## 6. Réglages : chaque service possède les siens

Exposés au backoffice par `GET/PUT /admin/<service>-settings` ; chaque changement publie `settings.changed` (`service` + `section`).

| Service | Route | Contenu |
|---|---|---|
| `identity` | `/admin/identity-settings` | conservation : données personnelles, journal d'audit |
| `order` | `/admin/order-settings` | conservation : commandes, factures ; statuts annulables par le client ; annulation automatique des prépaiements ; moment d'émission de la facture |
| `payment` | `/admin/payment-settings` | moyens activés ; délais de paiement ; garde-fous du paiement à la livraison. Par entreprise : `/admin/companies/{id}/credit-terms` (plafond de crédit, délai de règlement) |
| `fulfillment` | `/admin/fulfillment-settings` | politique de réservation de stock par type de paiement ; seuil de stock faible |
| `quote` | `/admin/quote-settings` | validité des devis ; conservation (seuils d'accès : `/admin/quote-rules`) |
| `pricing` | `/admin/pricing-settings` | assiette de la TVA ; heure du rafraîchissement du taux USD |
| `support` | `/admin/support-settings` | conservation du support |
| `search` | `/admin/search-settings` | conservation du journal de recherche |
| `content` | `/admin/content-settings` | langues, modération des avis |

Toutes les valeurs sont **vides** tant que tu ne les as pas fournies (suggestions notées dans les descriptions, jamais appliquées).

## 7. Hypothèses (statut)

| # | Hypothèse | Statut |
|---|---|---|
| 1 | Un nouveau code promo remplace le précédent | validée |
| 2 | Annulation par le client : `placed` seulement | validée (réglable) |
| 3 | Transitions de statut | **refaite** : voir PROPOSITION-STATUTS.md, validée |
| 4 | Données légales modifiées → entreprise repasse en `pending` | validée |
| 5 | Avis modérés ; achat vérifié seulement si lié à une ligne de commande | validée |
| 6 | Réglages hébergés par `content` | **refusée** : réglages par service (§6) |
| 7 | Corps d'article = liste de sections | validée |
| 8 | Permissions : admin:read, admin:write, analytics:read, audit:read, catalog:read, catalog:write, companies:approve, content:publish, content:read, content:write, customers:privacy, customers:read, customers:write, inventory:read, inventory:write, invoices:read, notifications:read, notifications:write, orders:read, orders:refund, orders:write, payments:read, payments:reconcile, payments:settings, pricing:read, pricing:write, quotes:convert, quotes:read, quotes:write, reviews:moderate, reviews:read, shipping:read, shipping:write, support:read, support:write | validées ; rôles à confirmer plus tard |
| 9 | Journal d'audit hébergé par `identity` | validée |
| 10 | `captchaToken` facultatif, indépendant du fournisseur | validée |

## 8. À ma charge / à ta charge

- **Toi** : documentation GeniusPay (avant la phase 4) ; avis de ton comptable sur la TVA (assiette), les mentions et la numérotation des factures et avoirs, le moment d'émission de la facture.
- **Valeurs vides à fournir quand la phase concernée l'exigera** : durées de conservation, délais de paiement, validité des devis, zones et tarifs réels, seuil de livraison gratuite, entrepôts, fournisseur du taux USD, source de validation des N° TVA/SIRET, e-mail/SMS, hébergeur, fournisseur anti-spam, rôles, format du catalogue source.

- **Points ouverts hors périmètre** : litiges carte (contestation d'un paiement) à traiter après réception de la documentation GeniusPay ; cas limites (paiement tardif ou en double, expéditions partielles, source unique des transitions) : voir [PROPOSITION-CAS-LIMITES.md](PROPOSITION-CAS-LIMITES.md).
