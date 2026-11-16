import { Link } from "react-router-dom";
import { FaFacebookF, FaLinkedinIn, FaWhatsapp, FaYoutube } from "react-icons/fa";
import { SITE } from "../../config/site.js";
import { useI18n } from "../../i18n/i18nContext.js";
import { postPath } from "./format.js";

const waLink = (text) => `https://wa.me/${SITE.phone.replace(/\D/g, "")}?text=${encodeURIComponent(text)}`;

const SOCIALS = [
  { key: "linkedin", label: "LinkedIn", Icon: FaLinkedinIn, href: SITE.linkedinUrl, color: "bg-[#0A66C2]" },
  { key: "whatsapp", label: "WhatsApp", Icon: FaWhatsapp, href: waLink(""), color: "bg-[#25D366]" },
  { key: "facebook", label: "Facebook", Icon: FaFacebookF, href: SITE.facebookUrl, color: "bg-[#1877F2]" },
  { key: "youtube", label: "YouTube", Icon: FaYoutube, href: SITE.youtubeUrl, color: "bg-[#FF0000]" },
];

const Heading = ({ id, children }) => (
  <h2 id={id} className="border-b border-gray-200 pb-2 text-[16px] font-bold uppercase tracking-wide">
    <span className="-mb-[9px] inline-block border-b-[3px] border-action pb-2">{children}</span>
  </h2>
);

// Colonne latérale : « Restez connecté », « Les plus lus », « Besoin d'un conseil ? ».
export default function Sidebar({ popular }) {
  const { t } = useI18n();
  return (
    <aside className="space-y-4" aria-label={t("blog.sidebar")}>
      <section aria-labelledby="connect-title" className="rounded-[8px] bg-white p-5 shadow-card">
        <Heading id="connect-title">{t("blog.connect")}</Heading>
        <ul className="mt-4 flex justify-between gap-2">
          {SOCIALS.map(({ key, label, Icon, href, color }) => (
            <li key={key} className="text-center">
              {href ? (
                <a href={href} target="_blank" rel="noopener noreferrer" aria-label={label} className={`mx-auto flex h-11 w-11 items-center justify-center rounded-full text-white transition-opacity hover:opacity-85 ${color}`}>
                  <Icon size={18} aria-hidden="true" />
                </a>
              ) : (
                <span title={t("blog.social.todo")} aria-label={`${label} — ${t("blog.social.todo")}`} role="img" className="mx-auto flex h-11 w-11 items-center justify-center rounded-full bg-gray-200 text-gray-500">
                  <Icon size={18} aria-hidden="true" />
                </span>
              )}
              <span className="mt-1 block text-xs text-gray-700">{label}</span>
            </li>
          ))}
        </ul>
      </section>

      <section aria-labelledby="popular-title" className="rounded-[8px] bg-white p-5 shadow-card">
        <Heading id="popular-title">{t("blog.popular")}</Heading>
        <ol className="mt-3 divide-y divide-gray-100">
          {popular.map((a, i) => (
            <li key={a.slug} className="flex items-start gap-3 py-3">
              <span aria-hidden="true" className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-action text-sm font-bold text-black">{i + 1}</span>
              <Link to={postPath(a.slug)} className="font-medium leading-snug hover:underline">{a.title}</Link>
            </li>
          ))}
        </ol>
      </section>

      <section aria-labelledby="advice-title" className="rounded-[8px] bg-navy p-5 text-white shadow-card">
        <h2 id="advice-title" className="text-[20px] font-bold">{t("blog.advice.title")}</h2>
        <p className="mt-1 text-white/90">{t("blog.advice.text")}</p>
        <a href={waLink(t("blog.advice.message"))} target="_blank" rel="noopener noreferrer" className="mt-4 inline-flex items-center gap-2 rounded-[4px] bg-action px-5 py-2.5 font-bold text-black transition-colors hover:bg-action-dark">
          <FaWhatsapp size={18} aria-hidden="true" /> {t("blog.advice.cta")}
        </a>
      </section>
    </aside>
  );
}
