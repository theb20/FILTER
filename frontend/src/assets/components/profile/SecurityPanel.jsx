import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { KeyRound, LogOut, Mail, MapPin, Monitor, ShieldCheck, Smartphone, Tablet } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";
import { useNotify } from "../../configurations/notificationContext.js";
import { useAuth } from "../../store/authStore.js";
import { fetchLoginActivity, fetchSessions, placeLabel, revokeOtherSessions, revokeSession } from "../../lib/account.js";
import PasswordInput from "../PasswordInput.jsx";
import LoginMap from "./LoginMap.jsx";
import { badge, btnDanger, btnDark, btnGhost, card, cardTitle, fmtDate, fmtRelative, input, label } from "./ui.js";

const DeviceIcon = ({ kind, ...p }) => (kind === "mobile" ? <Smartphone {...p} /> : kind === "tablet" ? <Tablet {...p} /> : <Monitor {...p} />);

// Onglet « Sécurité » : mot de passe, adresse e-mail, appareils connectés, historique des connexions sur une carte.
export default function SecurityPanel({ user }) {
  return (
    <div className="space-y-4">
      <Devices />
      <LoginActivity />
      <div className="grid gap-4 lg:grid-cols-2">
        <PasswordCard />
        <EmailCard user={user} />
      </div>
    </div>
  );
}

function PasswordCard() {
  const { t } = useI18n();
  const notify = useNotify();
  const call = useAuth((s) => s.call);
  const [cur, setCur] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    setError("");
    if (next.length < 8) return setError(t("auth.err.passwordShort"));
    if (next !== confirm) return setError(t("auth.err.match"));
    setBusy(true);
    const r = await call("/me/password", { method: "POST", body: { currentPassword: cur, newPassword: next } });
    setBusy(false);
    if (!r.ok) {
      if (r.code === "INVALID_CREDENTIALS") return setError(t("profile.sec.pwWrong"));
      if (r.fields?.some((f) => f.code === "PASSWORD_COMPROMISED")) return setError(t("auth.err.compromised"));
      return setError(r.status === 429 ? t("auth.err.throttle") : t("profile.err.generic"));
    }
    setCur(""); setNext(""); setConfirm("");
    notify(t("profile.sec.pwChanged"), "success");
  };

  return (
    <section className={card} aria-labelledby="pw-title">
      <h2 id="pw-title" className={cardTitle}><KeyRound size={20} aria-hidden="true" /> {t("profile.sec.pwTitle")}</h2>
      <p className="mt-1 text-sm text-gray-600">{t("profile.sec.pwLead")}</p>
      <form onSubmit={submit} className="mt-4 space-y-3">
        {error && <p role="alert" className="rounded-[8px] bg-red-50 px-3 py-2 text-sm text-red-800">{error}</p>}
        <label className={label}>{t("profile.sec.pwCurrent")}<div className="mt-1.5"><PasswordInput value={cur} onChange={(e) => setCur(e.target.value)} autoComplete="current-password" required className={input.replace("mt-1.5 ", "")} /></div></label>
        <label className={label}>{t("profile.sec.pwNew")}<div className="mt-1.5"><PasswordInput value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" minLength={8} maxLength={128} required className={input.replace("mt-1.5 ", "")} /></div></label>
        <label className={label}>{t("auth.confirm")}<div className="mt-1.5"><PasswordInput value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required className={input.replace("mt-1.5 ", "")} /></div></label>
        <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
          <Link to="/reset" className="text-sm text-link hover:underline">{t("profile.sec.pwNone")}</Link>
          <button type="submit" disabled={busy} className={btnDark}>{busy ? t("profile.form.saving") : t("profile.sec.pwSubmit")}</button>
        </div>
      </form>
    </section>
  );
}

function EmailCard({ user }) {
  const { t } = useI18n();
  const call = useAuth((s) => s.call);
  const [email, setEmail] = useState("");
  const [pass, setPass] = useState("");
  const [state, setState] = useState(/** @type {"" | "sent" | "busy"} */ (""));
  const [error, setError] = useState("");

  const submit = async (e) => {
    e.preventDefault();
    setError("");
    setState("busy");
    const r = await call("/me/email", { method: "POST", body: { newEmail: email.trim(), ...(pass ? { password: pass } : {}) } });
    if (!r.ok) {
      setState("");
      if (r.code === "INVALID_CREDENTIALS") return setError(t("profile.sec.pwWrong"));
      return setError(r.status === 429 ? t("auth.err.throttle") : r.status === 400 || r.status === 422 ? t("profile.sec.emailInvalid") : t("profile.err.generic"));
    }
    setState("sent");
    setPass("");
  };

  return (
    <section className={card} aria-labelledby="email-title">
      <h2 id="email-title" className={cardTitle}><Mail size={20} aria-hidden="true" /> {t("profile.sec.emailTitle")}</h2>
      <p className="mt-1 text-sm text-gray-600">{t("profile.sec.emailCurrent")} <strong className="text-night">{user.email}</strong></p>
      {state === "sent" ? (
        <p role="status" className="mt-4 rounded-[8px] bg-green-50 px-3 py-3 text-sm text-green-900">{t("profile.sec.emailSent", { email: email.trim() })}</p>
      ) : (
        <form onSubmit={submit} className="mt-4 space-y-3">
          {error && <p role="alert" className="rounded-[8px] bg-red-50 px-3 py-2 text-sm text-red-800">{error}</p>}
          <label className={label}>{t("profile.sec.emailNew")}<input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" required className={input} /></label>
          <label className={label}>{t("profile.sec.pwCurrent")}<div className="mt-1.5"><PasswordInput value={pass} onChange={(e) => setPass(e.target.value)} autoComplete="current-password" className={input.replace("mt-1.5 ", "")} /></div></label>
          <p className="text-xs text-gray-500">{t("profile.sec.emailHint")}</p>
          <div className="flex justify-end"><button type="submit" disabled={state === "busy"} className={btnDark}>{state === "busy" ? t("profile.form.saving") : t("profile.sec.emailSubmit")}</button></div>
        </form>
      )}
    </section>
  );
}

function Devices() {
  const { t, lang } = useI18n();
  const notify = useNotify();
  const [list, setList] = useState(/** @type {any[] | null | "error"} */ (null));
  const load = useCallback(() => fetchSessions().then((r) => setList(r.ok ? r.data.items : "error")), []);
  useEffect(() => { load(); }, [load]);

  const out = async (id) => {
    const r = await revokeSession(id);
    notify(r.ok ? t("profile.sec.devOut") : t("profile.err.generic"), r.ok ? "success" : "error");
    load();
  };
  const outAll = async () => {
    if (!window.confirm(t("profile.sec.devOutAllConfirm"))) return;
    const r = await revokeOtherSessions();
    notify(r.ok ? t("profile.sec.devOutAllDone", { n: r.data.revoked }) : t("profile.err.generic"), r.ok ? "success" : "error");
    load();
  };
  const others = Array.isArray(list) ? list.filter((s) => !s.current).length : 0;

  return (
    <section className={card} aria-labelledby="dev-title">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 id="dev-title" className={cardTitle}><ShieldCheck size={20} aria-hidden="true" /> {t("profile.sec.devTitle")}</h2>
          <p className="mt-1 text-sm text-gray-600">{t("profile.sec.devLead")}</p>
        </div>
        {others > 0 && <button type="button" onClick={outAll} className={btnGhost}><LogOut size={16} aria-hidden="true" /> {t("profile.sec.devOutAll")}</button>}
      </div>
      {list === null && <p className="mt-4 text-sm text-gray-500" role="status">{t("loading")}</p>}
      {list === "error" && <p className="mt-4 text-sm text-red-700">{t("profile.err.generic")}</p>}
      {Array.isArray(list) && (
        <ul className="mt-4 divide-y divide-gray-100">
          {list.map((s) => (
            <li key={s.id} className="flex flex-wrap items-center gap-4 py-3.5">
              <span className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${s.current ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-600"}`}>
                <DeviceIcon kind={s.device.kind} size={20} aria-hidden="true" />
              </span>
              <div className="min-w-0 flex-1">
                <p className="flex flex-wrap items-center gap-2 font-semibold text-night">
                  {s.device.browser} · {s.device.os}
                  {s.current && <span className={`${badge} bg-green-50 text-green-800`}>{t("profile.sec.devCurrent")}</span>}
                </p>
                <p className="mt-0.5 text-sm text-gray-600">
                  {placeLabel(s.location) ? <><MapPin size={13} className="mb-0.5 inline" aria-hidden="true" /> {placeLabel(s.location)} · </> : null}
                  IP {s.ip || "—"} · {s.current ? t("profile.sec.devActiveNow") : t("profile.sec.devSeen", { when: fmtRelative(lang, s.lastSeenAt) })}
                </p>
                <p className="text-xs text-gray-500">{t("profile.sec.devSince", { date: fmtDate(lang, s.createdAt, { dateStyle: "medium", timeStyle: "short" }) })}</p>
              </div>
              {!s.current && <button type="button" onClick={() => out(s.id)} className={btnDanger}><LogOut size={15} aria-hidden="true" /> {t("profile.sec.devOutOne")}</button>}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function LoginActivity() {
  const { t, lang } = useI18n();
  const [items, setItems] = useState(/** @type {any[] | null} */ (null));
  const [page, setPage] = useState({ page: 0, totalPages: 0 });
  const [error, setError] = useState(false);
  const [selected, setSelected] = useState(null);

  const load = useCallback(async (p) => {
    const r = await fetchLoginActivity(p);
    if (!r.ok) return setError(true);
    setItems((prev) => (p === 1 ? r.data.items : [...(prev ?? []), ...r.data.items]));
    setPage({ page: r.data.page.page, totalPages: r.data.page.totalPages });
  }, []);
  useEffect(() => { load(1); }, [load]);

  const located = useMemo(() => (items ?? []).filter((e) => e.location), [items]);
  const current = selected ?? located[0] ?? null;

  return (
    <section className={card} aria-labelledby="act-title">
      <h2 id="act-title" className={cardTitle}><MapPin size={20} aria-hidden="true" /> {t("profile.sec.actTitle")}</h2>
      <p className="mt-1 text-sm text-gray-600">{t("profile.sec.actLead")}</p>
      {error && <p className="mt-4 text-sm text-red-700">{t("profile.err.generic")}</p>}
      {items === null && !error && <p className="mt-4 text-sm text-gray-500" role="status">{t("loading")}</p>}
      {items && items.length === 0 && <p className="mt-4 text-sm text-gray-600">{t("profile.sec.actNone")}</p>}
      {items && items.length > 0 && (
        <div className="mt-4 grid gap-5 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)]">
          <div>
            {located.length > 0
              ? <LoginMap points={located} selected={current} label={t("profile.sec.actMap", { place: placeLabel(current?.location) })} />
              : <div className="flex h-[240px] items-center justify-center rounded-[8px] bg-gray-50 px-6 text-center text-sm text-gray-500">{t("profile.sec.actNoPlace")}</div>}
            <p className="mt-2 text-xs text-gray-500">{t("profile.sec.actApprox")}</p>
          </div>
          <div>
            <ul className="max-h-[340px] divide-y divide-gray-100 overflow-y-auto pr-1">
              {items.map((e) => {
                const on = current && e.id === current.id;
                return (
                  <li key={e.id}>
                    <button type="button" onClick={() => setSelected(e)} aria-pressed={!!on} disabled={!e.location}
                      className={`flex w-full items-start gap-3 rounded-[8px] px-2 py-3 text-left transition ${on ? "bg-action/15" : "hover:bg-gray-50"} disabled:cursor-default`}>
                      <DeviceIcon kind={e.device.kind} size={18} className="mt-0.5 shrink-0 text-gray-500" aria-hidden="true" />
                      <span className="min-w-0 flex-1">
                        <span className="flex flex-wrap items-center gap-1.5 text-[15px] font-semibold text-night">
                          {placeLabel(e.location) || t("profile.sec.actUnknownPlace")}
                          {e.newDevice && <span className={`${badge} bg-amber-50 text-amber-800`}>{t("profile.sec.actNewDevice")}</span>}
                          {e.method === "google" && <span className={`${badge} bg-gray-100 text-gray-700`}>Google</span>}
                        </span>
                        <span className="block text-sm text-gray-600">{e.device.browser} · {e.device.os} · IP {e.ip || "—"}</span>
                        <span className="block text-xs text-gray-500">{fmtDate(lang, e.at, { dateStyle: "full", timeStyle: "short" })}</span>
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
            {page.page < page.totalPages && <button type="button" onClick={() => load(page.page + 1)} className={`${btnGhost} mt-3 w-full`}>{t("profile.sec.actMore")}</button>}
            <p className="mt-3 rounded-[8px] bg-gray-50 px-3 py-2.5 text-xs text-gray-600">{t("profile.sec.actNotMe")}</p>
          </div>
        </div>
      )}
    </section>
  );
}
