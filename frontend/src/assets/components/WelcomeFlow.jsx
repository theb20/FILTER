import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Building2, ChevronLeft, ChevronRight, FileText, Languages, Loader2, Mail, Search, ShieldCheck, ShoppingCart, Tag, Truck, UserRound, X } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";

// Après l'inscription : 1) écran de chargement, 2) fenêtre « confidentialité » (à lire/accepter), 3) guide visuel de tout ce
// qu'un client peut faire sur le site. `type` : "particulier" | "entreprise" (une étape de plus pour les comptes pro).
const GUIDE = [
  { key: "search", Icon: Search },
  { key: "catalog", Icon: Tag },
  { key: "cart", Icon: ShoppingCart },
  { key: "promo", Icon: Truck },
  { key: "quote", Icon: FileText },
  { key: "pro", Icon: Building2, onlyCompany: true },
  { key: "account", Icon: UserRound },
  { key: "lang", Icon: Languages },
  { key: "contact", Icon: Mail },
];

/** Écran de chargement plein écran (création du compte en cours). */
export function CreatingAccount() {
  const { t } = useI18n();
  return (
    <div role="alertdialog" aria-modal="true" aria-live="assertive" aria-label={t("welcome.creating")} className="fixed inset-0 z-[2000] flex flex-col items-center justify-center gap-4 bg-white/90 backdrop-blur-sm">
      <Loader2 size={44} className="animate-spin text-night" aria-hidden="true" />
      <p className="text-lg font-semibold">{t("welcome.creating")}</p>
    </div>
  );
}

function Modal({ children, label, onClose }) {
  const ref = useRef(null);
  useEffect(() => {
    const prev = document.activeElement;
    ref.current?.focus();
    const onKey = (e) => {
      if (e.key === "Escape" && onClose) onClose();
      if (e.key === "Tab") { // le focus reste dans la fenêtre
        const f = ref.current?.querySelectorAll("a[href], button:not([disabled]), input, [tabindex]:not([tabindex='-1'])");
        if (!f?.length) return;
        const first = f[0], last = f[f.length - 1];
        if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
        else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
      }
    };
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => { document.removeEventListener("keydown", onKey); document.body.style.overflow = ""; prev?.focus?.(); };
  }, [onClose]);
  return (
    <div className="fixed inset-0 z-[2000] flex items-center justify-center bg-black/60 p-4">
      <div ref={ref} role="dialog" aria-modal="true" aria-label={label} tabIndex={-1} className="relative max-h-[92vh] w-full max-w-lg overflow-y-auto rounded-[12px] bg-white p-6 shadow-2xl outline-none sm:p-8">
        {children}
      </div>
    </div>
  );
}

function PrivacyStep({ onAccept }) {
  const { t } = useI18n();
  const [read, setRead] = useState(false);
  return (
    <Modal label={t("welcome.privacy.title")}>
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-night text-action"><ShieldCheck size={26} aria-hidden="true" /></span>
      <h2 className="mt-4 text-[24px] font-bold leading-tight">{t("welcome.privacy.title")}</h2>
      <p className="mt-2 text-gray-700">{t("welcome.privacy.lead")}</p>
      <ul className="mt-4 space-y-2.5 text-[15px]">
        {[1, 2, 3, 4].map((n) => (
          <li key={n} className="flex gap-2.5"><span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-action" aria-hidden="true" /><span>{t(`welcome.privacy.p${n}`)}</span></li>
        ))}
      </ul>
      <Link to="/privacy" target="_blank" rel="noopener noreferrer" className="mt-4 inline-block font-medium text-link underline">{t("welcome.privacy.full")}</Link>
      <label className="mt-5 flex items-start gap-2.5 rounded-[8px] bg-gray-50 p-3 text-sm">
        <input type="checkbox" checked={read} onChange={(e) => setRead(e.target.checked)} className="mt-0.5 h-4 w-4 accent-slate-900" />
        <span>{t("welcome.privacy.ack")}</span>
      </label>
      <button type="button" disabled={!read} onClick={onAccept} className="mt-5 w-full rounded-[8px] bg-night px-5 py-3 font-medium text-white hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50">
        {t("welcome.privacy.continue")}
      </button>
    </Modal>
  );
}

function GuideStep({ type, onDone }) {
  const { t } = useI18n();
  const steps = GUIDE.filter((s) => !s.onlyCompany || type === "entreprise");
  const [i, setI] = useState(0);
  const s = steps[i];
  const last = i === steps.length - 1;
  return (
    <Modal label={t("welcome.guide.title")} onClose={onDone}>
      <button type="button" onClick={onDone} aria-label={t("welcome.guide.skip")} className="absolute right-3 top-3 rounded-full p-2 text-gray-500 hover:bg-gray-100 hover:text-black"><X size={18} aria-hidden="true" /></button>
      <p className="text-xs font-bold uppercase tracking-wide text-gray-500">{t("welcome.guide.title")} · {i + 1}/{steps.length}</p>
      <div className="mt-4 flex aspect-[16/9] items-center justify-center rounded-[10px] bg-gradient-to-br from-navy to-night text-action" aria-hidden="true">
        <s.Icon size={72} strokeWidth={1.4} />
      </div>
      <h2 className="mt-5 text-[24px] font-bold leading-tight" aria-live="polite">{t(`welcome.g.${s.key}.t`)}</h2>
      <p className="mt-2 min-h-[4.5rem] text-gray-700">{t(`welcome.g.${s.key}.d`)}</p>
      <div className="mt-3 flex justify-center gap-1.5" role="tablist" aria-label={t("welcome.guide.title")}>
        {steps.map((x, n) => (
          <button key={x.key} type="button" role="tab" aria-selected={n === i} aria-label={t(`welcome.g.${x.key}.t`)} onClick={() => setI(n)} className={`h-2 rounded-full transition-all ${n === i ? "w-6 bg-night" : "w-2 bg-gray-300 hover:bg-gray-400"}`} />
        ))}
      </div>
      <div className="mt-6 flex items-center justify-between gap-3">
        <button type="button" onClick={() => setI((v) => Math.max(0, v - 1))} disabled={i === 0} className="inline-flex items-center gap-1 rounded-[8px] border border-gray-300 px-4 py-2.5 font-medium hover:bg-gray-50 disabled:invisible">
          <ChevronLeft size={16} aria-hidden="true" /> {t("welcome.guide.prev")}
        </button>
        {last ? (
          <button type="button" onClick={onDone} className="rounded-[8px] bg-action px-5 py-2.5 font-bold text-black hover:bg-action-dark">{t("welcome.guide.finish")}</button>
        ) : (
          <button type="button" onClick={() => setI((v) => v + 1)} className="inline-flex items-center gap-1 rounded-[8px] bg-night px-5 py-2.5 font-medium text-white hover:opacity-90">
            {t("welcome.guide.next")} <ChevronRight size={16} aria-hidden="true" />
          </button>
        )}
      </div>
    </Modal>
  );
}

/** Enchaîne confidentialité puis guide ; `onDone` renvoie le client vers la connexion. */
export default function WelcomeFlow({ type, onDone }) {
  const [step, setStep] = useState("privacy");
  return step === "privacy" ? <PrivacyStep onAccept={() => setStep("guide")} /> : <GuideStep type={type} onDone={onDone} />;
}

