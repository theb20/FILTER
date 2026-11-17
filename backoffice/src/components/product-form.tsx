import { inputCls } from "@/components/ui/ui";
import { LocalizedFields } from "./localized-fields";
import { ATTRIBUTE_TYPE_LABEL, type AttributeDef, type Brand, type Product, type Sector } from "@/lib/types";

const label = (l: Record<string, string>) => l.fr ?? Object.values(l)[0] ?? "";
const mediaText = (p?: Product) => (p?.media ?? []).map((m) => (m.alt ? `${m.url} ; ${m.alt}` : m.url)).join("\n");
const refsText = (p?: Product) => (p?.crossReferences ?? []).map((x) => `${x.brand} ; ${x.number} ; ${x.kind}`).join("\n");
const num = (v: number | null | undefined) => (v === null || v === undefined ? "" : String(v));

function AttrInput({ def, value }: { def: AttributeDef; value?: string | number | boolean }) {
  const name = `attr_${def.key}`;
  const title = `${label(def.label)}${def.unit ? ` (${def.unit})` : ""}${def.required ? " *" : ""}`;
  let field: React.ReactNode;
  if (def.type === "boolean") {
    field = (
      <select name={name} defaultValue={value === undefined ? "" : String(value)} className={inputCls}>
        <option value="">—</option><option value="true">Oui</option><option value="false">Non</option>
      </select>
    );
  } else if (def.type === "enum") {
    field = (
      <select name={name} defaultValue={value === undefined ? "" : String(value)} className={inputCls}>
        <option value="">—</option>
        {def.options.map((o) => <option key={o.value} value={o.value}>{label(o.label)}</option>)}
      </select>
    );
  } else {
    field = <input name={name} defaultValue={value === undefined ? "" : String(value)} inputMode={def.type === "number" ? "decimal" : undefined} className={inputCls} />;
  }
  return (
    <label className="block text-sm font-medium text-night">
      {title} <span className="font-normal text-slate">· {ATTRIBUTE_TYPE_LABEL[def.type]}</span>
      {field}
      <input type="hidden" name={`attrtype_${def.key}`} value={def.type} />
    </label>
  );
}

function Num({ name, text, value, hint }: { name: string; text: string; value?: number | null; hint?: string }) {
  return (
    <label className="block text-sm font-medium text-night">{text}
      <input name={name} inputMode="numeric" pattern="[0-9]*" defaultValue={num(value)} className={inputCls} />
      {hint && <span className="mt-1 block text-xs font-normal text-slate">{hint}</span>}
    </label>
  );
}

/** Fiche produit (création et modification). La catégorie est fixée avant d'arriver ici : ses attributs en dépendent. */
export function ProductForm({ product, categoryId, categoryName, defs, brands, sectors }: { product?: Product; categoryId: string; categoryName: string; defs: AttributeDef[]; brands: Brand[]; sectors: Sector[] }) {
  const frozen = !!product?.publishedAt;
  const vals = new Map((product?.attributes ?? []).map((a) => [a.key, a.value]));
  return (
    <>
      <input type="hidden" name="categoryId" value={categoryId} />
      {product && <input type="hidden" name="version" value={product.version} />}
      <p className="text-sm text-slate">Catégorie : <strong className="text-night">{categoryName}</strong></p>

      <LocalizedFields name="name" label="Nom" value={product?.name} />
      <LocalizedFields name="description" label="Description (facultatif)" value={product?.description} multiline required={false} />

      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm font-medium text-night">SKU
          <input name="sku" defaultValue={product?.sku ?? ""} readOnly={frozen} placeholder="Généré automatiquement si vide" pattern="[A-Za-z0-9]+(-[A-Za-z0-9]+)*" maxLength={64} className={`${inputCls} ${frozen ? "bg-page" : ""}`} />
          <span className="mt-1 block text-xs font-normal text-slate">Majuscules, chiffres et tirets. {frozen ? "Figé : le produit a été publié." : "Figé après la première publication."}</span>
        </label>
        <label className="block text-sm font-medium text-night">Adresse (slug)
          <input name="slug" defaultValue={product?.slug ?? ""} readOnly={frozen} placeholder="Calculée depuis le nom si vide" pattern="[a-z0-9]+(-[a-z0-9]+)*" maxLength={80} className={`${inputCls} ${frozen ? "bg-page" : ""}`} />
          <span className="mt-1 block text-xs font-normal text-slate">{frozen ? "Figée : le produit a été publié." : "Figée après la première publication."}</span>
        </label>
        <label className="block text-sm font-medium text-night">Marque
          <select name="brandId" defaultValue={product?.brandId ?? ""} className={inputCls}>
            <option value="">— Aucune —</option>
            {brands.map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
          </select>
        </label>
        <label className="block text-sm font-medium text-night">Référence constructeur
          <input name="manufacturerRef" defaultValue={product?.manufacturerRef ?? ""} maxLength={100} className={inputCls} />
          <span className="mt-1 block text-xs font-normal text-slate">Unique par marque (espaces et ponctuation ignorés).</span>
        </label>
      </div>

      <fieldset className="space-y-3 rounded border border-line p-4">
        <legend className="px-1 text-sm font-semibold text-night">Prix et commande</legend>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block text-sm font-medium text-night">Mode de prix
            <select name="priceMode" defaultValue={product?.priceMode ?? "fixed"} className={inputCls}>
              <option value="fixed">Prix fixe</option><option value="on_quote">Sur devis</option>
            </select>
          </label>
          <Num name="priceAmount" text="Prix de vente HORS TAXES (F CFA)" value={product?.priceAmount} hint="Entier, sans décimale. Ignoré pour un produit sur devis." />
          <label className="block text-sm font-medium text-night">Disponibilité
            <select name="fulfillmentMode" defaultValue={product?.fulfillmentMode ?? "stock"} className={inputCls}>
              <option value="stock">Sur stock</option><option value="made_to_order">Fabriqué à la commande</option>
            </select>
          </label>
          <Num name="leadTimeDays" text="Délai (jours)" value={product?.leadTimeDays} hint="Obligatoire pour « fabriqué à la commande »." />
          <Num name="minOrderQuantity" text="Quantité minimale de commande" value={product?.minOrderQuantity} />
          <label className="block text-sm font-medium text-night">Unité de vente
            <input name="saleUnit" defaultValue={product?.saleUnit ?? ""} placeholder="pièce, lot de 10…" maxLength={50} className={inputCls} />
          </label>
        </div>
      </fieldset>

      <fieldset className="space-y-3 rounded border border-line p-4">
        <legend className="px-1 text-sm font-semibold text-night">Identification et dimensions <span className="font-normal text-slate">(facultatif)</span></legend>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block text-sm font-medium text-night">GTIN / EAN
            <input name="gtin" defaultValue={product?.gtin ?? ""} inputMode="numeric" pattern="[0-9]{8,14}" className={inputCls} />
            <span className="mt-1 block text-xs font-normal text-slate">8, 12, 13 ou 14 chiffres ; le chiffre de contrôle est vérifié.</span>
          </label>
          <Num name="weightGrams" text="Poids (grammes)" value={product?.weightGrams} />
          <Num name="lengthMm" text="Longueur (mm)" value={product?.lengthMm} />
          <Num name="widthMm" text="Largeur (mm)" value={product?.widthMm} />
          <Num name="heightMm" text="Hauteur (mm)" value={product?.heightMm} />
        </div>
      </fieldset>

      {defs.length > 0 && (
        <fieldset className="space-y-3 rounded border border-line p-4">
          <legend className="px-1 text-sm font-semibold text-night">Attributs de la catégorie <span className="font-normal text-slate">(* obligatoire pour publier)</span></legend>
          <div className="grid gap-4 sm:grid-cols-2">{defs.map((d) => <AttrInput key={d.id} def={d} value={vals.get(d.key)} />)}</div>
        </fieldset>
      )}

      {sectors.length > 0 && (
        <fieldset className="rounded border border-line p-4">
          <legend className="px-1 text-sm font-semibold text-night">Secteurs d’activité</legend>
          <div className="flex flex-wrap gap-x-5 gap-y-2">
            {sectors.map((s) => (
              <label key={s.id} className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="sectorIds" value={s.id} defaultChecked={product?.sectorIds.includes(s.id)} />{label(s.name)}</label>
            ))}
          </div>
        </fieldset>
      )}

      <label className="block text-sm font-medium text-night">Images : une adresse par ligne <span className="font-normal text-slate">(la première est l’image principale)</span>
        <textarea name="media" rows={4} defaultValue={mediaText(product)} placeholder={"https://… ; texte alternatif"} className={`${inputCls} font-mono text-sm`} />
        <span className="mt-1 block text-xs font-normal text-slate">Adresses https:// ; texte alternatif facultatif après « ; ». Le téléversement de fichiers viendra avec le stockage.</span>
      </label>
      <label className="block text-sm font-medium text-night">Références croisées <span className="font-normal text-slate">(facultatif)</span>
        <textarea name="crossReferences" rows={3} defaultValue={refsText(product)} placeholder={"SKF ; 6204-2RS ; equivalent"} className={`${inputCls} font-mono text-sm`} />
        <span className="mt-1 block text-xs font-normal text-slate">Une ligne : marque ; numéro ; type (oem = d’origine, equivalent, supersedes = remplace).</span>
      </label>
    </>
  );
}
