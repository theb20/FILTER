import Link from "next/link";
import { Building2, FilePenLine, PackageCheck, PackagePlus, ScrollText, ShieldAlert, Users, UserCog, ArrowRight, type LucideIcon } from "lucide-react";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader, StatCard } from "@/components/ui/ui";
import type { CompanyPage, CustomerPage, ProductPage, StaffPage } from "@/lib/types";

export const metadata = { title: "Tableau de bord" };

// Compteur = le total d'une liste d'une seule ligne (l'endpoint d'aperçu n'existe pas encore).
const count = async <T extends { page: { total: number } }>(path: string, returnTo: string, query: Record<string, string | number | undefined> = {}) =>
  (await adminGet<T>(path, returnTo, { query: { ...query, page: 1, pageSize: 1 } })).page.total;

const hour = () => new Date().getHours();
const greeting = () => (hour() < 18 ? "Bonjour" : "Bonsoir");

export default async function DashboardPage() {
  const { me } = await currentStaff();
  const perms = me?.permissions ?? [];
  const sales = can(perms, "customers:read");
  const catalog = can(perms, "catalog:read");
  const admin = can(perms, "admin:read");

  const [pending, customers, drafts, review, active, staff] = await Promise.all([
    sales ? count<CompanyPage>("/admin/companies", "/dashboard", { status: "pending" }) : null,
    sales ? count<CustomerPage>("/admin/customers", "/dashboard") : null,
    catalog ? count<ProductPage>("/admin/products", "/dashboard", { status: "draft" }) : null,
    catalog ? count<ProductPage>("/admin/products", "/dashboard", { status: "review" }) : null,
    catalog ? count<ProductPage>("/admin/products", "/dashboard", { status: "active" }) : null,
    admin ? count<StaffPage>("/admin/staff-users", "/dashboard") : null,
  ]);

  const todo = [
    pending ? { href: "/companies", label: `${pending} entreprise${pending > 1 ? "s" : ""} à valider`, icon: Building2 } : null,
    review ? { href: "/products?status=review", label: `${review} produit${review > 1 ? "s" : ""} à valider avant publication`, icon: PackageCheck } : null,
    drafts ? { href: "/products?status=draft", label: `${drafts} brouillon${drafts > 1 ? "s" : ""} à compléter`, icon: FilePenLine } : null,
  ].filter((x): x is NonNullable<typeof x> => x !== null);

  const shortcuts: { href: string; label: string; icon: LucideIcon }[] = [];
  if (can(perms, "catalog:write")) shortcuts.push({ href: "/products/new", label: "Ajouter un produit", icon: PackagePlus });
  if (can(perms, "admin:write")) shortcuts.push({ href: "/staff/new", label: "Inviter un membre", icon: UserCog });
  if (can(perms, "audit:read")) shortcuts.push({ href: "/audit", label: "Consulter le journal d'audit", icon: ScrollText });
  shortcuts.push({ href: "/account", label: "Sécuriser mon compte", icon: ShieldAlert });

  return (
    <div>
      <PageHeader title={`${greeting()} ${me?.name?.split(" ")[0] ?? ""}`} subtitle="Voici l'état de la plateforme et ce qui demande votre attention." />

      <div data-guide="dashboard-stats" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {pending !== null && <StatCard label="Entreprises à valider" value={pending} href="/companies" icon={<Building2 className="h-5 w-5" />} tone={pending > 0 ? "alert" : "default"} hint={pending > 0 ? "Demandes en attente de décision" : "Aucune demande en attente"} />}
        {customers !== null && <StatCard label="Clients" value={customers} href="/customers" icon={<Users className="h-5 w-5" />} hint="Comptes particuliers et entreprises" />}
        {active !== null && <StatCard label="Produits publiés" value={active} href="/products?status=active" icon={<PackageCheck className="h-5 w-5" />} hint={`${(drafts ?? 0) + (review ?? 0)} en préparation`} />}
        {staff !== null && <StatCard label="Membres du personnel" value={staff} href="/staff" icon={<UserCog className="h-5 w-5" />} hint="Comptes avec accès au backoffice" />}
      </div>

      <div className="mt-6 grid gap-4 lg:grid-cols-3">
        <Card guide="dashboard-todo" title="À traiter" description="Les actions qui attendent une décision." className="lg:col-span-2">
          {todo.length === 0 ? (
            <p className="rounded-lg bg-ok/5 px-4 py-6 text-center text-sm text-[#007a52]">Tout est à jour. Rien n&apos;attend de décision pour le moment.</p>
          ) : (
            <ul className="divide-y divide-line">
              {todo.map(({ href, label, icon: Icon }) => (
                <li key={href}>
                  <Link href={href} className="group flex items-center gap-3 py-3 text-sm transition hover:text-link">
                    <span className="grid h-9 w-9 place-items-center rounded-lg bg-action/20 text-[#7a5a00]"><Icon className="h-[18px] w-[18px]" aria-hidden /></span>
                    <span className="flex-1 font-medium text-night group-hover:text-link">{label}</span>
                    <ArrowRight className="h-4 w-4 text-slate transition group-hover:translate-x-0.5" aria-hidden />
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Card>
        <Card guide="dashboard-shortcuts" title="Raccourcis">
          <ul className="space-y-1">
            {shortcuts.map(({ href, label, icon: Icon }) => (
              <li key={href}>
                <Link href={href} className="flex items-center gap-3 rounded-lg px-2 py-2 text-sm text-night transition hover:bg-page">
                  <Icon className="h-[18px] w-[18px] text-slate" aria-hidden />{label}
                </Link>
              </li>
            ))}
          </ul>
        </Card>
      </div>
    </div>
  );
}
