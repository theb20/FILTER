import { useEffect } from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "../store/authStore.js";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import GuideSettings from "../guide/GuideSettings.jsx";

export default function Profile() {
  const { t } = useI18n();
  const notify = useNotify();
  const { status, user, logout } = useAuth();

  useEffect(() => {
    document.title = t("meta.profile");
  }, [t]);

  if (status === "unknown") return <div className="flex min-h-[50vh] items-center justify-center text-gray-500" role="status">{t("loading")}</div>;
  if (status === "anonymous") return <Navigate to="/sign" replace />;
  const fullName = [user.firstName, user.lastName].filter(Boolean).join(" ") || "—";

  const info = [
    ["profile.name", fullName],
    ["profile.email", user.email],
  ];

  return (
    <div className="mx-2.5 mt-2.5 space-y-2.5">
      <section data-guide="profile-info" className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="profile-title">
        <h1 id="profile-title" className="text-[40px] font-bold leading-tight">{t("profile.title")}</h1>

        <h2 className="mt-6 text-[24px] font-semibold">{t("profile.info")}</h2>
        <dl className="mt-3 grid gap-4 sm:grid-cols-2">
          {info.map(([key, value]) => (
            <div key={key}>
              <dt className="text-sm text-gray-600">{t(key)}</dt>
              <dd className="font-medium">{value}</dd>
            </div>
          ))}
        </dl>
        <div className="mt-5 flex flex-wrap gap-3">
          <button type="button" onClick={() => notify(t("demo.notice"), "demo")} className="rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">
            {t("profile.edit")}
          </button>
          <button type="button" onClick={logout} className="rounded-[3px] border border-gray-300 px-5 py-2.5 font-medium hover:bg-gray-50">
            {t("nav.signout")}
          </button>
        </div>
      </section>

      <section data-guide="profile-orders" className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="orders-title">
        <h2 id="orders-title" className="text-[24px] font-semibold">{t("profile.orders")}</h2>
        <p className="mt-3 text-gray-700">{t("profile.noOrders")}</p>
      </section>

      <section data-guide="profile-guide" className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="guide-settings-title">
        <h2 id="guide-settings-title" className="mb-4 text-[24px] font-semibold">{t("guide.settings.title")}</h2>
        <GuideSettings />
      </section>
    </div>
  );
}
