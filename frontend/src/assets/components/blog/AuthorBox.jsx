import { useI18n } from "../../i18n/i18nContext.js";

const initials = (name) => name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0].toUpperCase()).join("");

// « À propos de l'auteur » : la biographie se renseigne au backoffice (Auteurs).
export default function AuthorBox({ author, bio }) {
  const { t } = useI18n();
  return (
    <aside aria-label={t("blog.author")} className="flex gap-4 rounded-[8px] bg-gray-50 p-5">
      <span aria-hidden="true" className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-night text-lg font-bold text-action">{initials(author)}</span>
      <div>
        <p className="text-xs font-bold uppercase tracking-wide text-gray-500">{t("blog.author")}</p>
        <p className="text-[18px] font-bold">{author}</p>
        {bio && <p className="mt-1 text-[15px] text-gray-700">{bio}</p>}
      </div>
    </aside>
  );
}
