import { inputCls } from "@/components/ui/ui";
import { LocalizedFields } from "./localized-fields";
import { ATTRIBUTE_TYPE_LABEL, type AttributeDef } from "@/lib/types";

const optionsText = (a?: AttributeDef) => (a?.options ?? []).map((o) => [o.value, o.label.fr ?? "", o.label.en ?? "", o.label.es ?? ""].join(" ; ").replace(/( ; )+$/, "")).join("\n");

export function AttributeFields({ categoryId, value }: { categoryId: string; value?: AttributeDef }) {
  const frozen = !!value; // la clé et le type sont figés : les produits s'y réfèrent
  return (
    <>
      <input type="hidden" name="categoryId" value={categoryId} />
      <label className="block text-sm font-medium text-night">Clé technique<input name="key" required defaultValue={value?.key} readOnly={frozen} pattern="[a-z][a-z0-9_]{1,63}" maxLength={64} className={`${inputCls} ${frozen ? "bg-page" : ""}`} /><span className="mt-1 block text-xs font-normal text-slate">Minuscules, chiffres et _ (ex. debit_max). {frozen ? "Non modifiable." : "Ne pourra plus changer."}</span></label>
      <LocalizedFields name="label" label="Libellé" value={value?.label} />
      <label className="block text-sm font-medium text-night">Type
        <select name="type" defaultValue={value?.type ?? "text"} className={`${inputCls} ${frozen ? "bg-page" : ""}`} {...(frozen ? { "data-frozen": true } : {})}>
          {(Object.keys(ATTRIBUTE_TYPE_LABEL) as (keyof typeof ATTRIBUTE_TYPE_LABEL)[]).map((k) => <option key={k} value={k} disabled={frozen && k !== value?.type}>{ATTRIBUTE_TYPE_LABEL[k]}</option>)}
        </select>
        {frozen && <span className="mt-1 block text-xs font-normal text-slate">Le type ne peut pas changer.</span>}
      </label>
      <label className="block text-sm font-medium text-night">Unité <span className="font-normal text-slate">(nombres uniquement, ex. m³/h)</span><input name="unit" defaultValue={value?.unit ?? ""} maxLength={20} className={inputCls} /></label>
      <label className="block text-sm font-medium text-night">Choix possibles <span className="font-normal text-slate">(type « Liste de choix » uniquement)</span>
        <textarea name="options" rows={5} defaultValue={optionsText(value)} placeholder={"inox ; Inox ; Stainless steel ; Acero inoxidable\nlaiton ; Laiton"} className={`${inputCls} font-mono text-sm`} />
        <span className="mt-1 block text-xs font-normal text-slate">Une ligne par choix : valeur ; libellé français ; libellé anglais ; libellé espagnol (les deux derniers sont facultatifs).</span>
      </label>
      <label className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="filterable" defaultChecked={value?.filterable} /> Utilisable comme filtre sur le site <span className="text-slate">(pas pour le texte libre)</span></label>
      <label className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="required" defaultChecked={value?.required} /> Obligatoire pour publier un produit de cette catégorie</label>
      <label className="block text-sm font-medium text-night">Ordre d’affichage<input name="sortOrder" type="number" min={0} defaultValue={value?.sortOrder ?? 0} className={inputCls} /></label>
    </>
  );
}
