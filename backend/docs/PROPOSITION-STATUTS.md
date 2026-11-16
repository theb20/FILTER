# Statut de commande et statut de paiement séparés

Statut : **VALIDÉ** (avec la correction demandée : état d'attente de remboursement). Les contrats (`openapi.yaml`, `.proto`) sont alignés ; ces règles sont **vérifiées par machine** (`tools/check_order_states.py`).

---

## 1. Modèle

`order.status` porte la **logistique** ; `order.paymentStatus` porte l'**argent** ; `order.refundPending` signale un remboursement qui attend une action.

| `order.status` (5) | `order.paymentStatus` (6) |
|---|---|
| `placed`, `preparing`, `shipped`, `delivered`, `cancelled` | `pending`, `paid`, `partially_refunded`, `refunded`, `failed`, `voided` |

| Ancien statut | Devient |
|---|---|
| `pending_payment` | `placed` + `pending` |
| `paid` | `paymentStatus = paid` |
| `refunded` | `paymentStatus = refunded` (ou `partially_refunded`) |
| `preparing`, `shipped`, `delivered`, `cancelled` | inchangés |

- `authorized` est **retiré** pour l'instant ; on l'ajoutera si la documentation GeniusPay confirme l'autorisation puis la capture.
- « En retard » est un **calcul** (`pending` et `dueAt` dépassé), pas un statut.
- `refundPending = true` tant qu'un remboursement est `pending` ou `failed`. **État d'attente explicite** : une commande `cancelled` dont le paiement est `paid` ou `partially_refunded` a **toujours** `refundPending = true` (remboursement créé automatiquement à l'annulation, en attente ou échoué). Un remboursement échoué lève une alerte au backoffice (`RefundFailed`, `GET /admin/refunds?status=failed`, compteur du tableau de bord).

## 2. Règles par type de paiement

COD et différé sont **désactivés au lancement** (réglages vides) mais modélisés.

| | `mobile_money` | `card` | `bank_transfer` | `cash_on_delivery` | `deferred_invoice` |
|---|---|---|---|---|---|
| Type | Prépaiement | Prépaiement | Prépaiement différé | À la réception | Crédit (entreprise validée) |
| Paiement à la création | `pending` | `pending` | `pending` (instructions + échéance) | `pending` | `pending` (échéance `dueAt`) |
| Passage à `paid` | webhook | webhook | webhook **ou** rapprochement manuel (comptabilité, super-admin, audité) | encaissement à la livraison (rapprochement) | règlement de la facture (rapprochement) |
| Échec | `failed`, nouvel essai possible | idem | `failed` à l'expiration | refus à la livraison : `shipped → cancelled` + `voided` | impayé : reste `pending`, signalé « en retard » |
| Libération `placed → preparing` | exige `paid` | exige `paid` | exige `paid` | immédiate (garde-fous : montant max, zones — réglages vides) | immédiate après contrôle du plafond de crédit (par entreprise, valeurs vides) |
| **Expiration automatique** | **oui** : `placed → cancelled`, `paymentStatus → voided`, stock libéré, client notifié | **oui** | **oui** (délai : réglage, suggestion 5 jours ouvrés) | **jamais** | **jamais** |
| Réservation de stock (réglage `fulfillment`) | durée fixe (suggestion 30 min) | durée fixe (suggestion 30 min) | égale au délai de virement | jusqu'à la livraison | jusqu'à la livraison |
| Annulation avant paiement | `cancelled` + `voided` | idem | idem | idem | idem (avoir si facture émise) |
| Annulation après paiement | `cancelled` + remboursement **obligatoire** (état d'attente `refundPending`) | idem | idem (virement manuel) | — (annulation impossible après encaissement : la commande est livrée) | idem |
| Canal de remboursement | moyen d'origine | carte d'origine | virement manuel | manuel | avoir ou virement |

### Cas du refus à la livraison (paiement à la livraison)

`shipped → cancelled` avec motif `delivery_refused` (obligatoire), `paymentStatus : pending → voided`, stock remis en vente. Aucun remboursement (rien n'a été encaissé). Vérifié par le script.

## 3. Transitions de `order.status`

| De → vers | Condition |
|---|---|
| `placed → preparing` | condition de libération du type de paiement |
| `placed → cancelled`, `preparing → cancelled` | client (selon réglage) ou personnel ; remboursement automatique si payée |
| `preparing → shipped` | un envoi existe |
| `shipped → delivered` | confirmation du transporteur ou du personnel |
| `shipped → cancelled` | **exception**, personnel seulement, **motif obligatoire** (colis perdu, refus à la livraison…) ; remboursement automatique si payée |
| `delivered`, `cancelled` | **terminaux** : un retour ou un défaut se traite par un remboursement avec motif, sans changer le statut |

Pas de statut `lost` ni de processus de retour (RMA) au lancement.

## 4. Remboursements

- **Annuler une commande payée crée automatiquement un remboursement total** de ce qui reste à rembourser.
- Un remboursement **partiel** (par **montant** ou par **ligne** + quantité, frais de livraison remboursables) est accepté à `placed`, `preparing`, `shipped`, `delivered`. Un remboursement **total** n'est accepté que si la commande est `cancelled` ou `delivered` : avant livraison, on **annule** (ce qui crée le remboursement).
- Motifs : `cancellation`, `lost_parcel`, `return`, `defect`, `goodwill`, `other`. Option `restock` par ligne.
- La somme des remboursements ne dépasse jamais le total encaissé. Chaque remboursement produit un **avoir** (numérotation et mentions : à valider par le comptable).
- Canaux manuels (virement, espèces, mobile money) : marqués effectués par la comptabilité (`POST /admin/refunds/{id}/complete`, audité).

## 5. Combinaisons autorisées (générées par le modèle)

| `order.status` | `mobile_money` | `card` | `bank_transfer` | `cash_on_delivery` | `deferred_invoice` |
|---|---|---|---|---|---|
| `placed` | `pending`, `paid`(⚑), `partially_refunded`(⚑), `failed` | `pending`, `paid`(⚑), `partially_refunded`(⚑), `failed` | `pending`, `paid`(⚑), `partially_refunded`(⚑), `failed` | `pending` | `pending`, `paid`(⚑), `partially_refunded`(⚑) |
| `preparing` | `paid`(⚑), `partially_refunded`(⚑) | `paid`(⚑), `partially_refunded`(⚑) | `paid`(⚑), `partially_refunded`(⚑) | `pending` | `pending`, `paid`(⚑), `partially_refunded`(⚑) |
| `shipped` | `paid`(⚑), `partially_refunded`(⚑) | `paid`(⚑), `partially_refunded`(⚑) | `paid`(⚑), `partially_refunded`(⚑) | `pending` | `pending`, `paid`(⚑), `partially_refunded`(⚑) |
| `delivered` | `paid`(⚑), `partially_refunded`(⚑), `refunded` | `paid`(⚑), `partially_refunded`(⚑), `refunded` | `paid`(⚑), `partially_refunded`(⚑), `refunded` | `pending`, `paid`(⚑), `partially_refunded`(⚑), `refunded` | `pending`, `paid`(⚑), `partially_refunded`(⚑), `refunded` |
| `cancelled` | `paid`⚑, `partially_refunded`⚑, `refunded`, `voided` | `paid`⚑, `partially_refunded`⚑, `refunded`, `voided` | `paid`⚑, `partially_refunded`⚑, `refunded`, `voided` | `voided` | `paid`⚑, `partially_refunded`⚑, `refunded`, `voided` |

⚑ = `refundPending = true` obligatoirement ; (⚑) = `refundPending` vrai ou faux.


Toute combinaison absente est **refusée** (`INVALID_STATE_TRANSITION`). Rappel : `refundPending` peut aussi être vrai sur un paiement `paid` ou `partially_refunded` d'une commande non annulée (remboursement partiel en cours ou échoué) ; il est **obligatoire** quand la commande est `cancelled`.

## 6. Vérification

`tools/check_order_states.py` explore **tous** les états atteignables depuis la création de la commande, pour chacun des 5 types, et contrôle :

- aucune commande annulée et payée sans `refundPending` ; `voided` seulement sur une commande annulée ; `refunded` seulement si annulée ou livrée ;
- `failed` seulement en prépaiement à l'état `placed` ; jamais de prépaiement préparé, expédié ou livré sans être payé ;
- l'expiration automatique existe pour les trois prépaiements et **jamais** pour la livraison ni le différé ;
- aucune impasse : tout état non final a une sortie ; le refus à la livraison (`shipped → cancelled` + `voided`) est atteignable ;
- les valeurs de `OpenAPI` et de `order.proto` sont identiques à celles du modèle (`check_contract.py`).

Résultat :

```
mobile_money       états atteignables=23  transitions= 63  erreurs=0
card               états atteignables=23  transitions= 63  erreurs=0
bank_transfer      états atteignables=23  transitions= 63  erreurs=0
cash_on_delivery   états atteignables=10  transitions= 17  erreurs=0
deferred_invoice   états atteignables=25  transitions= 64  erreurs=0
```

Test de mutation : retirer `refundPending` à l'annulation d'une commande payée, ou supprimer le refus à la livraison, fait échouer le script (5 et 1 erreurs).

## 7. Réglages liés (chacun dans son service)

| Réglage | Service | Valeur au lancement |
|---|---|---|
| Moyens de paiement activés | `payment` | COD et différé **absents** |
| Délais de paiement (mobile money, carte : minutes ; virement : jours ouvrés) | `payment` | vides (suggestions 30 min / 5 jours ouvrés) |
| Garde-fous du paiement à la livraison (montant max, zones) | `payment` | vides |
| Plafond de crédit et délai de règlement **par entreprise** | `payment` | vides |
| Politique de réservation de stock par type de paiement | `fulfillment` | mobile money/carte : durée fixe (vide) ; virement : égale au délai de paiement ; livraison/différé : jusqu'à la livraison |
| Statuts annulables par le client ; annulation automatique des prépaiements ; moment d'émission de la facture | `order` | `placed` ; activée ; **vide** (comptable) |

## 8. Points ouverts

- **Litiges carte** (contestation d'un paiement) : hors périmètre, à traiter après réception de la documentation GeniusPay.
- **Paiement tardif ou en double, expéditions partielles, source unique de la table** : proposition en attente de validation dans [PROPOSITION-CAS-LIMITES.md](PROPOSITION-CAS-LIMITES.md).
