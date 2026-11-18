import { useEffect, useId, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Bookmark, ChevronDown, LogOut, ShieldCheck, UserRound } from "lucide-react";
import Avatar from "./profile/Avatar.jsx";
import { useAuth } from "../store/authStore.js";
import { useI18n } from "../i18n/i18nContext.js";


// Bulle du compte (en-tête) quand le client est connecté : pastille à ses initiales, prénom, menu déroulant
// (Mon compte, Mes brouillons de devis, Déconnexion). Fermeture : clic à l'extérieur, Échap, choix d'un lien.
export default function AccountMenu({ user }) {
  const { t } = useI18n();
  const logout = useAuth((a) => a.logout);
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const box = useRef(null);
  const id = useId();

  useEffect(() => {
    if (!open) return undefined;
    const onDown = (e) => { if (!box.current?.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === "Escape") { setOpen(false); box.current?.querySelector("button")?.focus(); } };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("mousedown", onDown); document.removeEventListener("keydown", onKey); };
  }, [open]);

  const name = [user.firstName, user.lastName].filter(Boolean).join(" ") || user.email;
  const item = "flex w-full items-center gap-2.5 rounded-[8px] px-3 py-2 text-left text-[14px] text-gray-800 hover:bg-gray-100";

  return (
    <div ref={box} className="relative hidden lg:block" data-guide="account">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-2 rounded-full bg-white/10 py-1 pl-1 pr-3 text-white transition-colors hover:bg-white/20"
      >
        <Avatar user={user} size={32} />
        <span className="max-w-[110px] truncate">{user.firstName || t("nav.account")}</span>
        <ChevronDown size={15} aria-hidden="true" className={`transition-transform ${open ? "rotate-180" : ""}`} />
      </button>
      {open && (
        <div id={id} role="menu" className="absolute right-0 top-[calc(100%+10px)] z-[1100] w-72 rounded-[14px] bg-white p-2 text-black shadow-2xl ring-1 ring-black/5">
          <div className="flex items-center gap-3 border-b border-gray-100 px-3 pb-3 pt-2">
            <Avatar user={user} size={44} />
            <div className="min-w-0">
              <p className="text-[11px] uppercase tracking-wide text-gray-500">{t("account.menu.signedAs")}</p>
              <p className="truncate font-semibold">{name}</p>
              <p className="truncate text-[12px] text-gray-500">{user.email}</p>
            </div>
          </div>
          <div className="py-1.5">
            <Link role="menuitem" to="/profil" onClick={() => setOpen(false)} className={item}><UserRound size={16} aria-hidden="true" /> {t("nav.myAccount")}</Link>
            <Link role="menuitem" to="/profil#devis" onClick={() => setOpen(false)} className={item}><Bookmark size={16} aria-hidden="true" /> {t("account.menu.drafts")}</Link>
            <Link role="menuitem" to="/profil#securite" onClick={() => setOpen(false)} className={item}><ShieldCheck size={16} aria-hidden="true" /> {t("account.menu.security")}</Link>
          </div>
          <button role="menuitem" type="button" onClick={async () => { setOpen(false); await logout(); navigate("/"); }} className={`${item} border-t border-gray-100 text-red-700 hover:bg-red-50`}>
            <LogOut size={16} aria-hidden="true" /> {t("nav.signout")}
          </button>
        </div>
      )}
    </div>
  );
}
