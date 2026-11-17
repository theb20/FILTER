import { currentStaff } from "@/lib/staff";
import { Badge, Card, inputCls, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { GuideSettings } from "@/components/guide/guide-settings";
import { changePassword, logoutEverywhere } from "./actions";

export const metadata = { title: "Mon compte" };

export default async function AccountPage() {
  const { me } = await currentStaff();
  return (
    <div className="max-w-xl space-y-5">
      <PageHeader title="Mon compte" subtitle={me?.email} />
      <Card>
        <dl className="grid gap-3 text-sm sm:grid-cols-2">
          <div><dt className="text-slate">Nom</dt><dd className="font-medium text-night">{me?.name}</dd></div>
          <div><dt className="text-slate">Rôles</dt><dd className="flex flex-wrap gap-1">{me?.roles.map((r) => <Badge key={r} tone="info">{r}</Badge>)}</dd></div>
        </dl>
      </Card>
      <Card guide="account-password">
        <h2 className="mb-4 text-lg font-semibold text-night">Changer mon mot de passe</h2>
        <ActionForm action={changePassword} submit="Changer le mot de passe">
          <label className="block text-sm font-medium text-night">Mot de passe actuel<input name="currentPassword" type="password" autoComplete="current-password" required className={inputCls} /></label>
          <label className="block text-sm font-medium text-night">Nouveau mot de passe <span className="font-normal text-slate">(12 caractères minimum)</span><input name="newPassword" type="password" autoComplete="new-password" minLength={12} required className={inputCls} /></label>
          <label className="block text-sm font-medium text-night">Confirmer<input name="confirm" type="password" autoComplete="new-password" required className={inputCls} /></label>
        </ActionForm>
      </Card>
      <Card guide="account-guide" title="Guide visuel" description="Visites guidées de chaque section. Ces choix sont propres à votre compte.">
        <GuideSettings />
      </Card>
      <Card guide="account-sessions">
        <h2 className="mb-2 text-lg font-semibold text-night">Sessions</h2>
        <p className="mb-3 text-sm text-slate">Ferme toutes vos sessions, y compris celle-ci.</p>
        <ActionButton action={logoutEverywhere} label="Me déconnecter partout" variant="danger" confirm="Fermer toutes vos sessions ?" />
      </Card>
    </div>
  );
}
