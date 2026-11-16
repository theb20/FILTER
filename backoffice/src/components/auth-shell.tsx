export function AuthShell({ children }: { children: React.ReactNode }) {
  return (
    <main className="grid min-h-screen lg:grid-cols-[1.1fr_1fr]">
      <section className="relative hidden flex-col justify-between overflow-hidden bg-night p-12 text-white lg:flex">
        <div className="flex items-center gap-3">
          <span className="grid h-10 w-10 place-items-center rounded bg-action text-lg font-bold text-night">F</span>
          <span className="text-lg font-medium tracking-wide">FilterFinder</span>
        </div>
        <div>
          <p className="text-4xl font-bold leading-tight">
            Pilotez le catalogue,
            <br />
            les commandes et les clients.
          </p>
          <p className="mt-4 max-w-md text-white/60">Espace réservé au personnel. Accès protégé par une double authentification.</p>
        </div>
        <p className="text-sm text-white/40">Holding Ledan &amp; Associé · Abidjan</p>
        <div className="pointer-events-none absolute -right-24 -top-24 h-72 w-72 rounded-full bg-action/10" />
      </section>
      <section className="flex items-center justify-center bg-white p-6">
        <div className="w-full max-w-sm">{children}</div>
      </section>
    </main>
  );
}

export const inputClass =
  "mt-1.5 w-full rounded border border-line bg-white px-3 py-2.5 text-night placeholder:text-slate/40 focus:border-link focus:outline-none";
export const buttonClass =
  "w-full rounded bg-action px-4 py-3 font-medium text-night transition hover:bg-action-dark disabled:cursor-not-allowed disabled:opacity-60";
