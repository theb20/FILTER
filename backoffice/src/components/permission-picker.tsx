import type { Permission } from "@/lib/types";

/** Cases à cocher des permissions, regroupées par domaine. `readOnly` : affichage seul. */
export function PermissionPicker({ permissions, selected, readOnly }: { permissions: Permission[]; selected: string[]; readOnly?: boolean }) {
  const groups = new Map<string, Permission[]>();
  for (const p of permissions) groups.set(p.group, [...(groups.get(p.group) ?? []), p]);
  const all = selected.includes("*");
  return (
    <div className="space-y-4">
      {all && <p className="rounded bg-action/15 px-3 py-2 text-sm text-night">Ce rôle donne <strong>toutes les permissions</strong> (présentes et futures).</p>}
      {[...groups.entries()].map(([group, items]) => (
        <fieldset key={group} className="rounded border border-line p-3">
          <legend className="px-1 text-sm font-semibold text-night">{group}</legend>
          <div className="grid gap-x-4 gap-y-1 sm:grid-cols-2">
            {items.map((p) => (
              <label key={p.code} className="flex items-start gap-2 text-sm">
                <input type="checkbox" name="permissions" value={p.code} defaultChecked={all || selected.includes(p.code)} disabled={readOnly || all} className="mt-1 h-4 w-4 accent-night" />
                <span><span className="text-night">{p.description.split(" — ")[0]}</span> <span className="font-mono text-xs text-slate">{p.code}</span></span>
              </label>
            ))}
          </div>
        </fieldset>
      ))}
    </div>
  );
}
