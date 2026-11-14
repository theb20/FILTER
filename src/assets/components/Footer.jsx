import { Link } from "react-router-dom";
import Logo from "./Logo.jsx";
import Todo from "./Todo.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { SITE } from "../config/site.js";

const LINKS = [
  ["/blog", "footer.blog"],
  ["/terms", "footer.terms"],
  ["/privacy", "footer.privacy"],
  ["/refund", "footer.refund"],
  ["/quote", "footer.quote"],
];

function Section({ title, children }) {
  // Accordéon sur mobile (details), colonne fixe sur desktop
  return (
    <details className="group border-b border-white/20 py-3 md:border-0 md:py-0 md:[&>summary]:pointer-events-none" open>
      <summary className="flex cursor-pointer list-none items-center justify-between text-base font-bold md:cursor-default">
        {title}
        <span className="text-xl leading-none transition-transform group-open:rotate-45 md:hidden" aria-hidden="true">+</span>
      </summary>
      <div className="mt-3">{children}</div>
    </details>
  );
}

export default function Footer() {
  const { t } = useI18n();

  return (
    <footer className="mx-2.5 my-2.5 bg-slate text-white">
      <div className="mx-auto grid max-w-[1500px] gap-6 px-6 py-10 md:grid-cols-4">
        <div>
          <Logo />
          <address className="mt-4 text-sm not-italic text-white/90">
            {SITE.address || <Todo>{t("todo.address")}</Todo>}
          </address>
        </div>

        <Section title={t("footer.follow")}>
          {SITE.linkedinUrl ? (
            <a href={SITE.linkedinUrl} target="_blank" rel="noopener noreferrer" className="hover:underline">
              LinkedIn
            </a>
          ) : (
            <Todo>{t("todo.linkedin")}</Todo>
          )}
        </Section>

        <Section title={t("footer.info")}>
          <ul className="space-y-2 text-sm">
            {LINKS.map(([to, key]) => (
              <li key={to}>
                <Link to={to} className="hover:underline">{t(key)}</Link>
              </li>
            ))}
          </ul>
        </Section>

        <Section title={t("footer.accept")}>
          <Todo>{t("todo.payments")}</Todo>
        </Section>
      </div>
      <p className="border-t border-white/20 px-6 py-4 text-center text-sm text-white/80">
        {t("footer.rights", { year: new Date().getFullYear(), name: SITE.name, publisher: SITE.publisher })}
      </p>
    </footer>
  );
}
