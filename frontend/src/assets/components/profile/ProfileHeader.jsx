import { useRef, useState } from "react";
import { BadgeCheck, Building2, Camera, Clock, Loader2, MailWarning, Pencil, Trash2 } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";
import { useNotify } from "../../configurations/notificationContext.js";
import { deleteMyImage, uploadMyImage, useMyImage } from "../../lib/account.js";
import LineArt from "../LineArt.jsx";
import Avatar from "./Avatar.jsx";
import { badge, btnGhost, btnPrimary, fmtDate } from "./ui.js";

// En-tête du profil, à la manière d'un réseau social : couverture, photo, nom, fonction et entreprise.
export default function ProfileHeader({ user, onEdit }) {
  const { t, lang } = useI18n();
  const notify = useNotify();
  const cover = useMyImage("cover");
  const [busy, setBusy] = useState(/** @type {null | "avatar" | "cover"} */ (null));
  const avatarInput = useRef(null);
  const coverInput = useRef(null);

  const send = async (kind, file) => {
    if (!file) return;
    setBusy(kind);
    try {
      const r = await uploadMyImage(kind, file);
      notify(r.ok ? t(`profile.img.saved.${kind}`) : t("profile.img.errInvalid"), r.ok ? "success" : "error");
    } catch (e) {
      notify(t(e?.message === "size" ? "profile.img.errSize" : "profile.img.errType"), "error");
    } finally {
      setBusy(null);
    }
  };
  const remove = async (kind) => {
    setBusy(kind);
    const r = await deleteMyImage(kind);
    setBusy(null);
    notify(r.ok ? t(`profile.img.removed.${kind}`) : t("profile.err.generic"), r.ok ? "success" : "error");
  };

  const fullName = [user.firstName, user.lastName].filter(Boolean).join(" ") || user.email;
  const company = user.company;
  const sub = [company?.jobTitle, company?.legalName].filter(Boolean).join(" · ");
  const camBtn = "inline-flex items-center gap-1.5 rounded-[8px] bg-white/90 px-3 py-2 text-sm font-semibold text-night shadow-sm backdrop-blur transition hover:bg-white disabled:opacity-60";

  return (
    <header className="overflow-hidden rounded-[10px] bg-white shadow-card" data-guide="profile-info">
      <div className="relative h-[150px] bg-navy sm:h-[210px] md:h-[260px]">
        {cover ? (
          <img src={cover} alt="" className="absolute inset-0 h-full w-full object-cover" />
        ) : (
          <div className="absolute inset-0 bg-gradient-to-br from-navy via-night to-slate" aria-hidden="true">
            <LineArt className="absolute -right-10 top-0 h-full opacity-15" />
          </div>
        )}
        <div className="absolute bottom-3 right-3 flex gap-2">
          {cover && (
            <button type="button" onClick={() => remove("cover")} disabled={busy !== null} className={camBtn} aria-label={t("profile.img.removeCover")} title={t("profile.img.removeCover")}>
              <Trash2 size={15} aria-hidden="true" />
            </button>
          )}
          <button type="button" onClick={() => coverInput.current?.click()} disabled={busy !== null} className={camBtn}>
            {busy === "cover" ? <Loader2 size={15} className="animate-spin" aria-hidden="true" /> : <Camera size={15} aria-hidden="true" />}
            <span className="hidden sm:inline">{cover ? t("profile.img.changeCover") : t("profile.img.addCover")}</span>
          </button>
        </div>
        <input ref={coverInput} type="file" accept="image/png,image/jpeg,image/webp" hidden onChange={(e) => { send("cover", e.target.files?.[0]); e.target.value = ""; }} />
      </div>

      <div className="px-5 pb-5 md:px-8">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div className="flex flex-col items-center gap-3 sm:flex-row sm:items-end sm:gap-5">
            <div className="relative -mt-14 sm:-mt-16">
              <Avatar user={user} size={132} className="ring-4 ring-white" />
              <button
                type="button"
                onClick={() => avatarInput.current?.click()}
                disabled={busy !== null}
                className="absolute bottom-1.5 right-1.5 flex h-9 w-9 items-center justify-center rounded-full bg-gray-100 text-night ring-2 ring-white transition hover:bg-gray-200 disabled:opacity-60"
                aria-label={user.avatarUpdatedAt ? t("profile.img.changeAvatar") : t("profile.img.addAvatar")}
                title={user.avatarUpdatedAt ? t("profile.img.changeAvatar") : t("profile.img.addAvatar")}
              >
                {busy === "avatar" ? <Loader2 size={17} className="animate-spin" aria-hidden="true" /> : <Camera size={17} aria-hidden="true" />}
              </button>
              <input ref={avatarInput} type="file" accept="image/png,image/jpeg,image/webp" hidden onChange={(e) => { send("avatar", e.target.files?.[0]); e.target.value = ""; }} />
            </div>
            <div className="pb-1 text-center sm:text-left">
              <h1 id="profile-title" className="text-[28px] font-bold leading-tight text-night md:text-[32px]">{fullName}</h1>
              {sub && <p className="mt-0.5 flex items-center justify-center gap-1.5 text-[15px] text-gray-600 sm:justify-start"><Building2 size={15} aria-hidden="true" /> {sub}</p>}
              <div className="mt-2 flex flex-wrap items-center justify-center gap-2 sm:justify-start">
                {user.emailVerified
                  ? <span className={`${badge} bg-green-50 text-green-800`}><BadgeCheck size={13} aria-hidden="true" /> {t("profile.badge.verified")}</span>
                  : <span className={`${badge} bg-amber-50 text-amber-800`}><MailWarning size={13} aria-hidden="true" /> {t("profile.badge.unverified")}</span>}
                {company && (
                  <span className={`${badge} ${company.status === "approved" ? "bg-blue-50 text-blue-800" : company.status === "rejected" ? "bg-red-50 text-red-800" : "bg-gray-100 text-gray-700"}`}>
                    {t(`profile.badge.company.${company.status}`)}
                  </span>
                )}
                <span className={`${badge} bg-gray-100 text-gray-700`}><Clock size={13} aria-hidden="true" /> {t("profile.badge.since", { date: fmtDate(lang, user.createdAt, { month: "long", year: "numeric" }) })}</span>
              </div>
            </div>
          </div>
          <div className="flex justify-center gap-2.5 pb-1">
            <button type="button" onClick={onEdit} className={btnPrimary}><Pencil size={16} aria-hidden="true" /> {t("profile.edit")}</button>
            {user.avatarUpdatedAt && (
              <button type="button" onClick={() => remove("avatar")} disabled={busy !== null} className={btnGhost} title={t("profile.img.removeAvatar")}>
                <Trash2 size={16} aria-hidden="true" /> <span className="hidden sm:inline">{t("profile.img.removeAvatarShort")}</span><span className="sr-only sm:hidden">{t("profile.img.removeAvatar")}</span>
              </button>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
