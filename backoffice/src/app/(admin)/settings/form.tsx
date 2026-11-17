"use client";

import { ActionForm } from "@/components/ui/client";
import { inputCls } from "@/components/ui/ui";
import { saveIdentitySettings } from "./actions";

export type IdentitySettings = {
  accessTokenMinutes: number; customerRefreshDays: number; staffRefreshHours: number; passwordResetCodeMinutes: number; passwordResetMaxAttempts: number;
  staffChallengeMinutes: number; invitationHours: number; emailVerificationHours: number; emailChangeHours: number; recentLoginMinutes: number; loginAlertAfterFailures: number;
  loginThrottle?: { freeAttempts: number; baseDelaySeconds: number; maxDelaySeconds: number };
  retention?: { personalDataDays: number | null; auditLogDays: number | null };
};

function Num({ name, label, hint, value, min = 1, optional, disabled }: { name: string; label: string; hint?: string; value: number | null | undefined; min?: number; optional?: boolean; disabled: boolean }) {
  return (
    <label className="block text-sm font-medium text-night">
      {label}
      <input name={name} type="number" min={min} step={1} defaultValue={value ?? ""} required={!optional} disabled={disabled} placeholder={optional ? "non renseigné" : undefined} className={inputCls} />
      {hint && <span className="mt-1 block text-xs font-normal text-slate">{hint}</span>}
    </label>
  );
}

export function SettingsForm({ s, canWrite }: { s: IdentitySettings; canWrite: boolean }) {
  const d = !canWrite;
  const t = s.loginThrottle;
  const r = s.retention;
  const group = "grid gap-4 sm:grid-cols-2";
  const legend = "text-base font-semibold text-night";
  const body = (
    <div className="space-y-6">
      <fieldset className="space-y-4"><legend className={legend}>Sessions</legend>
        <div className={group}>
          <Num name="accessTokenMinutes" label="Jeton d'accès (minutes)" hint="Défaut 15" value={s.accessTokenMinutes} disabled={d} />
          <Num name="customerRefreshDays" label="Session client (jours)" hint="Défaut 30" value={s.customerRefreshDays} disabled={d} />
          <Num name="staffRefreshHours" label="Session du personnel (heures)" hint="Défaut 8" value={s.staffRefreshHours} disabled={d} />
          <Num name="staffChallengeMinutes" label="Jeton 2FA temporaire (minutes)" hint="Défaut 5" value={s.staffChallengeMinutes} disabled={d} />
          <Num name="recentLoginMinutes" label="« Connexion récente » (minutes)" hint="Défaut 10" value={s.recentLoginMinutes} disabled={d} />
        </div>
      </fieldset>
      <fieldset className="space-y-4"><legend className={legend}>Liens et codes à durée limitée</legend>
        <div className={group}>
          <Num name="invitationHours" label="Invitation du personnel (heures)" hint="Défaut 72" value={s.invitationHours} disabled={d} />
          <Num name="emailVerificationHours" label="Vérification d'adresse (heures)" hint="Défaut 48" value={s.emailVerificationHours} disabled={d} />
          <Num name="emailChangeHours" label="Changement d'adresse (heures)" hint="Défaut 24" value={s.emailChangeHours} disabled={d} />
          <Num name="passwordResetCodeMinutes" label="Code de réinitialisation (minutes)" hint="Défaut 15 — fonction pas encore active" value={s.passwordResetCodeMinutes} disabled={d} />
          <Num name="passwordResetMaxAttempts" label="Essais du code de réinitialisation" hint="Défaut 5 — fonction pas encore active" value={s.passwordResetMaxAttempts} disabled={d} />
        </div>
      </fieldset>
      <fieldset className="space-y-4"><legend className={legend}>Ralentissement des connexions</legend>
        <p className="text-sm text-slate">Par couple compte + adresse IP. Aucun verrouillage de compte : un tiers ne peut pas bloquer un client.</p>
        <div className={group}>
          <Num name="freeAttempts" label="Échecs sans ralentissement" hint="Défaut 3" value={t?.freeAttempts} min={0} disabled={d} />
          <Num name="baseDelaySeconds" label="Délai de base (secondes)" hint="Défaut 1, doublé à chaque échec" value={t?.baseDelaySeconds} disabled={d} />
          <Num name="maxDelaySeconds" label="Délai maximal (secondes)" hint="Défaut 900" value={t?.maxDelaySeconds} disabled={d} />
          <Num name="loginAlertAfterFailures" label="E-mail d'alerte après N échecs (15 min)" hint="Défaut 5" value={s.loginAlertAfterFailures} disabled={d} />
        </div>
      </fieldset>
      <fieldset className="space-y-4"><legend className={legend}>Conservation (jours)</legend>
        <p className="text-sm text-slate">Laisser vide = non renseigné : aucune purge automatique. Les durées légales sont à confirmer avec votre comptable ou juriste.</p>
        <div className={group}>
          <Num name="personalDataDays" label="Données personnelles" value={r?.personalDataDays} min={0} optional disabled={d} />
          <Num name="auditLogDays" label="Journal d'audit" value={r?.auditLogDays} min={0} optional disabled={d} />
        </div>
      </fieldset>
    </div>
  );
  return canWrite ? <ActionForm action={saveIdentitySettings} submit="Enregistrer les réglages">{body}</ActionForm> : <>{body}<p className="mt-4 text-sm text-slate">Lecture seule : la permission « Créer et modifier — administration » est nécessaire pour modifier.</p></>;
}
