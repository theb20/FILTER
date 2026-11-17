import { redirect } from "next/navigation";
import { logout } from "@/app/login/actions";
import { currentStaff } from "@/lib/staff";
import { can } from "@/lib/admin";
import { AdminShell, type NavGroup } from "@/components/admin-shell";
import { GuideProvider } from "@/components/guide/guide-provider";

// Chaque entrée n'apparaît que si le membre a la permission de la page (le gateway la contrôle de toute façon).
const GROUPS: { title: string; items: { href: string; label: string; icon: string; perm?: string }[] }[] = [
  { title: "Pilotage", items: [{ href: "/dashboard", label: "Tableau de bord", icon: "dashboard" }] },
  {
    title: "Ventes",
    items: [
      { href: "/customers", label: "Clients", icon: "customers", perm: "customers:read" },
      { href: "/companies", label: "Entreprises", icon: "companies", perm: "customers:read" },
    ],
  },
  {
    title: "Boutique",
    items: [
      { href: "/content/shipping-zones", label: "Zones de livraison", icon: "shipping", perm: "shipping:read" },
      { href: "/content/promo-codes", label: "Codes promotionnels", icon: "promo", perm: "pricing:read" },
      { href: "/content/settings/commerce", label: "Règles commerciales", icon: "settings", perm: "pricing:read" },
      { href: "/content/quote-requests", label: "Demandes de devis", icon: "quotes", perm: "quotes:read" },
      { href: "/content/reviews", label: "Avis clients", icon: "reviews", perm: "reviews:read" },
    ],
  },
  {
    title: "Contenu du site",
    items: [
      { href: "/content/settings/site", label: "Informations du site", icon: "settings", perm: "content:read" },
      { href: "/content/faq", label: "Questions fréquentes", icon: "faq", perm: "content:read" },
      { href: "/content/pages", label: "Pages légales", icon: "pages", perm: "content:read" },
      { href: "/content/articles", label: "Articles du blog", icon: "articles", perm: "content:read" },
      { href: "/content/blog-categories", label: "Catégories du blog", icon: "categories", perm: "content:read" },
      { href: "/content/blog-tags", label: "Étiquettes du blog", icon: "brands", perm: "content:read" },
      { href: "/content/authors", label: "Auteurs", icon: "staff", perm: "content:read" },
    ],
  },
  {
    title: "Messages reçus",
    items: [
      { href: "/content/contact-messages", label: "Messages de contact", icon: "mail", perm: "support:read" },
      { href: "/content/newsletter-subscribers", label: "Lettre d'information", icon: "mail", perm: "support:read" },
    ],
  },
  {
    title: "Catalogue",
    items: [
      { href: "/products", label: "Produits", icon: "products", perm: "catalog:read" },
      { href: "/categories", label: "Catégories", icon: "categories", perm: "catalog:read" },
      { href: "/brands", label: "Marques", icon: "brands", perm: "catalog:read" },
      { href: "/sectors", label: "Secteurs", icon: "sectors", perm: "catalog:read" },
    ],
  },
  {
    title: "Finance",
    items: [
      { href: "/finance", label: "Vue d'ensemble", icon: "finance", perm: "finance:read" },
      { href: "/finance/entries", label: "Écritures", icon: "ledger", perm: "finance:read" },
      { href: "/finance/bank", label: "Comptes bancaires", icon: "bank", perm: "treasury:read" },
      { href: "/finance/forecast", label: "Prévisions de trésorerie", icon: "forecast", perm: "treasury:read" },
      { href: "/finance/operations", label: "Dépenses et revenus", icon: "wallet", perm: "finance:read" },
      { href: "/finance/recurring", label: "Opérations récurrentes", icon: "recurring", perm: "finance:read" },
      { href: "/finance/third-parties", label: "Tiers", icon: "customers", perm: "finance:read" },
      { href: "/finance/categories", label: "Catégories", icon: "categories", perm: "finance:read" },
      { href: "/finance/accounts", label: "Plan comptable", icon: "accounts", perm: "finance:read" },
      { href: "/finance/journals", label: "Journaux", icon: "categories", perm: "finance:read" },
      { href: "/finance/periods", label: "Exercices et périodes", icon: "calendar", perm: "finance:read" },
      { href: "/finance/approvals", label: "Approbations", icon: "approvals", perm: "finance:read" },
      { href: "/finance/reports/trial-balance", label: "Balance", icon: "scale", perm: "finance:read" },
      { href: "/finance/reports/ledger", label: "Grand livre", icon: "ledger", perm: "finance:read" },
      { href: "/finance/settings", label: "Réglages", icon: "settings", perm: "finance:read" },
    ],
  },
  {
    title: "Administration",
    items: [
      { href: "/staff", label: "Personnel", icon: "staff", perm: "admin:read" },
      { href: "/roles", label: "Rôles et permissions", icon: "roles", perm: "admin:read" },
      { href: "/audit", label: "Journal d'audit", icon: "audit", perm: "audit:read" },
      { href: "/settings", label: "Réglages d'identité", icon: "settings", perm: "admin:read" },
      { href: "/account", label: "Mon compte", icon: "account" },
    ],
  },
];

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const { me, needsRefresh } = await currentStaff();
  if (!me) redirect(needsRefresh ? "/session/refresh?next=/dashboard" : "/login");

  const groups: NavGroup[] = GROUPS.map((g) => ({
    title: g.title,
    items: g.items.filter((n) => !n.perm || can(me.permissions, n.perm)).map(({ href, label, icon }) => ({ href, label, icon })),
  })).filter((g) => g.items.length > 0);

  return (
    <GuideProvider userId={me.id} permissions={me.permissions}>
      <AdminShell groups={groups} user={{ name: me.name, roles: me.roles }} org="Holding Ledan & Associé" logout={logout}>
        {children}
      </AdminShell>
    </GuideProvider>
  );
}
