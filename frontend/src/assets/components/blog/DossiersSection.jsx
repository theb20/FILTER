import { Link } from "react-router-dom";
import { useI18n } from "../../i18n/i18nContext.js";
import { postPath } from "./format.js";

// Dossiers techniques : section bleu nuit, 3 grandes cartes numérotées 01, 02, 03.
export default function DossiersSection({ articles }) {
  const { t } = useI18n();
  return (
    <section aria-labelledby="dossiers-title" className="mt-2.5 bg-night text-white">
      <div className="mx-auto max-w-[1500px] px-6 py-10">
        <h2 id="dossiers-title" className="text-[28px] font-bold">{t("blog.dossiers")}</h2>
        <ol className="mt-6 grid gap-4 md:grid-cols-3">
          {articles.map((a, i) => (
            <li key={a.slug} className="relative flex flex-col rounded-[8px] bg-navy p-6 transition-colors hover:bg-slate">
              <span aria-hidden="true" className="text-[48px] font-bold leading-none text-action">{String(i + 1).padStart(2, "0")}</span>
              <h3 className="mt-3 text-[22px] font-bold leading-tight">{a.title}</h3>
              <p className="mt-2 text-white/85">{a.excerpt}</p>
              <Link to={postPath(a.slug)} className="mt-auto pt-4 font-bold text-action after:absolute after:inset-0 hover:underline">
                {t("blog.dossiers.read")} <span aria-hidden="true">→</span>
                <span className="sr-only"> : {a.title}</span>
              </Link>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
