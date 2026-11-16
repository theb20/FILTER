import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Building2, Check, Eye, EyeOff, User } from "lucide-react";
import LanguageSwitcher from "../components/LanguageSwitcher.jsx";
import AccountForm from "../components/AccountForm.jsx";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import { SITE } from "../config/site.js";
import Logo from "../components/Logo.jsx";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const ACCOUNT_TYPES = [
  { type: "particulier", Icon: User, label: "auth.individual", desc: "auth.individual.d" },
  { type: "entreprise", Icon: Building2, label: "auth.company", desc: "auth.company.d" },
];

const inputBase =
  "w-full rounded-[8px] border bg-white px-3.5 py-2.5 text-[15px] outline-none transition-shadow " +
  "placeholder:text-gray-400 focus:border-transparent focus:ring-2 focus:ring-action";

const fieldClass = (hasError) => `${inputBase} ${hasError ? "border-red-500" : "border-gray-300"}`;

function GoogleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 48 48" aria-hidden="true" focusable="false">
      <path fill="#EA4335" d="M24 9.5c3.5 0 6.6 1.2 9.1 3.6l6.8-6.8C35.8 2.4 30.3 0 24 0 14.6 0 6.5 5.4 2.6 13.2l7.9 6.1C12.4 13.6 17.7 9.5 24 9.5z" />
      <path fill="#4285F4" d="M46.1 24.5c0-1.6-.1-3.1-.4-4.5H24v9h12.4c-.5 2.9-2.2 5.3-4.6 6.9l7.5 5.8c4.4-4 6.8-10 6.8-17.2z" />
      <path fill="#FBBC05" d="M10.5 28.7c-.5-1.4-.8-3-.8-4.7s.3-3.3.8-4.7l-7.9-6.1C.9 16.5 0 20.1 0 24s.9 7.5 2.6 10.8l7.9-6.1z" />
      <path fill="#34A853" d="M24 48c6.5 0 11.9-2.1 15.9-5.8l-7.5-5.8c-2.1 1.4-4.8 2.3-8.4 2.3-6.3 0-11.6-4.1-13.5-9.8l-7.9 6.1C6.5 42.6 14.6 48 24 48z" />
    </svg>
  );
}

export default function Sign() {
  const { t } = useI18n();
  const notify = useNotify();
  const [isLogin, setIsLogin] = useState(true);
  const [accountType, setAccountType] = useState("particulier"); // "particulier" | "entreprise"
  const [data, setData] = useState({ email: "", password: "" });
  const [errors, setErrors] = useState({});
  const [showPassword, setShowPassword] = useState(false);
  const emailRef = useRef(null);
  const passwordRef = useRef(null);

  useEffect(() => {
    document.title = t("meta.sign");
  }, [t]);

  const toggle = () => {
    setIsLogin((v) => !v);
    setErrors({});
    setShowPassword(false);
  };

  const onSubmit = (ev) => {
    ev.preventDefault();
    const e = {};
    const email = data.email.trim();
    if (!email) e.email = "auth.err.email";
    else if (!EMAIL_RE.test(email)) e.email = "auth.err.emailInvalid";
    if (!data.password) e.password = "auth.err.password";
    setErrors(e);
    if (Object.keys(e).length) {
      // Place le focus sur le premier champ en erreur.
      (e.email ? emailRef : passwordRef).current?.focus();
      return;
    }
    // Aucune authentification réelle : on ne simule pas de succès.
    notify(t("demo.notice"), "demo");
  };

  const change = (ev) => {
    const { name, value } = ev.target;
    setData((d) => ({ ...d, [name]: value }));
    if (errors[name]) setErrors((x) => ({ ...x, [name]: undefined }));
  };

  return (
    <div className="grid min-h-screen bg-white lg:grid-cols-2">
      {/* Panneau visuel (desktop) — reste fixe pendant que le formulaire défile */}
      <aside className="relative hidden overflow-hidden bg-navy text-white lg:sticky lg:top-0 lg:flex lg:h-screen lg:flex-col lg:justify-between lg:self-start">
        <img
          src="/img/background/bg_sign.webp"
          alt=""
          className="absolute inset-0 h-full w-full object-cover"
        />
        <div
          className="absolute inset-0 bg-gradient-to-t from-night via-night/50 to-night/20"
          aria-hidden="true"
        />
        <Logo className="relative p-8 text-xl font-bold" />
        <div className="relative p-10 xl:p-14">
          <p className="max-w-md text-[34px] font-bold leading-[1.15] tracking-tight">
            {t("auth.panel.title")}
          </p>
          <p className="mt-4 max-w-sm text-[15px] leading-relaxed text-white/85">
            {t("auth.panel.text")}
          </p>
        </div>
      </aside>

      {/* Colonne formulaire */}
      <div className="flex min-h-screen flex-col">
        <header className="flex items-center justify-between gap-4 px-4 py-4 sm:px-10">
          <Logo className="text-lg text-black font-bold lg:invisible" />
          <div className="rounded-[8px] bg-night text-white">
            <LanguageSwitcher className="px-2 py-1 " />
          </div>
        </header>

        <main
          className="flex flex-1 items-center justify-center px-4 pb-16 pt-6 sm:px-10"
          aria-labelledby="sign-title"
        >
          <div className="w-full max-w-[400px]">
            <h1 id="sign-title" className="text-[28px] font-bold leading-tight tracking-tight">
              {isLogin ? t("auth.welcomeBack") : t("auth.welcome", { name: SITE.name })}
            </h1>
            <p className="mt-2 text-[15px] text-gray-600">
              {isLogin ? t("auth.lead.signin") : t("auth.lead.signup")}
            </p>

            <button
              type="button"
              onClick={() => notify(t("demo.notice"), "demo")}
              className="mt-8 flex w-full items-center justify-center gap-2.5 rounded-[8px] border border-gray-300 bg-white px-5 py-2.5 text-sm font-medium transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-action"
            >
              <GoogleIcon /> {isLogin ? t("auth.googleIn") : t("auth.googleUp")}
            </button>

            <div className="my-6 flex items-center gap-3 text-xs uppercase tracking-wide text-gray-500">
              <span className="h-px flex-1 bg-gray-200" aria-hidden="true" />
              {t("auth.or")}
              <span className="h-px flex-1 bg-gray-200" aria-hidden="true" />
            </div>

            {isLogin ? (
              <form onSubmit={onSubmit} noValidate className="space-y-5">
                <div>
                  <label htmlFor="email" className="mb-1.5 block text-sm font-medium">
                    {t("auth.email")}
                  </label>
                  <input
                    ref={emailRef}
                    id="email"
                    name="email"
                    type="email"
                    placeholder={t("auth.placeholder.email")}
                    inputMode="email"
                    autoComplete="email"
                    autoCapitalize="none"
                    spellCheck={false}
                    required
                    value={data.email}
                    onChange={change}
                    aria-invalid={Boolean(errors.email)}
                    aria-describedby={errors.email ? "email-error" : undefined}
                    className={fieldClass(errors.email)}
                  />
                  {errors.email && (
                    <p id="email-error" role="alert" className="mt-1.5 text-sm text-red-600">
                      {t(errors.email)}
                    </p>
                  )}
                </div>

                <div>
                  <div className="mb-1.5 flex items-baseline justify-between gap-3">
                    <label htmlFor="password" className="block text-sm font-medium">
                      {t("auth.password")}
                    </label>
                    <Link to="/reset" className="text-sm text-link hover:underline">
                      {t("auth.forgot")}
                    </Link>
                  </div>
                  <div className="relative">
                    <input
                      ref={passwordRef}
                      id="password"
                      name="password"
                      type={showPassword ? "text" : "password"}
                      autoComplete="current-password"
                      placeholder={t("auth.placeholder.password")}
                      required
                      value={data.password}
                      onChange={change}
                      aria-invalid={Boolean(errors.password)}
                      aria-describedby={errors.password ? "password-error" : undefined}
                      className={`${fieldClass(errors.password)} pr-11`}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword((v) => !v)}
                      aria-label={showPassword ? t("auth.hidePassword") : t("auth.showPassword")}
                      aria-pressed={showPassword}
                      className="absolute inset-y-0 right-0 flex w-11 items-center justify-center rounded-r-[8px] text-gray-500 hover:text-black focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-action"
                    >
                      {showPassword ? <EyeOff size={18} aria-hidden="true" /> : <Eye size={18} aria-hidden="true" />}
                    </button>
                  </div>
                  {errors.password && (
                    <p id="password-error" role="alert" className="mt-1.5 text-sm text-red-600">
                      {t(errors.password)}
                    </p>
                  )}
                </div>

                <button
                  type="submit"
                  className="w-full rounded-[8px] bg-night px-5 py-3 font-medium text-white transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-action focus-visible:ring-offset-2"
                >
                  {t("auth.signin")}
                </button>
              </form>
            ) : (
              <div>
                <fieldset>
                  <legend className="mb-3 text-sm font-medium">{t("auth.chooseType")}</legend>
                  <div className="grid gap-3 sm:grid-cols-2">
                    {ACCOUNT_TYPES.map(({ type, Icon, label, desc }) => {
                      const active = accountType === type;
                      return (
                        <label
                          key={type}
                          className={`relative cursor-pointer rounded-[10px] border p-4 transition-colors focus-within:ring-2 focus-within:ring-action ${
                            active
                              ? "border-night bg-gray-50 ring-1 ring-night"
                              : "border-gray-300 hover:border-gray-500 hover:bg-gray-50"
                          }`}
                        >
                          <input
                            type="radio"
                            name="accountType"
                            value={type}
                            checked={active}
                            onChange={() => setAccountType(type)}
                            className="sr-only"
                          />
                          {active && (
                            <span
                              className="absolute right-3 top-3 flex h-5 w-5 items-center justify-center rounded-full bg-night text-white"
                              aria-hidden="true"
                            >
                              <Check size={13} strokeWidth={3} />
                            </span>
                          )}
                          <Icon size={22} aria-hidden="true" />
                          <span className="mt-2 block font-bold">{t(label)}</span>
                          <span className="mt-0.5 block text-xs leading-snug text-gray-600">{t(desc)}</span>
                        </label>
                      );
                    })}
                  </div>
                </fieldset>
                <div className="mt-5">
                  <AccountForm key={accountType} type={accountType} />
                </div>
              </div>
            )}

            <p className="mt-8 text-center text-sm text-gray-700">
              {isLogin ? t("auth.noAccount") : t("auth.haveAccount")}{" "}
              <button type="button" onClick={toggle} className="font-medium text-link hover:underline">
                {isLogin ? t("auth.toSignup") : t("auth.toSignin")}
              </button>
            </p>
            <p className="mt-3 text-center text-xs text-gray-500">
              <Link to="/privacy" className="underline hover:text-gray-700">
                {t("auth.privacyLink")}
              </Link>
            </p>
          </div>
        </main>
      </div>
    </div>
  );
}