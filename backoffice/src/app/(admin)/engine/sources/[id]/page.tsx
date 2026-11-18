import { adminGet, adminGetOpt, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { ERROR_KIND_LABEL, HEALTH_LABEL, HEALTH_TONE, MICROS, fromMicros, SOURCE_KIND_LABEL } from "@/lib/engine";
import { Badge, Card, Notice, PageHeader, StatCard, inputCls } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { SourceFields } from "@/components/engine/source-fields";
import SourceTest from "@/components/engine/source-test";
import type { Source, SourceHealth, SourceQuota } from "@/lib/types";
import { disableSource, enableSource, saveQuota, setRawStorage, updateSource } from "../actions";

const amount = (micros: number) => String(micros / MICROS).replace(".", ",");
const pct = (part: number, whole: number) => (whole > 0 ? Math.min(100, Math.round((part / whole) * 100)) : 0);
const when = (iso: string | null) => (iso ? new Date(iso).toLocaleString("fr-FR") : "—");

export default async function SourcePage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const to = `/engine/sources/${id}`;
  const [{ me }, source, quota, health] = await Promise.all([
    currentStaff(),
    adminGet<Source>(`/admin/sources/${id}`, to),
    adminGetOpt<SourceQuota>(`/admin/sources/${id}/quota`, to),
    adminGetOpt<SourceHealth>(`/admin/sources/${id}/health`, to),
  ]);
  const perms = me?.permissions ?? [];
  const canWrite = can(perms, "engine:write");
  const canBudget = can(perms, "engine:budget");
  const st = health ? { tone: HEALTH_TONE[health.status], label: HEALTH_LABEL[health.status] } : { tone: "off" as const, label: source.enabled ? "Active" : "Désactivée" };
  const monthUsed = quota ? quota.monthSpentMicros + quota.monthReservedMicros : 0;
  const dayUsed = quota ? quota.daySpentMicros + quota.dayReservedMicros : 0;
  const noBudget = !quota || quota.monthlyBudgetMicros === 0;

  return (
    <div className="max-w-4xl space-y-5">
      <PageHeader breadcrumb={[{ label: "Sources externes", href: "/engine/sources" }, { label: source.name }]} title={source.name}
        subtitle={`${SOURCE_KIND_LABEL[source.kind] ?? source.kind} · ${source.baseUrl}`} action={<Badge tone={st.tone}>{st.label}</Badge>} />
      {ok === "created" && <Notice kind="ok">Source créée.</Notice>}
      {ok === "updated" && <Notice kind="ok">Modifications enregistrées.</Notice>}
      {ok === "quota" && <Notice kind="ok">Budget enregistré.</Notice>}
      {ok === "enabled" && <Notice kind="ok">Source activée : elle sera interrogée.</Notice>}
      {ok === "disabled" && <Notice kind="ok">Source désactivée : plus aucun appel.</Notice>}
      {ok === "raw-on" && <Notice kind="ok">Stockage des résultats bruts autorisé.</Notice>}
      {ok === "raw-off" && <Notice kind="ok">Stockage des résultats bruts interdit.</Notice>}
      {noBudget && <Notice kind="error">Aucun budget n&apos;est réglé : la source ne sera pas appelée. Renseignez le budget ci-dessous.</Notice>}
      {source.suspended && <Notice kind="error">Source suspendue : le budget est atteint. Elle repartira à la période suivante, ou dès que vous relèverez le budget.</Notice>}

      <Card guide="source-state" title="État" description="Une source désactivée n'est jamais interrogée. L'activation n'a d'effet que si un budget est réglé.">
        <div className="flex flex-wrap items-center gap-3">
          <Badge tone={source.enabled ? "ok" : "off"}>{source.enabled ? "Activée" : "Désactivée"}</Badge>
          {canWrite ? (
            source.enabled
              ? <ActionButton action={disableSource.bind(null, id)} label="Désactiver" variant="ghost" confirm="Désactiver cette source ? Elle ne sera plus interrogée." />
              : <ActionButton action={enableSource.bind(null, id)} label="Activer" />
          ) : <p className="text-sm text-slate">Vous n&apos;avez pas le droit de modifier les sources.</p>}
        </div>
        {health && (
          <dl className="mt-4 grid grid-cols-2 gap-3 text-sm lg:grid-cols-4">
            <div><dt className="text-xs uppercase tracking-wide text-slate">Appels (24 h)</dt><dd className="text-night tabular-nums">{health.calls24h}</dd></div>
            <div><dt className="text-xs uppercase tracking-wide text-slate">Temps médian</dt><dd className="text-night tabular-nums">{health.p50Ms} ms</dd></div>
            <div><dt className="text-xs uppercase tracking-wide text-slate">Taux d&apos;erreur</dt><dd className="text-night tabular-nums">{Math.round(health.errorRate * 100)} %</dd></div>
            <div><dt className="text-xs uppercase tracking-wide text-slate">Dernier succès</dt><dd className="text-night">{when(health.lastSuccessAt)}</dd></div>
            {health.lastErrorKind && <div className="col-span-2 lg:col-span-4"><dt className="text-xs uppercase tracking-wide text-slate">Dernière erreur</dt><dd className="text-night">{ERROR_KIND_LABEL[health.lastErrorKind] ?? health.lastErrorKind} ({when(health.lastErrorAt)})</dd></div>}
          </dl>
        )}
      </Card>

      {canWrite && (
        <Card guide="source-test-card" title="Tester la source" description="Vérifie d'un coup la clé d'API, le réseau et le budget.">
          <SourceTest id={id} errorLabels={ERROR_KIND_LABEL} />
        </Card>
      )}

      <Card guide="source-budget" title="Budget" description="Le service réserve le coût d'un appel avant de le lancer : sans budget restant, l'appel n'a pas lieu.">
        {quota && (
          <div className="mb-4 grid gap-3 sm:grid-cols-2">
            <StatCard label="Consommé ce mois" value={fromMicros(monthUsed, quota.currency)} hint={`sur ${fromMicros(quota.monthlyBudgetMicros, quota.currency)} (${pct(monthUsed, quota.monthlyBudgetMicros)} %)`} tone={quota.alert ? "alert" : "default"} />
            <StatCard label="Consommé aujourd'hui" value={fromMicros(dayUsed, quota.currency)} hint={`sur ${fromMicros(quota.dailyCapMicros, quota.currency)} (${pct(dayUsed, quota.dailyCapMicros)} %)`} />
          </div>
        )}
        {canBudget ? (
          <ActionForm action={saveQuota.bind(null, id)} submit="Enregistrer le budget">
            <div className="grid gap-4 sm:grid-cols-3">
              <label className="block text-sm font-medium text-night">Budget mensuel ({source.currency})
                <input name="monthlyBudget" required inputMode="decimal" defaultValue={quota ? amount(quota.monthlyBudgetMicros) : "5"} className={inputCls} />
              </label>
              <label className="block text-sm font-medium text-night">Plafond par jour ({source.currency})
                <input name="dailyCap" required inputMode="decimal" defaultValue={quota ? amount(quota.dailyCapMicros) : "0,333333"} className={inputCls} />
                <span className="mt-1 block text-xs font-normal text-slate">Proposition : budget ÷ 30 × 2.</span>
              </label>
              <label className="block text-sm font-medium text-night">Alerte à (%)
                <input name="alertThresholdPct" type="number" min={1} max={100} required defaultValue={quota?.alertThresholdPct ?? 80} className={inputCls} />
              </label>
            </div>
            <input type="hidden" name="fallbackSourceId" value={quota?.fallbackSourceId ?? ""} />
            <p className="text-xs text-slate">Avec {fromMicros(source.costPerCallMicros, source.currency)} par appel, un budget de 5 {source.currency} représente environ {Math.floor((5 * MICROS) / Math.max(1, source.costPerCallMicros))} appels par mois.</p>
          </ActionForm>
        ) : <p className="text-sm text-slate">Le réglage du budget demande la permission « Moteur — budget ».</p>}
      </Card>

      <Card guide="source-info" title="Paramètres">
        {canWrite ? <ActionForm action={updateSource.bind(null, id)} submit="Enregistrer"><SourceFields value={source} /></ActionForm> : (
          <dl className="space-y-2 text-sm">
            <div><dt className="text-xs uppercase tracking-wide text-slate">Clé lue dans</dt><dd className="text-night">{source.secretRef}</dd></div>
            <div><dt className="text-xs uppercase tracking-wide text-slate">Coût par appel</dt><dd className="text-night">{fromMicros(source.costPerCallMicros, source.currency)}</dd></div>
          </dl>
        )}
      </Card>

      <Card guide="source-raw" title="Stockage des résultats bruts" description="Conserver les réponses de la source permet de rejouer une recherche. Ce n'est autorisé que si les conditions d'usage de la source le permettent par écrit.">
        <div className="flex flex-wrap items-center gap-3">
          <Badge tone={source.allowsRawStorage ? "warn" : "off"}>{source.allowsRawStorage ? "Autorisé" : "Interdit"}</Badge>
          {canBudget ? (
            source.allowsRawStorage
              ? <ActionButton action={setRawStorage.bind(null, id, false)} label="Interdire le stockage" variant="ghost" />
              : <ActionButton action={setRawStorage.bind(null, id, true)} label="Autoriser le stockage" variant="ghost" confirm="Confirmez-vous que les conditions d'usage de cette source autorisent la conservation des résultats bruts ?" />
          ) : <p className="text-sm text-slate">Demande la permission « Moteur — budget ».</p>}
        </div>
      </Card>
    </div>
  );
}
