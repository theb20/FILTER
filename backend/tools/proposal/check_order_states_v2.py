#!/usr/bin/env python3
"""PROTOTYPE de la version 2 de check_order_states.py : lit order-states.proposal.yaml (source unique).

Explore tous les états (status, payment, rf, held) de chaque type de paiement, vérifie les invariants et les scénarios
« paiement tardif / en double », et prouve que le comportement de la version validée n'est pas modifié.
"""
import sys
from collections import deque
from pathlib import Path
import yaml

HERE = Path(__file__).resolve().parent
SPEC = yaml.safe_load(open(sys.argv[1] if len(sys.argv) > 1 and not sys.argv[1].startswith("--") else HERE / "order-states.proposal.yaml"))
STATUSES, PAYMENTS, KINDS = SPEC["statuses"], SPEC["payment_statuses"], SPEC["kinds"]
GROUPS = SPEC["kind_groups"]
PREPAID = set(GROUPS["prepaid"])
I = SPEC["initial"]
START = (I["status"], I["payment"], I["rf"], I["held"])

def kinds_of(t):
    k = t["kinds"]
    return set(KINDS) if k == "all" else set(GROUPS[k]) if isinstance(k, str) else set(k)

def matches(frm, st):
    status, pay, rf, held = st
    return ((("status" not in frm) or status in frm["status"]) and (("payment" not in frm) or pay in frm["payment"])
            and (("rf" not in frm) or rf == frm["rf"]) and (("held" not in frm) or held == frm["held"]))

def apply(to, st):
    status, pay, rf, held = st
    return (to.get("status", status), to.get("payment", pay), to.get("rf", rf), to.get("held", held))

def moves(kind, st):
    return [(t["id"], t["label"], apply(t["to"], st)) for t in SPEC["transitions"] if kind in kinds_of(t) and matches(t["from"], st)]

def explore(kind):
    seen, edges, q = {START}, [], deque([START])
    while q:
        cur = q.popleft()
        for tid, label, nxt in moves(kind, cur):
            edges.append((cur, tid, nxt))
            if nxt not in seen: seen.add(nxt); q.append(nxt)
    return seen, edges

def final(st):
    status, pay, rf, held = st
    if rf or held: return False
    return (status == "cancelled" and pay in ("voided", "refunded")) or (status == "delivered" and pay in ("paid", "partially_refunded", "refunded"))

def check(kind):
    errs, (seen, edges) = [], explore(kind)
    outdeg = {s: 0 for s in seen}
    for a, _, _ in edges: outdeg[a] += 1
    for st in sorted(seen):
        status, pay, rf, held = st
        if status == "cancelled" and pay in ("paid", "partially_refunded") and not rf: errs.append(f"{st}: annulée et payée SANS remboursement en attente")
        if pay == "voided" and status != "cancelled": errs.append(f"{st}: voided hors commande annulée")
        if pay == "refunded" and status not in ("cancelled", "delivered"): errs.append(f"{st}: remboursement total avant livraison sans annulation")
        if pay == "failed" and (status != "placed" or kind not in PREPAID): errs.append(f"{st}: failed hors prépaiement placed")
        if rf and pay not in ("paid", "partially_refunded"): errs.append(f"{st}: remboursement de commande sans montant encaissé")
        if pay == "pending" and rf: errs.append(f"{st}: pending avec remboursement")
        if kind in PREPAID and status in ("preparing", "shipped", "delivered") and pay in ("pending", "failed", "voided"): errs.append(f"{st}: prépaiement préparé sans être payé")
        if outdeg[st] == 0 and not final(st): errs.append(f"{st}: impasse")
    # voided est TERMINAL pour le statut de paiement : l'argent tardif ne le modifie jamais
    for a, tid, b in edges:
        if a[1] == "voided" and b[1] != "voided": errs.append(f"voided n'est plus terminal : {a} --{tid}--> {b}")
    labels = {tid for _, tid, _ in edges}
    if kind not in PREPAID and "expire" in labels: errs.append("expiration automatique hors prépaiement")
    if kind in PREPAID and "expire" not in labels: errs.append("expiration automatique absente")
    # déterminisme : un même événement ne doit pas donner deux résultats depuis le même état
    by = {}
    for a, tid, b in edges: by.setdefault((a, tid), set()).add(b)
    for (a, tid), r in by.items():
        if len(r) > 1: errs.append(f"non déterministe : {a} --{tid}--> {r}")
    # scénarios
    def reach(s): return s in seen
    def edge(a, tid, b): return (a, tid, b) in set(edges)
    if kind in PREPAID:
        late = ("cancelled", "voided", False, True)
        if not reach(late): errs.append("scénario : paiement tardif après expiration/annulation inatteignable")
        if not edge(late, "held_refund_ok", ("cancelled", "voided", False, False)): errs.append("scénario : le remboursement automatique ne ramène pas à (annulée, voided) terminal")
        if not reach(("preparing", "paid", False, True)): errs.append("scénario : paiement en double inatteignable")
        if not edge(("placed", "failed", False, False), "pay_after_failure", ("placed", "paid", False, False)): errs.append("scénario : paiement reçu après échec déclaré non absorbé")
    if kind == "cash_on_delivery" and not edge(("shipped", "pending", False, False), "cancel_unpaid", ("cancelled", "voided", False, False)): errs.append("refus à la livraison inatteignable")
    if not any(final(s) for s in seen): errs.append("aucun état final")
    return seen, edges, errs

def unused_rows():
    used = set()
    for k in KINDS: used |= {tid for _, tid, _ in explore(k)[1]}
    return [t["id"] for t in SPEC["transitions"] if t["id"] not in used]

def v1_projection_unchanged():
    """Le comportement VALIDÉ ne change pas : sans les cas nouveaux, mêmes états atteignables que check_order_states.py."""
    sys.path.insert(0, str(HERE.parent))
    import check_order_states as v1
    bad = []
    for k in KINDS:
        v1_states = {(s, p, rp) for (s, p, rp) in v1.explore(k)[0]}
        v2_proj = {(s, p, rf) for (s, p, rf, h) in explore(k)[0]}
        v2_no_held = {(s, p, rf) for (s, p, rf, h) in explore(k)[0] if not h}
        if v1_states != v2_no_held: bad.append(f"{k}: états sans paiement non absorbé différents de la version validée ({sorted(v1_states ^ v2_no_held)[:3]})")
    return bad

def table():
    print("| `order.status` | " + " | ".join(f"`{k}`" for k in KINDS) + " |"); print("|---|" + "---|" * len(KINDS))
    reach = {k: explore(k)[0] for k in KINDS}
    for status in STATUSES:
        cells = []
        for k in KINDS:
            parts = []
            for pay in PAYMENTS:
                fl = {(rf, h) for (s, p, rf, h) in reach[k] if s == status and p == pay}
                if not fl: continue
                tag = ""
                if any(rf for rf, _ in fl): tag += "⚑" if all(rf for rf, _ in fl) else "(⚑)"
                if any(h for _, h in fl): tag += "ᴴ"
                parts.append(f"`{pay}`{tag}")
            cells.append(", ".join(parts) or "—")
        print(f"| `{status}` | " + " | ".join(cells) + " |")
    print("\n⚑ remboursement de la commande en cours/échoué ; ᴴ paiement tardif ou en double possible (tracé et remboursé automatiquement).")

def main():
    bad = False
    for k in KINDS:
        seen, edges, errs = check(k)
        print(f"{k:18s} états={len(seen):3d}  transitions={len(edges):3d}  erreurs={len(errs)}")
        for e in errs: print("   ✗", e); bad = True
    dead = unused_rows()
    if dead: print("   ✗ lignes de transition jamais utilisées :", dead); bad = True
    chg = v1_projection_unchanged()
    for c in chg: print("   ✗", c); bad = True
    if not chg: print("Comportement validé inchangé : mêmes états atteignables hors paiement tardif/en double.")
    if "--table" in sys.argv: print(); table()
    sys.exit(1 if bad else 0)

if __name__ == "__main__": main()
