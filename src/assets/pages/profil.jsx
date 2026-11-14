import { useEffect } from "react";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";

// Données entièrement fictives (démonstration).
const USER = { name: "John Doe", email: "john.doe@example.com", phone: "+00 000 000 000", address: "—" };
const ORDERS = [
  { id: "DEMO-4587", date: "2025-10-15", total: 450, status: "delivered" },
  { id: "DEMO-4512", date: "2025-10-02", total: 1250, status: "progress" },
  { id: "DEMO-4468", date: "2025-09-22", total: 750, status: "cancelled" },
];

export default function Profile() {
  const { t, formatPrice, locale } = useI18n();
  const notify = useNotify();

  useEffect(() => {
    document.title = t("meta.profile");
  }, [t]);

  const date = (iso) => new Intl.DateTimeFormat(locale, { dateStyle: "medium" }).format(new Date(iso));
  const info = [
    ["profile.name", USER.name],
    ["profile.email", USER.email],
    ["profile.phone", USER.phone],
    ["profile.address", USER.address],
  ];

  return (
    <div className="mx-2.5 mt-2.5 space-y-2.5">
      <section className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="profile-title">
        <h1 id="profile-title" className="text-[40px] font-bold leading-tight">{t("profile.title")}</h1>
        <p className="mt-2 rounded-[3px] bg-info p-3 text-sm" role="note">{t("profile.demo")} {t("demo.notice")}</p>

        <h2 className="mt-6 text-[24px] font-semibold">{t("profile.info")}</h2>
        <dl className="mt-3 grid gap-4 sm:grid-cols-2">
          {info.map(([key, value]) => (
            <div key={key}>
              <dt className="text-sm text-gray-600">{t(key)}</dt>
              <dd className="font-medium">{value}</dd>
            </div>
          ))}
        </dl>
        <button type="button" onClick={() => notify(t("demo.notice"), "demo")} className="mt-5 rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">
          {t("profile.edit")}
        </button>
      </section>

      <section className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="orders-title">
        <h2 id="orders-title" className="text-[24px] font-semibold">{t("profile.orders")}</h2>
        <div className="mt-3 overflow-x-auto">
          <table className="w-full min-w-[480px] text-left">
            <thead>
              <tr className="text-sm text-gray-600">
                <th className="py-2 pr-4 font-medium">{t("profile.order")}</th>
                <th className="py-2 pr-4 font-medium">{t("profile.date")}</th>
                <th className="py-2 pr-4 font-medium">{t("profile.total")}</th>
                <th className="py-2 font-medium">{t("profile.status")}</th>
              </tr>
            </thead>
            <tbody>
              {ORDERS.map((o) => (
                <tr key={o.id} className="border-t border-gray-200">
                  <td className="py-2.5 pr-4 font-medium">{o.id}</td>
                  <td className="py-2.5 pr-4">{date(o.date)}</td>
                  <td className="py-2.5 pr-4">{formatPrice(o.total)}</td>
                  <td className="py-2.5">{t(`profile.s.${o.status}`)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
