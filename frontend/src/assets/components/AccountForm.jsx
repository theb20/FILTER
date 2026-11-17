import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import PasswordInput from "./PasswordInput.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { useNotify } from "../configurations/notificationContext.js";
import { useAuth } from "../store/authStore.js";
import WelcomeFlow, { CreatingAccount } from "./WelcomeFlow.jsx";

// [nom, clé i18n, type d'input, obligatoire, autocomplete]
const FIELDS = {
  particulier: [
    ["firstName", "auth.firstName", "text", true, "given-name"],
    ["lastName", "auth.lastName", "text", true, "family-name"],
    ["email", "auth.email", "email", true, "email"],
    ["phone", "auth.phone", "tel", false, "tel"],
    ["password", "auth.newPassword", "password", true, "new-password"],
    ["confirm", "auth.confirm", "password", true, "new-password"],
  ],
  entreprise: [
    ["companyName", "auth.companyName", "text", true, "organization"],
    ["taxId", "auth.taxId", "text", true, "off"],
    ["contactName", "auth.contactName", "text", true, "name"],
    ["jobTitle", "auth.jobTitle", "text", false, "organization-title"],
    ["email", "auth.workEmail", "email", true, "email"],
    ["phone", "auth.phone", "tel", true, "tel"],
    ["country", "auth.country", "text", true, "country-name"],
    ["password", "auth.newPassword", "password", true, "new-password"],
    ["confirm", "auth.confirm", "password", true, "new-password"],
  ],
};

const DIAL_CODE = "+225";
const DEFAULT_COUNTRY = "CI"; // Côte d'Ivoire

// Seul pays proposé pour l'instant (code ISO 3166-1) ; le nom suit la langue du site (Intl.DisplayNames).
const COUNTRY_CODES = [DEFAULT_COUNTRY];

// Exemples neutres (valables dans les 3 langues) affichés comme placeholder.
const PLACEHOLDERS = {
  firstName: "Kouassi",
  lastName: "Yao",
  email: "nom@exemple.com",
  phone: "0701234567",
  password: "••••••••",
  confirm: "••••••••",
  companyName: "Société Exemple SARL",
  taxId: "CI-1234567 A",
  contactName: "Kouassi Yao",
  jobTitle: "Responsable achats",
};

const input = "w-full rounded-[6px] border bg-white px-3.5 py-2.5 outline-none focus:ring-2 focus:ring-action";

// Une seule section est rendue à la fois (la page la remonte avec key={type}) :
// les champs de la section masquée n'existent pas et ne sont jamais validés.
export default function AccountForm({ type, onWelcomeDone }) {
  const { t, lang } = useI18n();
  const notify = useNotify();
  const register = useAuth((a) => a.register);
  const [pending, setPending] = useState(false);
  const [welcome, setWelcome] = useState(false);
  const [formError, setFormError] = useState("");
  const fields = FIELDS[type];
  const [data, setData] = useState({ country: DEFAULT_COUNTRY });
  const [terms, setTerms] = useState(false);
  const [errors, setErrors] = useState({});

  const countries = useMemo(() => {
    const names = new Intl.DisplayNames([lang], { type: "region" });
    return COUNTRY_CODES.map((code) => [code, names.of(code)]).sort((a, b) =>
      a[0] === DEFAULT_COUNTRY ? -1 : b[0] === DEFAULT_COUNTRY ? 1 : a[1].localeCompare(b[1], lang));
  }, [lang]);

  const validate = () => {
    const e = {};
    for (const [name, , kind, required] of fields) {
      const v = (data[name] ?? "").trim();
      if (required && !v) e[name] = "auth.err.required";
      else if (kind === "tel" && v && !/^\d{8,10}$/.test(v)) e[name] = "auth.err.phone";
      else if (kind === "email" && v && !/\S+@\S+\.\S+/.test(v)) e[name] = "auth.err.emailInvalid";
      else if (name === "password" && v && v.length < 8) e[name] = "auth.err.passwordShort";
      else if (name === "confirm" && v && v !== data.password) e[name] = "auth.err.match";
    }
    if (!terms) e.terms = "auth.err.terms";
    return e;
  };

  const onSubmit = async (ev) => {
    ev.preventDefault();
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length) return;
    const phone = data.phone ? DIAL_CODE + data.phone.trim() : null;
    const payload =
      type === "entreprise"
        ? {
            kind: "company", email: data.email.trim(), password: data.password, acceptTerms: true, locale: lang, phone,
            // Un seul champ « nom du contact » dans le formulaire : il est transmis comme prénom (le nom reste vide).
            firstName: data.contactName.trim(),
            company: { legalName: data.companyName.trim(), taxId: data.taxId.trim(), country: data.country || DEFAULT_COUNTRY, ...(data.jobTitle?.trim() ? { jobTitle: data.jobTitle.trim() } : {}) },
          }
        : { kind: "individual", email: data.email.trim(), password: data.password, acceptTerms: true, locale: lang, phone, firstName: data.firstName.trim(), lastName: data.lastName.trim() };
    setPending(true);
    setFormError("");
    const r = await register(payload);
    setPending(false);
    if (r.ok) {
      notify(t("auth.created"), "success", 8000);
      return setWelcome(true); // confidentialité puis guide
    }
    setFormError(t(r.error, { s: r.retryAfter ?? 60 }));
  };

  const change = (name, value) => {
    setData((d) => ({ ...d, [name]: value }));
    if (errors[name]) setErrors((x) => ({ ...x, [name]: undefined }));
  };

  return (
    <>
      {pending && <CreatingAccount />}
      {welcome && <WelcomeFlow type={type} onDone={() => onWelcomeDone?.(data.email?.trim() ?? "")} />}
      <form onSubmit={onSubmit} noValidate className="mt-5 grid gap-4 sm:grid-cols-2">
        {formError && <p role="alert" className="rounded-[6px] border border-red-300 bg-red-50 px-3.5 py-2.5 text-sm text-red-700 sm:col-span-2">{formError}</p>}
        {fields.map(([name, label, kind, required, auto]) => {
          const wide = ["email", "phone", "companyName", "taxId", "country"].includes(name) || kind === "password";
          const id = `acc-${type}-${name}`;
          return (
            <div key={name} className={wide ? "sm:col-span-2 w-full" : ""}>
              <label htmlFor={id} className="mb-1 block text-sm font-medium">
                {t(label)}{!required && <span className="font-normal text-gray-500"> {t("auth.optional")}</span>}
              </label>
              <div className="flex">
                {kind === "tel" && (
                  <span className="flex items-center rounded-l-[6px] border border-r-0 border-gray-300 bg-gray-100 px-3 text-gray-700" aria-hidden="true">
                    {DIAL_CODE}
                  </span>
                )}
                {name === "country" ? (
                  <select
                    id={id}
                    value={data[name] ?? DEFAULT_COUNTRY}
                    onChange={(e) => change(name, e.target.value)}
                    autoComplete="country"
                    aria-invalid={Boolean(errors[name])}
                    aria-describedby={errors[name] ? `${id}-err` : undefined}
                    className={`${input} ${errors[name] ? "border-red-500" : "border-gray-300"}`}
                  >
                    {countries.map(([code, label]) => <option key={code} value={code}>{label}</option>)}
                  </select>
                ) : kind === "password" ? (
                  <PasswordInput
                    id={id}
                    placeholder={PLACEHOLDERS[name]}
                    value={data[name] ?? ""}
                    onChange={(e) => change(name, e.target.value)}
                    autoComplete={auto}
                    aria-invalid={Boolean(errors[name])}
                    aria-describedby={errors[name] ? `${id}-err` : undefined}
                    className={`${input} ${errors[name] ? "border-red-500" : "border-gray-300"}`}
                  />
                ) : (
                  <input
                    id={id}
                    type={kind}
                    placeholder={PLACEHOLDERS[name]}
                    value={data[name] ?? ""}
                    onChange={(e) => change(name, kind === "tel" ? e.target.value.replace(/\D/g, "") : e.target.value)}
                    autoComplete={kind === "tel" ? "tel-national" : auto}
                    inputMode={kind === "tel" ? "numeric" : undefined}
                    aria-invalid={Boolean(errors[name])}
                    aria-describedby={[kind === "tel" && `${id}-hint`, errors[name] && `${id}-err`].filter(Boolean).join(" ") || undefined}
                    className={`${input} ${kind === "tel" ? "rounded-l-none" : ""} ${errors[name] ? "border-red-500" : "border-gray-300"}`}
                  />
                )}
              </div>
              {kind === "tel" && <p id={`${id}-hint`} className="mt-1 text-xs text-gray-600">{t("auth.phoneHint")}</p>}
              {errors[name] && <p id={`${id}-err`} className="mt-1 text-sm text-red-600">{t(errors[name])}</p>}
            </div>
          );
        })}

        <div className="sm:col-span-2">
          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              checked={terms}
              onChange={(e) => { setTerms(e.target.checked); setErrors((x) => ({ ...x, terms: undefined })); }}
              aria-invalid={Boolean(errors.terms)}
              className="mt-1 h-4 w-4 accent-slate-900"
            />
            <span>
              {t("auth.terms")}{" "}
              <Link to="/privacy" target="_blank" className="text-link underline">{t("auth.privacyLink")}</Link>
            </span>
          </label>
          {errors.terms && <p className="mt-1 text-sm text-red-600">{t(errors.terms)}</p>}
        </div>

        <div className="flex gap-3 sm:col-span-2">
          <button type="submit" disabled={pending} className="flex-1 rounded-[6px] bg-night px-5 py-3 font-medium text-white hover:bg-navy disabled:opacity-60">
            {pending ? t("auth.pending") : t("auth.signup")}
          </button>
        </div>
      </form>
    </>
  );
}
