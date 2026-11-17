import { inputCls } from "@/components/ui/ui";
import { LocalizedFields } from "./localized-fields";
import type { Brand, Category, Sector } from "@/lib/types";

const slugHint = "Minuscules, chiffres et tirets (ex. pompes-doseuses). Sert dans l'adresse du site.";

export function CategoryFields({ value }: { value?: Category }) {
  return (
    <>
      <label className="block text-sm font-medium text-night">Identifiant d’URL<input name="slug" required defaultValue={value?.slug} pattern="[a-z0-9]+(-[a-z0-9]+)*" maxLength={80} className={inputCls} /><span className="mt-1 block text-xs font-normal text-slate">{slugHint}</span></label>
      <LocalizedFields name="name" label="Nom" value={value?.name} />
      <LocalizedFields name="description" label="Description (facultatif)" value={value?.description} multiline required={false} />
    </>
  );
}

export function BrandFields({ value }: { value?: Brand }) {
  return (
    <>
      <label className="block text-sm font-medium text-night">Nom<input name="name" required defaultValue={value?.name} maxLength={120} className={inputCls} /></label>
      <label className="block text-sm font-medium text-night">Identifiant d’URL<input name="slug" required defaultValue={value?.slug} pattern="[a-z0-9]+(-[a-z0-9]+)*" maxLength={80} className={inputCls} /><span className="mt-1 block text-xs font-normal text-slate">{slugHint}</span></label>
      <label className="block text-sm font-medium text-night">Logo : adresse de l’image <span className="font-normal text-slate">(facultatif, https://…)</span><input name="logoUrl" type="url" defaultValue={value?.logoUrl ?? ""} className={inputCls} /></label>
    </>
  );
}

export function SectorFields({ value }: { value?: Sector }) {
  return (
    <>
      <label className="block text-sm font-medium text-night">Identifiant d’URL<input name="slug" required defaultValue={value?.slug} pattern="[a-z0-9]+(-[a-z0-9]+)*" maxLength={80} className={inputCls} /><span className="mt-1 block text-xs font-normal text-slate">{slugHint}</span></label>
      <LocalizedFields name="name" label="Nom" value={value?.name} />
      <label className="block text-sm font-medium text-night">Image : adresse <span className="font-normal text-slate">(facultatif, https://…)</span><input name="imageUrl" type="url" defaultValue={value?.imageUrl ?? ""} className={inputCls} /></label>
    </>
  );
}
