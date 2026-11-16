import { Link2 } from "lucide-react";
import { FaFacebookF, FaLinkedinIn, FaWhatsapp } from "react-icons/fa";
import { useNotify } from "../../configurations/notificationContext.js";
import { useI18n } from "../../i18n/i18nContext.js";

const btn = "flex h-9 w-9 items-center justify-center rounded-full text-white transition-opacity hover:opacity-85 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-action focus-visible:ring-offset-2";

// Partage : copier le lien, WhatsApp, LinkedIn, Facebook (ouvre le service choisi, action de l'utilisateur).
export default function ShareBar({ title }) {
  const { t } = useI18n();
  const notify = useNotify();
  const url = typeof window === "undefined" ? "" : window.location.href;
  const enc = encodeURIComponent;
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      notify(t("blog.share.copied"), "success");
    } catch {
      notify(t("blog.share.copyFail"), "error");
    }
  };
  return (
    <div>
      <p className="font-bold">{t("blog.share")}</p>
      <ul className="mt-2 flex gap-2">
        <li><button type="button" onClick={copy} aria-label={t("blog.share.copy")} className={`${btn} bg-slate`}><Link2 size={15} aria-hidden="true" /></button></li>
        <li><a href={`https://wa.me/?text=${enc(`${title} ${url}`)}`} target="_blank" rel="noopener noreferrer" aria-label="WhatsApp" className={`${btn} bg-[#25D366]`}><FaWhatsapp size={17} aria-hidden="true" /></a></li>
        <li><a href={`https://www.linkedin.com/sharing/share-offsite/?url=${enc(url)}`} target="_blank" rel="noopener noreferrer" aria-label="LinkedIn" className={`${btn} bg-[#0A66C2]`}><FaLinkedinIn size={16} aria-hidden="true" /></a></li>
        <li><a href={`https://www.facebook.com/sharer/sharer.php?u=${enc(url)}`} target="_blank" rel="noopener noreferrer" aria-label="Facebook" className={`${btn} bg-[#1877F2]`}><FaFacebookF size={15} aria-hidden="true" /></a></li>
      </ul>
    </div>
  );
}
