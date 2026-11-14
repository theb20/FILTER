import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";

const field =
  "w-full rounded-[3px] border bg-white px-4 py-2.5 outline-none focus:ring-2 focus:ring-action";

export default function Sign() {
  const { t } = useI18n();
  const notify = useNotify();
  const [isLogin, setIsLogin] = useState(true);
  const [data, setData] = useState({ name: "", email: "", password: "" });
  const [errors, setErrors] = useState({});

  useEffect(() => {
    document.title = t("meta.sign");
  }, [t]);

  const toggle = () => {
    setIsLogin((v) => !v);
    setErrors({});
    setData({ name: "", email: "", password: "" });
  };

  const validate = () => {
    const e = {};
    if (!isLogin && !data.name.trim()) e.name = "auth.err.name";
    if (!data.email.trim()) e.email = "auth.err.email";
    else if (!/\S+@\S+\.\S+/.test(data.email)) e.email = "auth.err.emailInvalid";
    if (!data.password) e.password = "auth.err.password";
    else if (data.password.length < 8) e.password = "auth.err.passwordShort";
    return e;
  };

  const onSubmit = (ev) => {
    ev.preventDefault();
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length) return;
    // Aucune authentification réelle : on ne simule pas de succès.
    notify(t("demo.notice"), "demo");
  };

  const onChange = (ev) => {
    const { name, value } = ev.target;
    setData((d) => ({ ...d, [name]: value }));
    if (errors[name]) setErrors((x) => ({ ...x, [name]: undefined }));
  };

  const Field = ({ name, type = "text", label, autoComplete }) => (
    <div>
      <label htmlFor={name} className="mb-1 block text-sm font-medium">{label}</label>
      <input
        id={name}
        name={name}
        type={type}
        value={data[name]}
        onChange={onChange}
        autoComplete={autoComplete}
        aria-invalid={Boolean(errors[name])}
        aria-describedby={errors[name] ? `${name}-err` : undefined}
        className={`${field} ${errors[name] ? "border-red-500" : "border-gray-300"}`}
      />
      {errors[name] && <p id={`${name}-err`} className="mt-1 text-sm text-red-600">{t(errors[name])}</p>}
    </div>
  );

  return (
    <div className="mx-2.5 mt-2.5 flex justify-center py-10">
      <section className="w-full max-w-md rounded-[4px] bg-white p-8 shadow-card" aria-labelledby="sign-title">
        <h1 id="sign-title" className="text-[40px] font-bold leading-tight">
          {isLogin ? t("auth.signin") : t("auth.signup")}
        </h1>
        <p className="mt-2 text-gray-700">{isLogin ? t("auth.lead.signin") : t("auth.lead.signup")}</p>
        <p className="mt-4 rounded-[3px] bg-info p-3 text-sm" role="note">{t("demo.notice")}</p>

        <form onSubmit={onSubmit} noValidate className="mt-5 space-y-4">
          {!isLogin && <Field name="name" label={t("auth.name")} autoComplete="name" />}
          <Field name="email" type="email" label={t("auth.email")} autoComplete="email" />
          <Field
            name="password"
            type="password"
            label={isLogin ? t("auth.password") : t("auth.newPassword")}
            autoComplete={isLogin ? "current-password" : "new-password"}
          />

          {isLogin && (
            <p className="text-right text-sm">
              <Link to="/reset" className="text-link hover:underline">{t("auth.forgot")}</Link>
            </p>
          )}

          <button type="submit" className="w-full rounded-[3px] bg-action px-5 py-3 font-bold text-black hover:bg-action-dark">
            {isLogin ? t("auth.signin") : t("auth.signup")} <span aria-hidden="true">→</span>
          </button>

          <p className="text-center text-sm text-gray-500">{t("auth.or")}</p>

          <button
            type="button"
            onClick={() => notify(t("demo.notice"), "demo")}
            className="w-full rounded-[3px] border border-gray-300 bg-white px-5 py-3 font-medium hover:bg-gray-50"
          >
            {t("auth.google")}
          </button>
        </form>

        <p className="mt-6 text-center text-sm">
          {isLogin ? t("auth.noAccount") : t("auth.haveAccount")}{" "}
          <button type="button" onClick={toggle} className="font-medium text-link hover:underline">
            {isLogin ? t("auth.toSignup") : t("auth.toSignin")}
          </button>
        </p>
      </section>
    </div>
  );
}
