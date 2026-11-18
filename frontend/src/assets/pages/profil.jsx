import { useEffect, useMemo, useState } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { Building2, ChevronRight, FileText, LayoutGrid, LogOut, Mail, MapPin, Package, Phone, Settings, Shield, ShieldCheck, UserRound } from "lucide-react";
import { useAuth } from "../store/authStore.js";
import { useI18n } from "../i18n/i18nContext.js";
import { fetchLoginActivity, fetchSessions, placeLabel } from "../lib/account.js";
import GuideSettings from "../guide/GuideSettings.jsx";
import ProfileHeader from "../components/profile/ProfileHeader.jsx";
import EditProfileDialog from "../components/profile/EditProfileDialog.jsx";
import SecurityPanel from "../components/profile/SecurityPanel.jsx";
import PrivacyPanel from "../components/profile/PrivacyPanel.jsx";
import QuotesPanel, { useQuoteDraft, useQuoteRequests } from "../components/profile/QuotesPanel.jsx";
import { btnGhost, card, cardTitle, fmtDate, fmtRelative } from "../components/profile/ui.js";

// Onglets, adressés par l'ancre de l'URL (#securite…) : un lien peut ouvrir directement le bon onglet.
const TABS = [
  { id: "apercu", key: "profile.tab.overview", Icon: LayoutGrid },
  { id: "devis", key: "profile.tab.quotes", Icon: FileText },
  { id: "commandes", key: "profile.tab.orders", Icon: Package },
  { id: "securite", key: "profile.tab.security", Icon: ShieldCheck },
  { id: "confidentialite", key: "profile.tab.privacy", Icon: Shield },
  { id: "reglages", key: "profile.tab.settings", Icon: Settings },
];
const ALIASES = { demandes: "devis" }; // anciennes ancres

export default function Profile() {
  const { t } = useI18n();
  const { status, user, logout } = useAuth();
  const { hash } = useLocation();
  const navigate = useNavigate();
  const [editing, setEditing] = useState(false);
  const raw = hash.replace("#", "");
  const tab = TABS.some((x) => x.id === (ALIASES[raw] ?? raw)) ? (ALIASES[raw] ?? raw) : "apercu";

  useEffect(() => {
    document.title = t("meta.profile");
  }, [t]);

  if (status === "unknown") return <div className="flex min-h-[50vh] items-center justify-center text-gray-500" role="status">{t("loading")}</div>;
  if (status === "anonymous") return <Navigate to="/sign" replace />;
  const fullName = [user.firstName, user.lastName].filter(Boolean).join(" ");
  const go = (id) => navigate({ hash: id === "apercu" ? "" : id }, { replace: true });

  return (
    <div className="mx-auto mt-2.5 w-full max-w-[1180px] space-y-3 px-2.5 pb-10" aria-labelledby="profile-title">
      <ProfileHeader user={user} onEdit={() => setEditing(true)} />

      <nav className="overflow-x-auto rounded-[10px] bg-white px-2 shadow-card" aria-label={t("profile.tab.aria")}>
        <ul className="flex min-w-max" role="tablist">
          {TABS.map(({ id, key, Icon }) => (
            <li key={id} role="presentation">
              <button type="button" role="tab" aria-selected={tab === id} aria-controls={`panel-${id}`} id={`tab-${id}`} onClick={() => go(id)}
                className={`relative flex items-center gap-2 px-4 py-3.5 text-[15px] font-semibold transition ${tab === id ? "text-night" : "text-gray-500 hover:text-night"}`}>
                <Icon size={17} aria-hidden="true" /> {t(key)}
                {tab === id && <span className="absolute inset-x-3 bottom-0 h-[3px] rounded-t bg-action" aria-hidden="true" />}
              </button>
            </li>
          ))}
        </ul>
      </nav>

      <div id={`panel-${tab}`} role="tabpanel" aria-labelledby={`tab-${tab}`}>
        {tab === "apercu" && <Overview user={user} go={go} />}
        {tab === "devis" && <QuotesPanel user={user} fullName={fullName} />}
        {tab === "commandes" && (
          <section className={card} data-guide="profile-orders" aria-labelledby="orders-title">
            <h2 id="orders-title" className={cardTitle}><Package size={20} aria-hidden="true" /> {t("profile.orders")}</h2>
            <p className="mt-3 text-gray-700">{t("profile.noOrders")}</p>
            <Link to="/category" className={`${btnGhost} mt-4`}>{t("profile.ov.browse")}</Link>
          </section>
        )}
        {tab === "securite" && <SecurityPanel user={user} />}
        {tab === "confidentialite" && <PrivacyPanel />}
        {tab === "reglages" && (
          <div className="space-y-4">
            <section className={card} data-guide="profile-guide" aria-labelledby="guide-settings-title">
              <h2 id="guide-settings-title" className={`${cardTitle} mb-4`}>{t("guide.settings.title")}</h2>
              <GuideSettings />
            </section>
            <section className={card}>
              <h2 className={cardTitle}><LogOut size={20} aria-hidden="true" /> {t("profile.set.signoutTitle")}</h2>
              <p className="mt-1 text-sm text-gray-600">{t("profile.set.signoutLead")}</p>
              <button type="button" onClick={logout} className={`${btnGhost} mt-4`}>{t("nav.signout")}</button>
            </section>
          </div>
        )}
      </div>

      {editing && <EditProfileDialog user={user} onClose={() => setEditing(false)} />}
    </div>
  );
}

// Aperçu : coordonnées, entreprise, résumé de sécurité et de devis, chacun menant à son onglet.
function Overview({ user, go }) {
  const { t, lang } = useI18n();
  const [draft] = useQuoteDraft();
  const requests = useQuoteRequests();
  const [devices, setDevices] = useState(null);
  const [last, setLast] = useState(undefined);
  useEffect(() => {
    fetchSessions().then((r) => setDevices(r.ok ? r.data.items.length : null));
    fetchLoginActivity(1).then((r) => setLast(r.ok ? (r.data.items[0] ?? null) : null));
  }, []);
  const open = useMemo(() => (Array.isArray(requests) ? requests.filter((q) => q.status !== "closed").length : 0), [requests]);
  const c = user.company;
  const row = (Icon, text, muted = false) => (
    <li className="flex items-start gap-3 py-2"><Icon size={18} className="mt-0.5 shrink-0 text-gray-500" aria-hidden="true" /><span className={muted ? "text-gray-500" : "text-night"}>{text}</span></li>
  );

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.35fr)]">
      <div className="space-y-4">
        <section className={card} aria-labelledby="about-title">
          <h2 id="about-title" className={cardTitle}><UserRound size={20} aria-hidden="true" /> {t("profile.ov.about")}</h2>
          <ul className="mt-2">
            {row(Mail, user.email)}
            {row(Phone, user.phone || t("profile.ov.noPhone"), !user.phone)}
            {c && row(Building2, [c.jobTitle, c.legalName].filter(Boolean).join(" · "))}
            {c && row(FileText, `${t("auth.taxId")} : ${c.taxId}`)}
          </ul>
          {c?.status === "rejected" && c.rejectionReason && <p className="mt-2 rounded-[8px] bg-red-50 px-3 py-2 text-sm text-red-800">{t("profile.ov.rejected", { reason: c.rejectionReason })}</p>}
          {c?.status === "pending" && <p className="mt-2 rounded-[8px] bg-gray-50 px-3 py-2 text-sm text-gray-700">{t("profile.ov.pending")}</p>}
        </section>
      </div>
      <div className="space-y-4">
        <button type="button" onClick={() => go("securite")} className={`${card} flex w-full items-center gap-4 text-left transition hover:shadow-md`}>
          <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-green-50 text-green-700"><ShieldCheck size={22} aria-hidden="true" /></span>
          <span className="min-w-0 flex-1">
            <span className="block text-[17px] font-semibold text-night">{t("profile.ov.security")}</span>
            <span className="block text-sm text-gray-600">
              {devices === null ? "—" : t("profile.ov.devices", { n: devices })}
              {last && <> · {t("profile.ov.lastLogin", { when: fmtRelative(lang, last.at) })}{placeLabel(last.location) && <> · <MapPin size={13} className="mb-0.5 inline" aria-hidden="true" /> {placeLabel(last.location)}</>}</>}
            </span>
          </span>
          <ChevronRight size={20} className="text-gray-400" aria-hidden="true" />
        </button>
        <button type="button" onClick={() => go("devis")} className={`${card} flex w-full items-center gap-4 text-left transition hover:shadow-md`}>
          <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-action/20 text-night"><FileText size={22} aria-hidden="true" /></span>
          <span className="min-w-0 flex-1">
            <span className="block text-[17px] font-semibold text-night">{t("profile.ov.quotes")}</span>
            <span className="block text-sm text-gray-600">
              {requests === undefined ? t("loading") : requests === "unverified" ? t("profile.ov.quotesUnverified") : t("profile.ov.quotesOpen", { n: open })}
              {draft && <> · {t("profile.ov.draft", { date: fmtDate(lang, draft.updatedAt) })}</>}
            </span>
          </span>
          <ChevronRight size={20} className="text-gray-400" aria-hidden="true" />
        </button>
        <button type="button" onClick={() => go("confidentialite")} className={`${card} flex w-full items-center gap-4 text-left transition hover:shadow-md`}>
          <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-blue-50 text-blue-800"><Shield size={22} aria-hidden="true" /></span>
          <span className="min-w-0 flex-1">
            <span className="block text-[17px] font-semibold text-night">{t("profile.ov.privacy")}</span>
            <span className="block text-sm text-gray-600">{t("profile.ov.privacyLead")}</span>
          </span>
          <ChevronRight size={20} className="text-gray-400" aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}
