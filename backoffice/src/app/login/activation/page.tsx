import QRCode from "qrcode";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { AuthShell, buttonClass } from "@/components/auth-shell";
import { CodeForm } from "@/components/code-form";
import { COOKIE, getChallenge } from "@/lib/session";
import { confirmEnroll, startEnroll } from "../actions";

type Enroll = { secret: string; uri: string; codes: string[] };

export default async function ActivationPage() {
  if (!(await getChallenge())) redirect("/login");
  const raw = (await cookies()).get(COOKIE.enroll)?.value;
  let enroll: Enroll | null = null;
  try {
    enroll = raw ? (JSON.parse(raw) as Enroll) : null;
  } catch {
    enroll = null;
  }

  if (!enroll) {
    return (
      <AuthShell>
        <form action={startEnroll} className="space-y-5">
          <div>
            <h1 className="text-2xl font-bold text-night">Activer la double authentification</h1>
            <p className="mt-1 text-sm text-slate">
              Première connexion : vous devez associer une application d&apos;authentification (Google Authenticator, Microsoft Authenticator, 1Password…).
            </p>
          </div>
          <button type="submit" className={buttonClass}>Commencer</button>
        </form>
      </AuthShell>
    );
  }

  const qr = await QRCode.toDataURL(enroll.uri, { margin: 1, width: 192 });
  return (
    <AuthShell>
      <div className="space-y-5">
        <div>
          <h1 className="text-2xl font-bold text-night">Associez votre application</h1>
          <p className="mt-1 text-sm text-slate">1. Scannez ce QR code. 2. Notez vos codes de secours. 3. Saisissez le code affiché.</p>
        </div>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src={qr} alt="QR code d'activation" width={192} height={192} className="mx-auto rounded border border-line" />
        <p className="break-all text-center font-mono text-xs text-slate">Clé manuelle : {enroll.secret}</p>
        <div className="rounded border border-action/50 bg-action/10 p-3">
          <p className="text-sm font-medium text-night">Codes de secours — affichés une seule fois</p>
          <p className="mt-1 text-xs text-slate">Chacun ne sert qu&apos;une fois si vous perdez votre téléphone. Conservez-les hors de cet ordinateur.</p>
          <ul className="mt-2 grid grid-cols-2 gap-1 font-mono text-sm text-night">
            {enroll.codes.map((c) => (
              <li key={c}>{c}</li>
            ))}
          </ul>
        </div>
        <CodeForm action={confirmEnroll} label="Code affiché par l'application" />
      </div>
    </AuthShell>
  );
}
