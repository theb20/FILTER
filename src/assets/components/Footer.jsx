import { Link, useLocation } from "react-router-dom";
import Logo from "./Logo.jsx";
import Todo from "./Todo.jsx";
import { SITE } from "../config/site.js";

const LINKS = [
  ["/blog", "Blog"],
  ["/terms", "Terms & Conditions"],
  ["/privacy", "Privacy Policy"],
  ["/refund", "Refund policy"],
  ["/quote", "Request a quote"],
];

function Section({ title, children }) {
  // Accordéon sur mobile (details), colonne fixe sur desktop
  return (
    <details className="group border-b border-white/20 py-3 md:border-0 md:py-0 md:[&>summary]:pointer-events-none" open>
      <summary className="flex cursor-pointer list-none items-center justify-between text-base font-bold md:cursor-default">
        {title}
        <span className="text-xl leading-none md:hidden group-open:rotate-45 transition-transform" aria-hidden="true">+</span>
      </summary>
      <div className="mt-3">{children}</div>
    </details>
  );
}

export default function Footer() {
  const { pathname } = useLocation();
  if (pathname === "/sign" || pathname === "/reset") return null;

  return (
    <footer className="mx-2.5 my-2.5 bg-slate text-white">
      <div className="mx-auto grid max-w-[1500px] gap-6 px-6 py-10 md:grid-cols-4">
        <div>
          <Logo />
          <address className="mt-4 text-sm not-italic text-white/90">
            {SITE.address || <Todo>Head office address</Todo>}
          </address>
        </div>

        <Section title="Follow us on">
          {SITE.linkedinUrl ? (
            <a href={SITE.linkedinUrl} target="_blank" rel="noopener noreferrer" className="hover:underline">
              LinkedIn
            </a>
          ) : (
            <Todo>LinkedIn</Todo>
          )}
        </Section>

        <Section title="Information">
          <ul className="space-y-2 text-sm">
            {LINKS.map(([to, label]) => (
              <li key={to}>
                <Link to={to} className="hover:underline">{label}</Link>
              </li>
            ))}
          </ul>
        </Section>

        <Section title="We accept">
          <Todo>Payment methods</Todo>
        </Section>
      </div>
      <p className="border-t border-white/20 px-6 py-4 text-center text-sm text-white/80">
        © {new Date().getFullYear()} {SITE.name} — operated by {SITE.publisher}. All rights reserved.
      </p>
    </footer>
  );
}
