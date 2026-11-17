import { inputCls, selectCls } from "@/components/ui/ui";
import { LocalizedFields } from "./localized-fields";
import type { FieldDef } from "@/lib/content-kinds";
import type { RefOptions } from "@/lib/content";

type Doc = Record<string, unknown>;

const isoLocal = (v: unknown) => (typeof v === "string" ? new Date(v).toISOString().slice(0, 16) : "");

/** Un champ du formulaire de contenu, d'après sa description. Les champs en lecture seule ne sont jamais des champs de saisie. */
export function ContentField({ f, value, refs }: { f: FieldDef; value?: unknown; refs: RefOptions }) {
  const t = f.type;
  const hint = f.help ? <span className="mt-1 block text-xs font-normal text-slate">{f.help}</span> : null;
  const title = <>{f.label}{!f.req && <span className="font-normal text-slate"> (facultatif)</span>}</>;
  if (t === "loc") return <LocalizedFields name={f.name} label={f.label} value={value as Record<string, string> | undefined} multiline={f.long} required={f.req} />;
  if (t === "bool") return <label className="flex items-center gap-2 text-sm font-medium text-night"><input type="checkbox" name={f.name} defaultChecked={value === true} className="h-4 w-4" />{f.label}</label>;
  if (t.startsWith("enum:")) {
    const opts = t.slice(5).split("|");
    return (
      <label className="block text-sm font-medium text-night">{title}
        <select name={f.name} required={f.req} defaultValue={(value as string) ?? (f.req ? opts[0] : "")} className={`${selectCls} mt-1.5 w-full`}>
          {!f.req && <option value="">—</option>}
          {opts.map((o) => <option key={o} value={o}>{f.labels?.[o] ?? o}</option>)}
        </select>{hint}
      </label>
    );
  }
  if (f.ref && refs[f.ref]) {
    return (
      <label className="block text-sm font-medium text-night">{title}
        <select name={f.name} required={f.req} defaultValue={(value as string) ?? ""} className={`${selectCls} mt-1.5 w-full`}>
          <option value="">—</option>
          {refs[f.ref].map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>{hint}
      </label>
    );
  }
  if (t === "text") return <label className="block text-sm font-medium text-night">{title}<textarea name={f.name} required={f.req} defaultValue={(value as string) ?? ""} rows={6} maxLength={20000} className={inputCls} />{hint}</label>;
  if (t === "strs" || t === "uuids") return <label className="block text-sm font-medium text-night">{title}<input name={f.name} defaultValue={Array.isArray(value) ? value.join(", ") : ""} className={inputCls} />{hint}</label>;
  if (t === "int" || t === "num") return <label className="block text-sm font-medium text-night">{title}<input type="number" name={f.name} required={f.req} min={f.min} max={f.max} step={t === "num" ? "any" : 1} defaultValue={(value as number | undefined) ?? ""} className={inputCls} />{hint}</label>;
  if (t === "ts") return <label className="block text-sm font-medium text-night">{title} <span className="font-normal text-slate">(heure UTC)</span><input type="datetime-local" name={f.name} defaultValue={isoLocal(value)} className={inputCls} />{hint}</label>;
  const type = t === "url" ? "url" : t === "email" ? "email" : "text";
  const pattern = t === "slug" ? "[a-z0-9]+(-[a-z0-9]+)*" : t === "code" ? "[A-Za-z0-9][A-Za-z0-9_\\-]{2,63}" : undefined;
  return <label className="block text-sm font-medium text-night">{title}<input type={type} name={f.name} required={f.req} pattern={pattern} maxLength={t === "slug" ? 80 : 255} defaultValue={(value as string) ?? ""} className={inputCls} />{hint}</label>;
}

export function ContentFields({ fields, value, refs }: { fields: FieldDef[]; value?: Doc; refs: RefOptions }) {
  return <>{fields.filter((f) => !f.ro && !f.gated && f.type !== "items").map((f) => <ContentField key={f.name} f={f} value={value?.[f.name]} refs={refs} />)}</>;
}

/** Valeur en lecture seule (messages reçus, lignes d'une demande de devis). */
export function ReadOnlyValue({ f, value }: { f: FieldDef; value: unknown }) {
  if (value === undefined || value === null || value === "") return <span className="text-slate">—</span>;
  if (f.type === "items" && Array.isArray(value)) {
    return (
      <ul className="space-y-1 text-sm">
        {(value as { reference: string; quantity: number; name?: string }[]).map((l, i) => <li key={i}>{l.quantity} × {l.reference}{l.name ? ` : ${l.name}` : ""}</li>)}
      </ul>
    );
  }
  if (f.type === "bool") return <>{value ? "Oui" : "Non"}</>;
  if (f.type.startsWith("enum:")) return <>{f.labels?.[String(value)] ?? String(value)}</>;
  return <span className="whitespace-pre-line break-words">{String(value)}</span>;
}
