import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";
import { useNotify } from "../../configurations/notificationContext.js";
import { useAuth } from "../../store/authStore.js";
import { refreshMe } from "../../lib/account.js";
import { btnGhost, btnPrimary, input, label } from "./ui.js";

const PHONE = /^\+[1-9][0-9]{6,14}$/;

// Modification du profil : nom, téléphone et, pour un compte professionnel, la fonction. Les données légales de
// l'entreprise (raison sociale, n° fiscal) ne se changent pas ici : leur modification relance la vérification.
export default function EditProfileDialog({ user, onClose }) {
  const { t } = useI18n();
  const notify = useNotify();
  const call = useAuth((s) => s.call);
  const [form, setForm] = useState({ firstName: user.firstName ?? "", lastName: user.lastName ?? "", phone: user.phone ?? "", jobTitle: user.company?.jobTitle ?? "" });
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const first = useRef(null);

  useEffect(() => {
    first.current?.focus();
    const onKey = (e) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => {
      window.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [onClose]);

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));

  const save = async (e) => {
    e.preventDefault();
    setError("");
    const phone = form.phone.replace(/[\s.-]/g, "");
    if (phone && !PHONE.test(phone)) return setError(t("profile.form.errPhone"));
    if (!form.firstName.trim() || !form.lastName.trim()) return setError(t("profile.form.errName"));
    setSaving(true);
    const body = { firstName: form.firstName.trim(), lastName: form.lastName.trim(), ...(phone ? { phone } : {}) };
    let r = await call("/me", { method: "PATCH", body });
    const c = user.company;
    if (r.ok && c && form.jobTitle.trim() !== (c.jobTitle ?? "")) {
      r = await call("/me/company", { method: "PUT", body: { legalName: c.legalName, taxId: c.taxId, country: c.country, jobTitle: form.jobTitle.trim() } });
    }
    setSaving(false);
    if (!r.ok) return setError(r.status === 400 || r.status === 422 ? t("profile.form.errInvalid") : t("profile.err.generic"));
    await refreshMe();
    notify(t("profile.form.saved"), "success");
    onClose();
  };

  return (
    <div className="fixed inset-0 z-[1100] flex items-end justify-center bg-black/50 p-0 sm:items-center sm:p-4" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <form onSubmit={save} role="dialog" aria-modal="true" aria-labelledby="edit-profile-title" className="max-h-[92vh] w-full overflow-y-auto rounded-t-[14px] bg-white shadow-xl sm:max-w-lg sm:rounded-[14px]">
        <div className="flex items-center justify-between border-b border-gray-200 px-5 py-4">
          <h2 id="edit-profile-title" className="text-[19px] font-bold text-night">{t("profile.edit")}</h2>
          <button type="button" onClick={onClose} className="rounded-full p-1.5 text-gray-500 hover:bg-gray-100" aria-label={t("profile.form.close")}><X size={20} aria-hidden="true" /></button>
        </div>
        <div className="space-y-4 px-5 py-5">
          {error && <p role="alert" className="rounded-[8px] bg-red-50 px-3 py-2 text-sm text-red-800">{error}</p>}
          <div className="grid gap-4 sm:grid-cols-2">
            <label className={label}>{t("auth.firstName")}<input ref={first} value={form.firstName} onChange={set("firstName")} maxLength={100} autoComplete="given-name" required className={input} /></label>
            <label className={label}>{t("auth.lastName")}<input value={form.lastName} onChange={set("lastName")} maxLength={100} autoComplete="family-name" required className={input} /></label>
          </div>
          <label className={label}>
            {t("auth.phone")}
            <input value={form.phone} onChange={set("phone")} inputMode="tel" autoComplete="tel" placeholder="+225 07 00 00 00 00" className={input} />
            <span className="mt-1 block text-xs font-normal text-gray-500">{t("profile.form.phoneHint")}</span>
          </label>
          {user.company && (
            <label className={label}>
              {t("auth.jobTitle")}
              <input value={form.jobTitle} onChange={set("jobTitle")} maxLength={100} autoComplete="organization-title" className={input} />
              <span className="mt-1 block text-xs font-normal text-gray-500">{t("profile.form.companyHint")}</span>
            </label>
          )}
          <p className="text-xs text-gray-500">{t("profile.form.emailHint")}</p>
        </div>
        <div className="flex justify-end gap-2.5 border-t border-gray-200 px-5 py-4">
          <button type="button" onClick={onClose} className={btnGhost}>{t("profile.form.cancel")}</button>
          <button type="submit" disabled={saving} className={btnPrimary}>{saving ? t("profile.form.saving") : t("profile.form.save")}</button>
        </div>
      </form>
    </div>
  );
}
