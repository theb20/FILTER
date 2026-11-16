#!/usr/bin/env python3
"""Génère la table « opération → permission requise » du gateway depuis openapi.yaml.

Sortie : services/gateway/internal/api/permissions_gen.go (code GÉNÉRÉ : jamais commité, produit par `make generate`).
Toute opération protégée par `staffAuth` figure dans la table : permission = valeur de `x-required-permission`,
ou "" pour les routes en libre-service du personnel (/admin/me*, /admin/auth/*). Le gateway REFUSE (fail closed)
toute opération du personnel absente de la table.
"""
import re, sys, pathlib

root = pathlib.Path(__file__).resolve().parent.parent
text = (root / "api/openapi.yaml").read_text()
ops = {}
cur_op = cur_perm = None
cur_staff = False
def flush():
    if cur_op and cur_staff:
        ops[cur_op] = cur_perm or ""
for line in text.split("\n"):
    m = re.match(r"^      operationId: (\w+)", line)
    if m:
        flush(); cur_op, cur_perm, cur_staff = m.group(1), None, False; continue
    if re.match(r"^  /", line) or re.match(r"^    (get|post|put|patch|delete):$", line):
        flush(); cur_op, cur_perm, cur_staff = None, None, False
    if cur_op is not None:
        if re.match(r"^      - staffAuth: \[\]|^      - staffAuth:", line): cur_staff = True
        p = re.match(r"^      x-required-permission: (\S+)", line)
        if p: cur_perm = p.group(1)
flush()
body = "\n".join(f'\t"{k[0].upper()+k[1:]}": "{v}",' for k, v in sorted(ops.items()))
out = root / "services/gateway/internal/api/permissions_gen.go"
out.write_text(f"""// Code généré par tools/gen_permissions.py — NE PAS MODIFIER.

package api

// RequiredPermission : permission exigée par opération du personnel ("" = libre-service d'un membre connecté).
var RequiredPermission = map[OperationName]string{{
{body}
}}
""")
print(f"{len(ops)} opérations du personnel, {sum(1 for v in ops.values() if v)} avec permission -> {out.relative_to(root)}")
