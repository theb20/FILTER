# Proposition : paiement tardif ou en double, expéditions partielles, source unique des transitions

Statut : **PROPOSITION — le contrat (`openapi.yaml`, `.proto`) et les fichiers validés ne sont PAS modifiés.**
Prototype exécutable : [`tools/proposal/order-states.proposal.yaml`](../tools/proposal/order-states.proposal.yaml) et [`check_order_states_v2.py`](../tools/proposal/check_order_states_v2.py).

---

## 1. Paiement tardif ou en double

### Principe : l'argent est suivi au niveau du PAIEMENT, pas en changeant le statut de la commande

Un webhook « payé » qui ne peut pas être appliqué à la commande crée un **paiement non appliqué** (`unapplied`). Le statut de la commande **ne change pas** : `voided` reste terminal (une commande annulée et non payée reste « non payée »), et la matrice validée n'est pas touchée.

Un succès de paiement est **non appliqué** quand la commande ne peut pas l'absorber :
- **tardif** : la commande est déjà annulée (`cancelled` + `voided`, y compris après expiration automatique) ;
- **en double** : la commande est déjà payée (ou remboursée) et un **second** encaissement distinct arrive (nouvelle tentative, deux virements…) ;
- **inattendu** : paiement en ligne sur une commande à payer à la livraison.

Cas absorbé (donc pas non appliqué) : un paiement reçu **après un échec déclaré** (délai dépassé côté prestataire) alors que la commande est encore `placed` : `failed → paid`. Nouvelle transition ajoutée, sinon on rembourserait à tort un paiement valide.

Le **rejeu d'un même webhook** n'est jamais un doublon : il est écarté par l'identifiant d'événement du prestataire (`provider_events`, unique).

### Cycle d'un paiement non appliqué

`reçu` → remboursement **automatique** créé (même canal que le paiement) → `réussi` (fin) ou `échoué` → **alerte** au backoffice, nouvelles tentatives, puis traitement manuel (`orders:refund`, `payments:reconcile`).

- Le montant remboursé est **exactement** celui du paiement non appliqué.
- `order.totalPaid` ne compte **que** les paiements appliqués : `totalRefunded`, `paymentStatus` et les remboursements « normaux » restent cohérents.
- `refundPending` (API) = remboursement de la commande en cours **ou** paiement non appliqué en attente. Le backoffice le voit donc dans la même file d'alertes.
- Un montant différent du total (sous- ou sur-paiement) est traité comme **non appliqué en entier** puis remboursé : jamais d'encaissement partiel appliqué en silence.

### Modèle d'états (prototype)

État = (`status`, `paymentStatus`, `rf`, `held`) : `rf` = remboursement de la commande en cours ; `held` = argent non appliqué en attente de remboursement. 24 règles dans le fichier YAML (6 nouvelles).

| Règle | Effet |
|---|---|
| paiement reçu après un échec déclaré (`placed`) | `failed → paid` |
| paiement non appliqué reçu (tardif, en double, inattendu) | `held = vrai`, statuts inchangés |
| remboursement automatique réussi | `held = faux` |
| remboursement automatique échoué | `held` reste vrai, **alerte** |

Résultat de l'exploration exhaustive (`check_order_states_v2.py`) :

```
mobile_money       états= 44  transitions=204  erreurs=0
card               états= 44  transitions=204  erreurs=0
bank_transfer      états= 44  transitions=204  erreurs=0
cash_on_delivery   états= 20  transitions= 74  erreurs=0
deferred_invoice   états= 46  transitions=202  erreurs=0
Comportement validé inchangé : mêmes états atteignables hors paiement tardif/en double.
```

| `order.status` | `mobile_money` | `card` | `bank_transfer` | `cash_on_delivery` | `deferred_invoice` |
|---|---|---|---|---|---|
| `placed` | `pending`, `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ, `failed` | `pending`, `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ, `failed` | `pending`, `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ, `failed` | `pending`ᴴ | `pending`, `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ |
| `preparing` | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ | `pending`ᴴ | `pending`, `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ |
| `shipped` | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ | `pending`ᴴ | `pending`, `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ |
| `delivered` | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ, `refunded`ᴴ | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ, `refunded`ᴴ | `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ, `refunded`ᴴ | `pending`ᴴ, `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ, `refunded`ᴴ | `pending`, `paid`(⚑)ᴴ, `partially_refunded`(⚑)ᴴ, `refunded`ᴴ |
| `cancelled` | `paid`⚑ᴴ, `partially_refunded`⚑ᴴ, `refunded`ᴴ, `voided`ᴴ | `paid`⚑ᴴ, `partially_refunded`⚑ᴴ, `refunded`ᴴ, `voided`ᴴ | `paid`⚑ᴴ, `partially_refunded`⚑ᴴ, `refunded`ᴴ, `voided`ᴴ | `voided`ᴴ | `paid`⚑ᴴ, `partially_refunded`⚑ᴴ, `refunded`ᴴ, `voided`ᴴ |

⚑ remboursement de la commande en cours/échoué ; ᴴ paiement tardif ou en double possible (tracé et remboursé automatiquement).


Contrôles ajoutés au script :
- `voided` reste terminal : aucune transition ne sort de `voided` vers un autre statut de paiement ;
- scénarios atteignables : paiement tardif après expiration → (`cancelled`, `voided`) avec `held` → remboursement automatique → retour à l'état terminal ; paiement en double pendant la préparation ; paiement reçu après échec ;
- aucune impasse (un état avec `held` a toujours une sortie) ; déterminisme ; **chaque règle est utilisée** ;
- le comportement validé est **inchangé** : sans paiement non appliqué, mêmes états atteignables que `check_order_states.py` actuel.

Test de mutation : supprimer le remboursement automatique, faire rouvrir `voided` par un paiement tardif, ignorer le paiement tardif, ou ne pas absorber le paiement après échec **fait échouer** le script (4 cas sur 4).

### Changements de contrat proposés (non appliqués)

| Où | Changement |
|---|---|
| OpenAPI `Order` | `refundPending` redéfini (ci-dessus) ; admin : `unappliedPayments[]` (`paymentId`, `amount`, `reason` : `late` \| `duplicate` \| `unexpected`, `receivedAt`, `refundStatus`) |
| OpenAPI paiements admin | `application` (`applied` \| `unapplied`) et `unappliedReason` sur `PaymentAdmin` ; filtre `GET /admin/payments?application=unapplied` |
| `RefundReason` | ajout de `late_payment`, `duplicate_payment`, `unexpected_payment` ; `Refund.kind` : `order` \| `unapplied` |
| Tableau de bord | compteur `unappliedPayments` (en plus de `pendingRefunds`) |
| `payment-settings` | politique de relance du remboursement automatique : nombre d'essais et délai (**valeurs vides**) |
| `payment.proto` | statut d'application sur le paiement ; RPC de relance ; |
| `events.proto` | `PaymentUnapplied` (`payment_id`, `order_id`, `amount`, `reason`) ; `RefundRequested.automatic` existe déjà |

### Questions

1. Un paiement **tardif** sur commande annulée est **toujours remboursé** (ma proposition, conforme à ta demande). On ne « ressuscite » jamais la commande, même si le stock est encore disponible : correct ?
2. Sous- ou sur-paiement : **non appliqué en entier puis remboursé** (ma proposition) plutôt qu'un encaissement partiel : correct ?
3. Politique de relance du remboursement automatique : nombre d'essais et délai **[À REMPLIR]**. Selon GeniusPay, certains moyens (mobile money) pourraient ne pas être remboursables automatiquement : dans ce cas, alerte immédiate et traitement manuel.

---

## 2. Expéditions partielles : règle de lancement

**Règle (à écrire dans le contrat) :** une commande = **une seule expédition**, créée quand **toutes les lignes sont prêtes** ; le **délai annoncé** au client est celui de la **ligne la plus longue**.

- Une ligne en stock est prête dès que son stock est confirmé ; une ligne `made_to_order` l'est quand le personnel la déclare prête.
- Délai d'une ligne : en stock → `shipsWithinDays.max` ; `made_to_order` → `leadTimeDays`. Délai de la commande = **maximum** des lignes. Même calcul pour le panier, l'estimation de livraison et la commande.
- Le statut logistique ne change pas : `preparing` couvre l'attente des lignes `made_to_order`. Garde de `preparing → shipped` : toutes les lignes prêtes.

Changements de contrat proposés (non appliqués) :
- `OrderLine.readiness` : `ready` \| `awaiting_stock` \| `awaiting_production` ; `Order.estimatedShipping` (`from`, `to`) ;
- route `POST /admin/orders/{id}/lines/{productId}/mark-ready` (`orders:write`) ;
- création d'expédition refusée (`SHIPMENT_INCOMPLETE`, 422) tant qu'une ligne n'est pas prête.

**Modèle ouvert à plusieurs expéditions** : l'expédition porte déjà `orderId` (relation 1-n) ; on ajoute `Shipment.lines[]` (table `shipment_lines` : ligne, quantité) et le réglage `fulfillment-settings.partialShipmentsEnabled`, **verrouillé à faux** au lancement (une expédition doit couvrir toutes les lignes). Activer plusieurs expéditions plus tard ne changera ni le schéma ni les routes ; il restera à définir le statut `partially_shipped`, exprimé alors dans le fichier des transitions (§3).

Question : le délai annoncé est-il une date d'**expédition** (ma proposition) ou de **livraison** (expédition + transport de la zone) ?

---

## 3. Source unique de la table de transitions

Un fichier déclaratif unique, `backend/api/order-states.yaml` (le prototype est `tools/proposal/order-states.proposal.yaml`) : états, types de paiement, règles (`from` → `to`).

| Consommateur | Lecture |
|---|---|
| `tools/check_order_states.py` | interprète le fichier : exploration exhaustive, invariants, scénarios |
| service `order` (Go) | **code généré** par `tools/genfsm` (programme Go) → `services/order/internal/domain/fsm_gen.go`, non commité, régénéré par `make generate` ; le service n'écrit aucune table à la main |
| `check_contract.py` | compare les énumérations `OrderStatus` / `OrderPaymentStatus` d'OpenAPI et de `order.proto` avec le fichier |

Garde-fous contre toute divergence :
- un test Go relit le **même fichier** et refait l'exploration exhaustive : il doit retrouver les **mêmes nombres d'états et de transitions** que le script Python ;
- la CI échoue si le code généré est périmé (`make generate` puis `git diff`) ;
- les descriptions des routes de transition renvoient au fichier plutôt que de recopier la table.

---

## 4. Points ouverts

- **Litiges carte (contestation d'un paiement, « chargeback ») : hors périmètre** pour l'instant ; à traiter quand la documentation GeniusPay aura été fournie. Impacts à prévoir alors : statut de paiement ou indicateur de litige, délai de réponse, blocage de l'expédition, preuves à fournir.
- Documentation GeniusPay (remboursements automatiques, autorisation puis capture).
- Valeurs vides listées au §8 de `DECISIONS-PHASE2.md`.
