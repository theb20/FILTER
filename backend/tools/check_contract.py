#!/usr/bin/env python3
"""Contrôles de cohérence du contrat (à lancer en CI). Dépendances : pyyaml, openapi-spec-validator.

1. OpenAPI 3.1 valide ; operationId uniques ; aucun schéma inutilisé.
2. Chaque tag porte `x-status: draft | validated`.
3. Chaque route d'administration (hors authentification) déclare `x-required-permission`.
4. Statuts de commande / paiement identiques dans OpenAPI, dans order.proto et dans le modèle d'états.
5. Périmètre validé : chaque route des tags validés figure dans api/route-rpc-map.yaml, et chaque RPC cité existe.
"""
import json, re, sys, collections
from pathlib import Path
import yaml
from openapi_spec_validator import validate
from openapi_spec_validator.readers import read_from_filename

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(Path(__file__).parent))
import check_order_states as sm

errors = []
def err(msg): errors.append(msg)

spec, _ = read_from_filename(str(ROOT / "api/openapi.yaml"))
try: validate(spec)
except Exception as e: err(f"OpenAPI invalide : {str(e)[:300]}")

ops = [(m, p, o) for p, v in spec["paths"].items() for m, o in v.items() if m in ("get", "post", "put", "patch", "delete")]
dup = [k for k, c in collections.Counter(o["operationId"] for _, _, o in ops).items() if c > 1]
if dup: err(f"operationId dupliqués : {dup}")
txt = json.dumps(spec)
unused = [n for n in spec["components"]["schemas"] if f'"#/components/schemas/{n}"' not in txt]
if unused: err(f"schémas inutilisés : {unused}")

tag_status = {t["name"]: t.get("x-status") for t in spec["tags"]}
for name, st in tag_status.items():
    if st not in ("draft", "validated"): err(f"tag « {name} » sans x-status valide ({st!r})")
for m, p, o in ops:
    if o["tags"][0] not in tag_status: err(f"{m.upper()} {p} : tag inconnu {o['tags'][0]}")

for m, p, o in ops:
    if p.startswith("/admin") and not p.startswith(("/admin/auth", "/admin/me")) and "x-required-permission" not in o:  # auth et /admin/me* : libre-service du personnel
        err(f"{m.upper()} {p} : x-required-permission manquant")

# G10 : le gateway refuse les champs inconnus -> AUCUN corps de requête ou de réponse « objet » inline
def _inline(sc): return "$ref" not in sc and sc.get("type") == "object"
for m, p, o in ops:
    for ct, mt in o.get("requestBody", {}).get("content", {}).items():
        if _inline(mt["schema"]): err(f"{m.upper()} {p} : corps de requête inline (schéma nommé requis)")
    for code, r in o["responses"].items():
        if code.startswith("2"):
            for ct, mt in r.get("content", {}).items():
                if _inline(mt["schema"]): err(f"{m.upper()} {p} : réponse {code} inline (schéma nommé requis)")
# les schémas de corps de requête refusent les champs inconnus (sauf corps volontairement opaques)
OPEN_OK = {"PaymentWebhookPayload"}
for m, p, o in ops:
    for ct, mt in o.get("requestBody", {}).get("content", {}).items():
        ref = mt["schema"].get("$ref", "").split("/")[-1]
        if ref and ref not in OPEN_OK and ref in spec["components"]["schemas"]:
            sc = spec["components"]["schemas"][ref]
            if sc.get("type") == "object" and sc.get("additionalProperties") is not False: err(f"{ref} : additionalProperties doit être false")
# sécurité des jetons (périmètre validé)
S_ = spec["components"]["schemas"]
if "refreshToken" in S_["AuthTokens"]["properties"]: err("AuthTokens ne doit PAS contenir de jeton de rafraîchissement (cookie HttpOnly uniquement)")
for pth in ("/auth/login", "/auth/refresh", "/auth/oauth/exchange"):
    op_ = spec["paths"][pth]["post"]
    if "Set-Cookie" not in next(r for c, r in op_["responses"].items() if c.startswith("2")).get("headers", {}): err(f"{pth} : cookie de rafraîchissement non documenté (Set-Cookie)")
for pth in ("/auth/refresh", "/auth/logout"):
    if not any(x.get("$ref", "").endswith("/CsrfHeader") for x in spec["paths"][pth]["post"].get("parameters", [])): err(f"{pth} : protection CSRF (X-Requested-With) absente")
for pth in ("/admin/auth/2fa/verify", "/admin/auth/2fa/enroll", "/admin/auth/2fa/enroll/confirm"):
    if spec["paths"][pth]["post"]["security"] != [{"staffChallenge": []}]: err(f"{pth} : doit exiger le jeton temporaire `staffChallenge`")
if "ACCOUNT_LOCKED" in S_["Problem"]["properties"]["code"]["enum"]: err("ACCOUNT_LOCKED : verrouillage dur de compte interdit (ralentissement progressif seulement)")
if "409" in spec["paths"]["/auth/register"]["post"]["responses"]: err("/auth/register : 409 révèle l'existence d'une adresse (réponse uniforme exigée)")
if any(r.get("rpc") == "identity.AppendAudit" for r in yaml.safe_load((ROOT / "api/route-rpc-map.yaml").read_text())["routes"]): err("AppendAudit : l'audit n'a qu'UNE voie d'écriture (événements audit.*)")
if "DELETE" in {m.upper() for m in spec["paths"].get("/admin/staff-users/{id}", {})}: err("DELETE /admin/staff-users/{id} : le personnel se désactive, il ne se supprime pas")

# statuts : OpenAPI = modèle d'états = proto
sch = spec["components"]["schemas"]
if sch["OrderStatus"]["enum"] != sm.STATUSES: err(f"OrderStatus OpenAPI {sch['OrderStatus']['enum']} ≠ modèle {sm.STATUSES}")
if sch["OrderPaymentStatus"]["enum"] != sm.PAYMENTS: err(f"OrderPaymentStatus OpenAPI ≠ modèle {sm.PAYMENTS}")
proto = (ROOT / "proto/order/v1/order.proto").read_text()
def proto_enum(name):
    body = re.search(rf"enum {name} \{{(.*?)\}}", proto, re.S).group(1)
    return [v.lower() for v in re.findall(rf"{re.sub(r'(?<!^)(?=[A-Z])', '_', name).upper()}_([A-Z_]+) =", body) if v != "UNSPECIFIED"]
if proto_enum("OrderStatus") != sm.STATUSES: err(f"OrderStatus proto {proto_enum('OrderStatus')} ≠ modèle")
if proto_enum("PaymentStatus") != sm.PAYMENTS: err(f"PaymentStatus proto {proto_enum('PaymentStatus')} ≠ modèle")

# périmètre validé : route ↔ RPC
rmap = yaml.safe_load((ROOT / "api/route-rpc-map.yaml").read_text())["routes"]
mapped = {(r["method"].lower(), r["path"]) for r in rmap}
validated = {(m, p) for m, p, o in ops if tag_status.get(o["tags"][0]) == "validated"}
for m, p in sorted(validated - mapped): err(f"route validée sans RPC : {m.upper()} {p}")
for m, p in sorted(mapped - {(m, p) for m, p, _ in ops}): err(f"route du mapping absente de l'OpenAPI : {m.upper()} {p}")
for m, p in sorted(mapped - validated): err(f"route mappée mais non validée (tag draft) : {m.upper()} {p}")
services = {}
for f in (ROOT / "proto").glob("*/v1/*.proto"):
    services[f.parent.parent.name] = set(re.findall(r"^\s+rpc (\w+)\(", f.read_text(), re.M))
for r in rmap:
    if r["rpc"] == "gateway": continue
    svc, rpc = r["rpc"].split(".")
    if rpc not in services.get(svc, set()): err(f"RPC inexistant : {r['rpc']} (route {r['method']} {r['path']})")

if "AppendAudit" in services.get("identity", set()): err("identity.AppendAudit : doit être supprimé (une seule voie d'écriture : événements audit.*)")
print(f"OpenAPI : {len(ops)} opérations, {len(sch)} schémas ; tags validés : {sorted(t for t, s in tag_status.items() if s == 'validated')}")
print(f"Mapping validé : {len(rmap)} routes ↔ RPC ; RPC identity : {len(services.get('identity', []))}")
if errors:
    print("\n".join(f"✗ {e}" for e in errors)); sys.exit(1)
print("Contrat COHÉRENT.")
