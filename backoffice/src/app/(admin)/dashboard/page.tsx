import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card } from "@/components/ui/ui";
import type { CompanyPage } from "@/lib/types";

export default async function DashboardPage() {
  const { me } = await currentStaff();
  const perms = me?.permissions ?? [];
  // Compteur « entreprises en attente » : le total d'une liste d'une seule ligne (l'endpoint d'aperçu n'existe pas encore).
  const pending = can(perms, "customers:read") ? (await adminGet<CompanyPage>("/admin/companies", "/dashboard", { query: { status: "pending", page: 1, pageSize: 1 } })).page.total : null;

  return (
    <div>
      <h1 className="text-2xl font-bold text-night">Bonjour {me?.name?.split(" ")[0]}</h1>
      <p className="mt-1 text-sm text-slate">Les autres indicateurs arriveront avec les services concernés.</p>
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {pending !== null && (
          <Card>
            <p className="text-sm text-slate">Entreprises à valider</p>
            <p className="mt-1 text-3xl font-bold text-night">{pending}</p>
            <Link href="/companies" className="mt-2 inline-block text-sm text-link hover:underline">{pending > 0 ? "Traiter les demandes →" : "Voir les entreprises →"}</Link>
          </Card>
        )}
      </div>
    </div>
  );
}
