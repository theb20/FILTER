import Link from "next/link";
import { redirect } from "next/navigation";
import { logout } from "@/app/login/actions";
import { currentStaff } from "@/lib/staff";
import { can } from "@/lib/admin";
import { ShieldCogIcon } from "lucide-react";

// Chaque entrée n'apparaît que si le membre a la permission de la page (le gateway la contrôle de toute façon).
const nav = [
  { href: "/dashboard", label: "Tableau de bord" },
  { href: "/customers", label: "Clients", perm: "customers:read" },
  { href: "/companies", label: "Entreprises", perm: "customers:read" },
  { href: "/categories", label: "Catégories", perm: "catalog:read" },
  { href: "/brands", label: "Marques", perm: "catalog:read" },
  { href: "/sectors", label: "Secteurs", perm: "catalog:read" },
  { href: "/staff", label: "Personnel", perm: "admin:read" },
  { href: "/roles", label: "Rôles et permissions", perm: "admin:read" },
  { href: "/audit", label: "Journal d'audit", perm: "audit:read" },
  { href: "/settings", label: "Réglages d'identité", perm: "admin:read" },
  { href: "/account", label: "Mon compte" },
];

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const { me, needsRefresh } = await currentStaff();
  if (!me) redirect(needsRefresh ? "/session/refresh?next=/dashboard" : "/login");

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-60 flex-col bg-night text-white">
        <div className="flex items-center gap-3 border-b border-white/10 px-5 py-5">
          <span className="grid h-8 w-8 place-items-center rounded bg-action font-bold text-night">
            <ShieldCogIcon className="h-5 w-5" />
          </span>
          <span className="font-medium">Backoffice</span>
        </div>
        <nav className="flex-1 space-y-1 p-3 text-sm">
          {nav.filter((n) => !n.perm || can(me.permissions, n.perm)).map((n) => (
            <Link key={n.href} href={n.href} className="block rounded px-3 py-2 text-white/80 hover:bg-white/10 hover:text-white">
              {n.label}
            </Link>
          ))}
        </nav>
      </aside>
      <div className="flex flex-1 flex-col">
        <header className="flex h-14 items-center justify-between border-b border-line bg-white px-6 text-sm text-slate">
          <span>
            <h3>Holding Ledan & Associe</h3>
          </span>
          <div className="flex items-center gap-4">
            <span>
              {me.name} <span className="text-slate/60">· {me.roles.join(", ")}</span>
            </span>
            <form action={logout}>
              <button type="submit" className="rounded border border-line px-3 py-1 text-night hover:bg-page">Déconnexion</button>
            </form>
          </div>
        </header>
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}
