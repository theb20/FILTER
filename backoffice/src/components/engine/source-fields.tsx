import { inputCls, selectCls } from "@/components/ui/ui";
import { BRAVE_PRESET, MICROS, SOURCE_KIND_LABEL } from "@/lib/engine";
import type { Source } from "@/lib/types";

const num = (v: number | null | undefined) => (v === null || v === undefined ? "" : String(v));
/** Micro-unités → saisie dans la devise, sans zéros inutiles (5000 → « 0,005 »). */
const amount = (micros: number | undefined) => (micros === undefined ? "" : String(micros / MICROS).replace(".", ","));

const lab = "block text-sm font-medium text-night";
const hint = (t: string) => <span className="mt-1 block text-xs font-normal text-slate">{t}</span>;

/** Champs d'une source externe. Sans valeur, le formulaire est prérempli pour Brave Search. */
export function SourceFields({ value }: { value?: Source }) {
  const p = BRAVE_PRESET;
  return (
    <div className="space-y-4">
      <label className={lab}>Type de source
        <select name="kind" defaultValue={value?.kind ?? p.kind} className={`${selectCls} mt-1.5 w-full`}>
          {Object.entries(SOURCE_KIND_LABEL).map(([k, l]) => <option key={k} value={k}>{l}</option>)}
        </select>
      </label>
      <label className={lab}>Nom<input name="name" required maxLength={120} defaultValue={value?.name ?? p.name} className={inputCls} /></label>
      <label className={lab}>Adresse de l&apos;API<input name="baseUrl" type="url" required defaultValue={value?.baseUrl ?? p.baseUrl} className={inputCls} />{hint("Point d'entrée exact appelé par le service, ex. https://api.search.brave.com/res/v1/web/search")}</label>
      <label className={lab}>Variable d&apos;environnement qui porte la clé
        <input name="secretRef" required defaultValue={value?.secretRef ?? p.secretRef} pattern="[A-Z][A-Z0-9_]{2,63}" title="Nom de variable : majuscules, chiffres et soulignés, par exemple BRAVE_SEARCH_API_KEY" className={inputCls} />
        {hint("Uniquement le NOM de la variable (ex. BRAVE_SEARCH_API_KEY) — surtout pas la clé, ni la ligne « NOM=valeur » : ce champ est enregistré en base et au journal d'audit. La clé, elle, se met dans backend/.env puis ./dev.sh up.")}
      </label>
      <div className="grid gap-4 sm:grid-cols-2">
        <label className={lab}>Coût maximal d&apos;un appel<input name="costPerCall" required inputMode="decimal" defaultValue={value ? amount(value.costPerCallMicros) : p.costPerCall} className={inputCls} />{hint("Dans la devise ci-contre, ex. 0,005. Sert à réserver le budget avant l'appel.")}</label>
        <label className={lab}>Devise de facturation<input name="currency" required maxLength={3} pattern="[A-Za-z]{3}" defaultValue={value?.currency ?? p.currency} className={`${inputCls} uppercase`} />{hint("Code à trois lettres, ex. USD. Les devises ne sont jamais additionnées entre elles.")}</label>
        <label className={lab}>Priorité<input name="priority" type="number" min={0} required defaultValue={num(value?.priority ?? p.priority)} className={inputCls} />{hint("Plus la valeur est basse, plus la source est interrogée tôt ; même valeur = en parallèle.")}</label>
        <label className={lab}>Délai maximal (ms)<input name="timeoutMs" type="number" min={1} required defaultValue={num(value?.timeoutMs ?? p.timeoutMs)} className={inputCls} /></label>
        <label className={lab}>Tentatives supplémentaires<input name="maxRetries" type="number" min={0} required defaultValue={num(value?.maxRetries ?? p.maxRetries)} className={inputCls} />{hint("Uniquement sur erreur réseau ou 5xx.")}</label>
        <label className={lab}>Durée de cache (secondes)<input name="maxCacheTtlSeconds" type="number" min={1} defaultValue={num(value?.maxCacheTtlSeconds)} className={inputCls} />{hint("Vide = aucune mise en cache. À ne remplir que si les conditions d'usage de la source l'autorisent.")}</label>
      </div>
    </div>
  );
}
