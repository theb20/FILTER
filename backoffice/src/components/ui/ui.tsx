import Link from "next/link";
import { ChevronLeft, ChevronRight, ChevronRight as CrumbIcon, Inbox, Search } from "lucide-react";

/* ───────── En-tête de page ───────── */
export type Crumb = { label: string; href?: string };

export function PageHeader({ title, subtitle, action, breadcrumb }: { title: string; subtitle?: string; action?: React.ReactNode; breadcrumb?: Crumb[] }) {
  return (
    <div data-guide="page-header" className="mb-6">
      {breadcrumb && breadcrumb.length > 0 && (
        <nav aria-label="Fil d'Ariane" data-guide="breadcrumb" className="mb-2 flex flex-wrap items-center gap-1 text-xs text-slate">
          {breadcrumb.map((c, i) => (
            <span key={c.label} className="flex items-center gap-1">
              {i > 0 && <CrumbIcon className="h-3 w-3 text-slate/40" aria-hidden />}
              {c.href ? <Link href={c.href} className="hover:text-night hover:underline">{c.label}</Link> : <span aria-current="page" className="text-night">{c.label}</span>}
            </span>
          ))}
        </nav>
      )}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold tracking-tight text-night">{title}</h1>
          {subtitle && <p className="mt-1 max-w-2xl text-sm text-slate">{subtitle}</p>}
        </div>
        {action && <div data-guide="page-action" className="flex shrink-0 items-center gap-2">{action}</div>}
      </div>
    </div>
  );
}

/* ───────── Conteneurs ───────── */
export function Card({ children, className = "", title, description, action, guide }: { children: React.ReactNode; className?: string; title?: string; description?: string; action?: React.ReactNode; guide?: string }) {
  return (
    <section data-guide={guide} className={`rounded-xl border border-line bg-white p-5 shadow-card ${className}`}>
      {(title || action) && (
        <header className="mb-4 flex items-start justify-between gap-3">
          <div>
            {title && <h2 className="text-base font-semibold text-night">{title}</h2>}
            {description && <p className="mt-0.5 text-sm text-slate">{description}</p>}
          </div>
          {action}
        </header>
      )}
      {children}
    </section>
  );
}

export function StatCard({ label, value, hint, href, icon, tone = "default" }: { label: string; value: React.ReactNode; hint?: string; href?: string; icon?: React.ReactNode; tone?: "default" | "alert" }) {
  const body = (
    <>
      <div className="flex items-start justify-between">
        <p className="text-sm font-medium text-slate">{label}</p>
        {icon && <span className={`grid h-9 w-9 place-items-center rounded-lg ${tone === "alert" ? "bg-action/25 text-[#7a5a00]" : "bg-night/5 text-night"}`}>{icon}</span>}
      </div>
      <p className="mt-3 text-3xl font-bold tabular-nums tracking-tight text-night">{value}</p>
      {hint && <p className="mt-1 text-xs text-slate">{hint}</p>}
    </>
  );
  const cls = "block rounded-xl border border-line bg-white p-5 shadow-card";
  return href ? <Link href={href} className={`${cls} transition hover:-translate-y-0.5 hover:border-night/20 hover:shadow-lg`}>{body}</Link> : <div className={cls}>{body}</div>;
}

/* ───────── Badges / alertes ───────── */
const tones = {
  ok: ["bg-ok/10 text-[#007a52]", "bg-ok"],
  warn: ["bg-action/20 text-[#7a5a00]", "bg-action-dark"],
  off: ["bg-slate/10 text-slate", "bg-slate/50"],
  info: ["bg-link/10 text-link", "bg-link"],
  danger: ["bg-danger/10 text-danger", "bg-danger"],
} as const;

export function Badge({ tone = "off", children }: { tone?: keyof typeof tones; children: React.ReactNode }) {
  return (
    <span className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-medium ${tones[tone][0]}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${tones[tone][1]}`} aria-hidden />
      {children}
    </span>
  );
}

export function Notice({ kind, children }: { kind: "ok" | "error"; children: React.ReactNode }) {
  const cls = kind === "ok" ? "border-ok/30 bg-ok/5 text-[#007a52]" : "border-danger/30 bg-danger/5 text-danger";
  return <p role={kind === "error" ? "alert" : "status"} className={`mb-4 rounded-lg border px-3.5 py-2.5 text-sm ${cls}`}>{children}</p>;
}

/* ───────── Boutons ───────── */
export const btnCls = "inline-flex items-center justify-center gap-2 rounded-lg bg-action px-4 py-2 text-sm font-semibold text-night shadow-sm transition hover:bg-action-dark active:scale-[.98] disabled:cursor-not-allowed disabled:opacity-60";
export const btnGhostCls = "inline-flex items-center justify-center gap-2 rounded-lg border border-line bg-white px-4 py-2 text-sm font-medium text-night transition hover:bg-page active:scale-[.98] disabled:cursor-not-allowed disabled:opacity-60";
export const btnDangerCls = "inline-flex items-center justify-center gap-2 rounded-lg border border-danger/40 bg-white px-4 py-2 text-sm font-medium text-danger transition hover:bg-danger/5 active:scale-[.98] disabled:cursor-not-allowed disabled:opacity-60";
export const inputCls = "mt-1.5 w-full rounded-lg border border-line bg-white px-3 py-2 text-night placeholder:text-slate/40 transition focus:border-link focus:outline-none focus:ring-2 focus:ring-link/20";
export const selectCls = "rounded-lg border border-line bg-white px-3 py-2 text-sm text-night transition focus:border-link focus:outline-none focus:ring-2 focus:ring-link/20";

export function LinkButton({ href, children, variant = "primary" }: { href: string; children: React.ReactNode; variant?: "primary" | "ghost" }) {
  return <Link href={href} className={variant === "primary" ? btnCls : btnGhostCls}>{children}</Link>;
}

/* ───────── Tableaux ───────── */
export function Table({ head, children, empty, emptyHint }: { head: string[]; children: React.ReactNode; empty?: string; emptyHint?: string }) {
  return (
    <div data-guide="table" className="overflow-x-auto rounded-xl border border-line bg-white shadow-card">
      <table className="w-full min-w-[640px] text-left text-sm">
        <thead className="border-b border-line bg-page/70 text-xs uppercase tracking-wide text-slate">
          <tr>{head.map((h, i) => <th key={`${h}-${i}`} scope="col" className="whitespace-nowrap px-4 py-3 font-semibold">{h}</th>)}</tr>
        </thead>
        <tbody className="divide-y divide-line">{children}</tbody>
      </table>
      {empty && <EmptyState title={empty} hint={emptyHint} compact />}
    </div>
  );
}

export function EmptyState({ title, hint, action, compact = false }: { title: string; hint?: string; action?: React.ReactNode; compact?: boolean }) {
  return (
    <div className={`flex flex-col items-center justify-center text-center ${compact ? "px-4 py-12" : "rounded-xl border border-dashed border-line bg-white px-6 py-16"}`}>
      <span className="grid h-12 w-12 place-items-center rounded-full bg-page text-slate"><Inbox className="h-6 w-6" aria-hidden /></span>
      <p className="mt-3 font-medium text-night">{title}</p>
      {hint && <p className="mt-1 max-w-sm text-sm text-slate">{hint}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

/* ───────── Filtres & pagination ───────── */
export function FilterBar({ children, reset }: { children: React.ReactNode; reset?: { href: string; show: boolean } }) {
  return (
    <form role="search" data-guide="filters" className="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-line bg-white p-3 shadow-card">
      {children}
      <button className={btnGhostCls}>Filtrer</button>
      {reset?.show && <Link href={reset.href} className="px-2 text-sm text-link hover:underline">Réinitialiser</Link>}
    </form>
  );
}

export function SearchInput({ name = "q", defaultValue, placeholder, className = "" }: { name?: string; defaultValue?: string; placeholder: string; className?: string }) {
  return (
    <label className={`relative block ${className}`}>
      <span className="sr-only">{placeholder}</span>
      <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate/60" aria-hidden />
      <input name={name} defaultValue={defaultValue} placeholder={placeholder} className="w-full rounded-lg border border-line bg-white py-2 pl-9 pr-3 text-sm placeholder:text-slate/50 focus:border-link focus:outline-none focus:ring-2 focus:ring-link/20" />
    </label>
  );
}

export function Pagination({ page, totalPages, total, noun, href }: { page: number; totalPages: number; total?: number; noun?: string; href: (p: number) => string }) {
  if (totalPages <= 1) return null;
  const nav = "inline-flex items-center gap-1 rounded-lg border border-line bg-white px-3 py-1.5 text-sm font-medium text-night transition hover:bg-page";
  const off = "inline-flex items-center gap-1 rounded-lg border border-line px-3 py-1.5 text-sm text-slate/40";
  return (
    <nav data-guide="pagination" className="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm" aria-label="Pagination">
      <p className="text-slate">Page {page} sur {totalPages}{total !== undefined && ` · ${total} ${noun ?? "résultats"}`}</p>
      <div className="flex gap-2">
        {page > 1 ? <Link href={href(page - 1)} className={nav}><ChevronLeft className="h-4 w-4" aria-hidden />Précédent</Link> : <span className={off}><ChevronLeft className="h-4 w-4" aria-hidden />Précédent</span>}
        {page < totalPages ? <Link href={href(page + 1)} className={nav}>Suivant<ChevronRight className="h-4 w-4" aria-hidden /></Link> : <span className={off}>Suivant<ChevronRight className="h-4 w-4" aria-hidden /></span>}
      </div>
    </nav>
  );
}

/* ───────── Chargement ───────── */
export function Skeleton({ className = "" }: { className?: string }) {
  return <div aria-hidden className={`animate-pulse rounded-lg bg-slate/10 ${className}`} />;
}
