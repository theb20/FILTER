import { inputCls } from "@/components/ui/ui";
import type { Localized } from "@/lib/types";

const LANGS = [
  { code: "fr", label: "Français", required: true },
  { code: "en", label: "Anglais", required: false },
  { code: "es", label: "Espagnol", required: false },
] as const;

/** Un champ par langue (`<name>_fr`, `<name>_en`…). Le français est obligatoire ; une langue vide se replie sur lui sur le site. */
export function LocalizedFields({ name, label, value, multiline = false, required = true }: { name: string; label: string; value?: Localized | null; multiline?: boolean; required?: boolean }) {
  return (
    <fieldset className="space-y-2">
      <legend className="text-sm font-medium text-night">{label}</legend>
      {LANGS.map((l) => {
        const props = { name: `${name}_${l.code}`, defaultValue: value?.[l.code] ?? "", required: required && l.required, className: inputCls.replace("mt-1.5", "mt-0"), "aria-label": `${label} (${l.label})` };
        return (
          <div key={l.code} className="flex items-start gap-2">
            <span className="mt-2 w-20 shrink-0 text-xs text-slate">{l.label}{required && l.required ? " *" : ""}</span>
            {multiline ? <textarea rows={3} {...props} /> : <input {...props} />}
          </div>
        );
      })}
    </fieldset>
  );
}
