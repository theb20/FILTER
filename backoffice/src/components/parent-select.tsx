import { inputCls } from "@/components/ui/ui";
import type { TreeNode } from "@/lib/catalog";

type Row = TreeNode & { level: number };

/** Choix d'une catégorie parente, présenté dans l'ordre de l'arbre. Vide = racine. */
export function ParentSelect({ rows, value, exclude, label = "Catégorie parente" }: { rows: Row[]; value?: string | null; exclude?: Set<string>; label?: string }) {
  return (
    <label className="block text-sm font-medium text-night">
      {label}
      <select name="parentId" defaultValue={value ?? ""} className={inputCls}>
        <option value="">— Aucune (à la racine) —</option>
        {rows.filter((r) => !exclude?.has(r.id)).map((r) => (
          <option key={r.id} value={r.id}>{"  ".repeat(r.level)}{r.level > 0 ? "└ " : ""}{r.name.fr ?? r.slug}</option>
        ))}
      </select>
    </label>
  );
}
