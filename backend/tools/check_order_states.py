#!/usr/bin/env python3
"""Vérifie la cohérence de la machine à états commande × paiement (stdlib uniquement).

Modèle : un état = (status, payment_status, refund_pending). Pour chaque type de paiement, on explore TOUS les
états atteignables depuis la création de la commande, puis on vérifie les invariants. Le script échoue (code 1)
si une règle est violée ; `--table` imprime les tableaux Markdown de la documentation.

Règles métier VALIDÉES (voir docs/PROPOSITION-STATUTS.md) :
  - order.status : placed → preparing → shipped → delivered, ou cancelled ; rien ne sort de delivered/cancelled.
  - payment_status : pending, paid, partially_refunded, refunded, failed, voided  (pas de `authorized` pour l'instant).
  - annuler une commande payée crée un remboursement total : état d'attente explicite (cancelled, paid, refund_pending=True).
  - prépaiements (mobile_money, card, bank_transfer) : préparation seulement une fois `paid` ; expiration automatique.
  - cash_on_delivery, deferred_invoice : jamais d'expiration automatique (désactivés au lancement, mais modélisés).
"""
import sys
from collections import deque

KINDS = ["mobile_money", "card", "bank_transfer", "cash_on_delivery", "deferred_invoice"]
PREPAID = {"mobile_money", "card", "bank_transfer"}
STATUSES = ["placed", "preparing", "shipped", "delivered", "cancelled"]
PAYMENTS = ["pending", "paid", "partially_refunded", "refunded", "failed", "voided"]

def transitions(kind, st):
    """Renvoie [(libellé, nouvel_état)] depuis l'état `st` pour ce type de paiement."""
    status, pay, rp = st
    out = []
    # --- paiement
    if kind in PREPAID and status == "placed":
        if pay == "pending" and not rp: out += [("payer", (status, "paid", False)), ("échec paiement", (status, "failed", False))]
        if pay == "failed": out.append(("nouvelle tentative", (status, "pending", False)))
    if kind == "cash_on_delivery" and status == "delivered" and pay == "pending":
        out.append(("encaissement à la livraison", (status, "paid", False)))
    if kind == "deferred_invoice" and pay == "pending" and status in ("placed", "preparing", "shipped", "delivered"):
        out.append(("règlement de la facture", (status, "paid", False)))
    # --- expiration automatique (prépaiements uniquement)
    if kind in PREPAID and status == "placed" and pay in ("pending", "failed"):
        out.append(("expiration automatique", ("cancelled", "voided", False)))
    # --- libération pour préparation
    if status == "placed":
        ok = (pay in ("paid", "partially_refunded")) if kind in PREPAID else (pay in ("pending", "paid"))
        if ok: out.append(("démarrer la préparation", ("preparing", pay, rp)))
    if status == "preparing": out.append(("expédier", ("shipped", pay, rp)))
    if status == "shipped": out.append(("livrer", ("delivered", pay, rp)))
    # --- annulation (placed, preparing, shipped)
    if status in ("placed", "preparing", "shipped"):
        if pay in ("pending", "failed"):
            out.append(("annuler (sans encaissement)", ("cancelled", "voided", False)))
        elif pay in ("paid", "partially_refunded"):
            out.append(("annuler (remboursement automatique créé)", ("cancelled", pay, True)))
    # --- remboursements
    if pay in ("paid", "partially_refunded"):
        if not rp and status != "cancelled":
            out.append(("demander remboursement partiel", (status, pay, True)))
        if not rp and status == "delivered":
            out.append(("demander remboursement total", (status, pay, True)))
        if rp:
            out.append(("remboursement échoué (alerte)", (status, pay, True)))
            # succès : total si annulée/livrée, partiel sinon (partiel seulement avant annulation)
            if status in ("cancelled", "delivered"): out.append(("remboursement total réussi", (status, "refunded", False)))
            if status != "cancelled": out.append(("remboursement partiel réussi", (status, "partially_refunded", False)))
    return out

def explore(kind):
    start = ("placed", "pending", False)
    seen, edges, q = {start}, [], deque([start])
    while q:
        cur = q.popleft()
        for label, nxt in transitions(kind, cur):
            edges.append((cur, label, nxt))
            if nxt not in seen: seen.add(nxt); q.append(nxt)
    return seen, edges

def final(st):
    status, pay, rp = st
    if rp: return False
    return (status == "cancelled" and pay in ("voided", "refunded")) or (status == "delivered" and pay in ("paid", "partially_refunded", "refunded"))

def check(kind):
    errors = []
    seen, edges = explore(kind)
    out_deg = {s: 0 for s in seen}
    for a, _, _ in edges: out_deg[a] += 1
    for st in sorted(seen):
        status, pay, rp = st
        if status == "cancelled" and pay in ("paid", "partially_refunded") and not rp: errors.append(f"{st}: commande annulée et payée SANS remboursement en attente")
        if pay == "voided" and status != "cancelled": errors.append(f"{st}: `voided` hors commande annulée")
        if pay == "refunded" and status not in ("cancelled", "delivered"): errors.append(f"{st}: remboursement total avant livraison sans annulation")
        if pay == "failed" and (status != "placed" or kind not in PREPAID): errors.append(f"{st}: `failed` hors prépaiement à l'état placed")
        if rp and pay not in ("paid", "partially_refunded"): errors.append(f"{st}: refund_pending sans montant encaissé")
        if pay == "pending" and rp: errors.append(f"{st}: pending avec remboursement")
        if kind in PREPAID and status in ("preparing", "shipped", "delivered") and pay in ("pending", "failed", "voided"): errors.append(f"{st}: prépaiement préparé/expédié sans être payé")
        if out_deg[st] == 0 and not final(st): errors.append(f"{st}: impasse (aucune sortie et pas un état final)")
    labels = {l for _, l, _ in edges}
    if kind not in PREPAID and "expiration automatique" in labels: errors.append("expiration automatique appliquée à un paiement non prépayé")
    if kind in PREPAID and "expiration automatique" not in labels: errors.append("expiration automatique absente pour un prépaiement")
    if ("cancelled", "paid", True) not in seen and kind != "cash_on_delivery": errors.append("état d'attente (cancelled, paid, refundPending) inatteignable")
    if kind == "cash_on_delivery":
        if (("shipped", "pending", False), "annuler (sans encaissement)", ("cancelled", "voided", False)) not in edges: errors.append("refus à la livraison : shipped → cancelled + voided inatteignable")
        if ("delivered", "paid", False) not in seen: errors.append("COD : encaissement inatteignable")
    # tout état final est atteignable ; toute commande se termine
    if not any(final(s) for s in seen): errors.append("aucun état final atteignable")
    return seen, edges, errors

def table():
    print("| `order.status` | " + " | ".join(f"`{k}`" for k in KINDS) + " |")
    print("|---|" + "---|" * len(KINDS))
    reach = {k: explore(k)[0] for k in KINDS}
    for status in STATUSES:
        row = []
        for k in KINDS:
            pays = []
            for pay in PAYMENTS:
                flags = sorted({rp for (s, p, rp) in reach[k] if s == status and p == pay})
                if not flags: continue
                pays.append(f"`{pay}`" + ("⚑" if flags == [True] else ("(⚑)" if True in flags else "")))
            row.append(", ".join(pays) or "—")
        print(f"| `{status}` | " + " | ".join(row) + " |")
    print("\n⚑ = `refundPending = true` obligatoirement ; (⚑) = `refundPending` vrai ou faux.")

def main():
    bad = False
    for k in KINDS:
        seen, edges, errors = check(k)
        print(f"{k:18s} états atteignables={len(seen):2d}  transitions={len(edges):3d}  erreurs={len(errors)}")
        for e in errors: print("   ✗", e); bad = True
    if "--table" in sys.argv: print(); table()
    sys.exit(1 if bad else 0)

if __name__ == "__main__":
    main()
