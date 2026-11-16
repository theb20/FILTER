import Link from "next/link";

export function PageHeader({ title, subtitle, action }: { title: string; subtitle?: string; action?: React.ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 className="text-2xl font-bold text-night">{title}</h1>
        {subtitle && <p className="mt-1 text-sm text-slate">{subtitle}</p>}
      </div>
      {action}
    </div>
  );
}

export function Card({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <section className={`rounded-lg border border-line bg-white p-5 shadow-card ${className}`}>{children}</section>;
}

const tones = {
  ok: "bg-ok/10 text-[#007a52]",
  warn: "bg-action/20 text-[#7a5a00]",
  off: "bg-slate/10 text-slate",
  info: "bg-link/10 text-link",
} as const;

export function Badge({ tone = "off", children }: { tone?: keyof typeof tones; children: React.ReactNode }) {
  return <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${tones[tone]}`}>{children}</span>;
}

export function LinkButton({ href, children, variant = "primary" }: { href: string; children: React.ReactNode; variant?: "primary" | "ghost" }) {
  const cls = variant === "primary" ? "bg-action text-night hover:bg-action-dark" : "border border-line bg-white text-night hover:bg-page";
  return <Link href={href} className={`inline-block rounded px-4 py-2 text-sm font-medium transition ${cls}`}>{children}</Link>;
}

export function Notice({ kind, children }: { kind: "ok" | "error"; children: React.ReactNode }) {
  const cls = kind === "ok" ? "border-ok/30 bg-ok/5 text-[#007a52]" : "border-danger/30 bg-danger/5 text-danger";
  return <p role={kind === "error" ? "alert" : "status"} className={`mb-4 rounded border px-3 py-2 text-sm ${cls}`}>{children}</p>;
}

export function Table({ head, children, empty }: { head: string[]; children: React.ReactNode; empty?: string }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-line bg-white shadow-card">
      <table className="w-full min-w-[640px] text-left text-sm">
        <thead className="border-b border-line bg-page text-xs uppercase tracking-wide text-slate">
          <tr>{head.map((h) => <th key={h} className="px-4 py-3 font-medium">{h}</th>)}</tr>
        </thead>
        <tbody className="divide-y divide-line">{children}</tbody>
      </table>
      {empty && <p className="px-4 py-8 text-center text-sm text-slate">{empty}</p>}
    </div>
  );
}

export const inputCls = "mt-1.5 w-full rounded border border-line bg-white px-3 py-2 text-night placeholder:text-slate/40 focus:border-link focus:outline-none";
export const btnCls = "rounded bg-action px-4 py-2 text-sm font-medium text-night transition hover:bg-action-dark disabled:opacity-60";
export const btnGhostCls = "rounded border border-line bg-white px-4 py-2 text-sm font-medium text-night transition hover:bg-page disabled:opacity-60";
export const btnDangerCls = "rounded border border-danger/40 bg-white px-4 py-2 text-sm font-medium text-danger transition hover:bg-danger/5 disabled:opacity-60";
