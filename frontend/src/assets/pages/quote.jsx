import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ArrowLeft, Bookmark, Building2, CheckCircle2, LogIn, Plus, Send, Trash2, User } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";
import { useSite } from "../store/siteStore.js";
import { useAuth } from "../store/authStore.js";
import { request } from "../lib/api.js";
import { sendForm } from "../lib/forms.js";
import { useCartStore } from "../store/cartStore.js";
import { useProductsById } from "../lib/catalog.js";

const EMPTY = { company: "", address: "", attention: "", jobTitle: "", email: "", phone: "", subject: "", reference: "", deliveryPlace: "", deliveryDelay: "", paymentTerms: "", notes: "" };
const NEW_ROW = () => ({ ref: "", qty: 1 });

const field = "w-full rounded-[10px] border border-gray-200 bg-gray-50/70 px-3.5 py-2.5 text-[14.5px] outline-none transition placeholder:text-gray-400 focus:border-gray-400 focus:bg-white focus:ring-2 focus:ring-action";
const lbl = "mb-1.5 block text-[13px] font-medium text-gray-700";
const Req = () => <span className="text-red-500"> *</span>;

function Section({ n, title, children, ...rest }) {
  return (
    <section {...rest} className="border-b border-gray-100 p-5 last:border-0">
      <h3 className="mb-4 flex items-center gap-2.5 text-[14px] font-semibold">
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-night text-[12px] text-white" aria-hidden="true">{n}</span>{title}
      </h3>
      {children}
    </section>
  );
}

// Demande de devis : formulaire à gauche, aperçu du document en direct à droite (même structure qu'un devis : émetteur, objet,
// correspondant, destinataire, conditions, articles, totaux). Le brouillon se garde dans le compte du client connecté.
export default function Quote() {
  const { t, locale } = useI18n();
  const SITE = useSite();
  const { status, user, accessToken } = useAuth();
  const authed = status === "authenticated";
  const [f, setF] = useState(EMPTY);
  const [name, setName] = useState("");
  const [kind, setKind] = useState("entreprise"); // "entreprise" | "particulier" : un particulier peut aussi demander un devis
  const company = kind === "entreprise";
  const [rows, setRows] = useState([NEW_ROW()]);
  const [submitted, setSubmitted] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState("");
  const [draftState, setDraftState] = useState(""); // "" | "login" | "saved" | "restored" | "expired"
  const [now] = useState(() => new Date());
  const cart = useCartStore((c) => c.items);
  const { products, ready } = useProductsById(cart.map((i) => i.id));
  const [filled, setFilled] = useState(false); // le formulaire a déjà été rempli (brouillon ou panier)
  // Arrivée depuis une carte de résultat web : la désignation devient la première ligne de la demande.
  const [params] = useSearchParams();
  const refFromCard = (params.get("ref") || "").trim().slice(0, 200);
  const qtyFromCard = Math.max(1, Math.min(100000, Math.round(Number(params.get("qty")) || 1)));
  useEffect(() => {
    if (!refFromCard) return;
    setRows((prev) => (prev.length === 1 && !prev[0].ref ? [{ ...prev[0], ref: refFromCard, qty: qtyFromCard }] : prev));
    setFilled(true);
  }, [refFromCard, qtyFromCard]);

  useEffect(() => {
    document.title = t("meta.quote");
  }, [t]);

  // Brouillon du compte : rechargé à l'arrivée du client connecté.
  useEffect(() => {
    if (!authed || filled) return undefined;
    let live = true;
    request("/me/quote-draft", { token: accessToken }).then((r) => {
      if (!live) return;
      if (r.ok) {
        const d = r.data;
        setF({ ...EMPTY, company: d.company ?? "", address: d.address ?? "", attention: d.attention ?? "", jobTitle: d.jobTitle ?? "", email: d.email ?? "", phone: d.phone ?? "", subject: d.subject ?? "",
          reference: d.reference ?? "", deliveryPlace: d.deliveryPlace ?? "", deliveryDelay: d.deliveryDelay ?? "", paymentTerms: d.paymentTerms ?? "", notes: d.message ?? "" });
        setName(d.name ?? "");
        setKind(d.company ? "entreprise" : "particulier");
        if (d.items?.length) setRows(d.items.map((i) => ({ ref: i.reference, qty: i.quantity, productId: i.productId })));
        setDraftState("restored");
        setFilled(true);
      }
    });
    return () => { live = false; };
  }, [authed, accessToken, filled]);

  // Sans brouillon : les articles du panier deviennent les lignes de la demande ; l'e-mail du compte préremplit le champ.
  useEffect(() => {
    if (filled || !ready) return;
    if (cart.length) {
      const lines = cart.filter((i) => products[i.id]).map((i) => ({ ref: products[i.id].reference, qty: i.quantity, productId: i.id }));
      if (lines.length) setRows(lines);
      setFilled(true);
    }
  }, [ready, cart, products, filled]);
  useEffect(() => {
    if (user) {
      if (!filled) setKind(user.kind === "individual" ? "particulier" : "entreprise");
      setF((v) => (v.email ? v : { ...v, email: user.email ?? "" }));
      setName((n) => n || [user.firstName, user.lastName].filter(Boolean).join(" "));
    }
  }, [user]);

  const set = (k) => (e) => setF((v) => ({ ...v, [k]: e.target.value }));
  const setRow = (i, patch) => setRows((r) => r.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));
  const lines = rows.filter((r) => r.ref.trim());
  const date = new Intl.DateTimeFormat(locale, { dateStyle: "long" }).format(now);

  const payload = useMemo(() => ({
    name: name || f.attention, email: f.email, company: company ? f.company : "", phone: f.phone, address: f.address, attention: company ? f.attention : "", jobTitle: company ? f.jobTitle : "", subject: f.subject, reference: f.reference,
    deliveryPlace: f.deliveryPlace, deliveryDelay: f.deliveryDelay, paymentTerms: f.paymentTerms, message: f.notes,
    items: lines.map((x) => ({ reference: x.ref.trim(), quantity: x.qty, ...(x.productId ? { productId: x.productId } : {}) })),
  }), [f, name, lines, company]);

  const saveDraft = async () => {
    if (!authed) return setDraftState("login");
    const body = Object.fromEntries(Object.entries(payload).filter(([, v]) => v !== "" && !(Array.isArray(v) && v.length === 0)));
    const r = await request("/me/quote-draft", { method: "PUT", token: accessToken, body });
    if (r.ok) return setDraftState("saved");
    setDraftState(r.status === 401 ? "expired" : "");
    if (r.status !== 401) setError(t(r.status === 0 || r.status >= 500 ? "form.err.unavailable" : "form.err.invalid"));
  };

  const onSubmit = async (e) => {
    e.preventDefault();
    setSending(true);
    setError("");
    const r = await sendForm("/quote-requests", payload);
    setSending(false);
    if (!r.ok) return setError(t(`form.err.${r.error}`));
    if (authed) request("/me/quote-draft", { method: "DELETE", token: accessToken }); // la demande est partie : le brouillon n'a plus lieu d'être
    setSubmitted(true);
  };

  if (submitted) {
    return (
      <div className="mx-2.5 mt-2.5 rounded-[12px] bg-white p-10 text-center shadow-card">
        <CheckCircle2 size={48} className="mx-auto text-green-600" aria-hidden="true" />
        <h1 className="mt-3 text-[32px] font-bold">{t("quote.title")}</h1>
        <p role="status" className="mx-auto mt-4 max-w-md rounded-[8px] bg-info p-3">{t("quote.sent")}</p>
        <Link to="/category" className="mt-6 inline-flex rounded-[8px] bg-night px-5 py-2.5 font-medium text-white hover:opacity-90">{t("cart.browse")} <span aria-hidden="true">&nbsp;→</span></Link>
      </div>
    );
  }

  const P = ({ label, children }) => (<div><dt className="text-[11px] text-gray-500">{label}</dt><dd className="font-medium text-black">{children || "—"}</dd></div>);

  return (
    <div className="mx-2.5 mt-2.5 rounded-[16px] bg-gray-100 p-3 sm:p-5">
      <Link to="/" className="inline-flex items-center gap-1.5 text-sm text-gray-600 hover:text-black"><ArrowLeft size={14} aria-hidden="true" /> {t("quote.back")}</Link>
      <div className="mt-3 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-[34px] font-bold leading-tight tracking-tight sm:text-[44px]">{t("quote.title")}</h1>
          <p className="mt-1 max-w-xl text-gray-600">{t("quote.lead")}</p>
        </div>
        <div className="flex gap-2.5">
          <button type="button" onClick={saveDraft} className="inline-flex items-center gap-2 rounded-[10px] border border-gray-200 bg-white px-5 py-3 font-medium hover:bg-gray-50"><Bookmark size={16} aria-hidden="true" /> {t("quote.draft")}</button>
          <button type="submit" form="quote-form" disabled={sending} className="inline-flex items-center gap-2 rounded-[10px] bg-night px-5 py-3 font-medium text-white hover:opacity-90 disabled:opacity-60"><Send size={16} aria-hidden="true" /> {sending ? t("form.sending") : t("quote.send")}</button>
        </div>
      </div>

      {draftState === "login" && (
        <div role="alert" className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-[12px] border border-amber-200 bg-amber-50 p-4">
          <div><p className="font-semibold">{t("quote.draft.loginTitle")}</p><p className="text-sm text-gray-700">{t("quote.draft.loginText")}</p></div>
          <Link to="/sign" className="inline-flex items-center gap-2 rounded-[10px] bg-night px-4 py-2.5 font-medium text-white"><LogIn size={16} aria-hidden="true" /> {t("quote.draft.login")}</Link>
        </div>
      )}
      {draftState === "saved" && (
        <div role="status" className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-[12px] border border-green-200 bg-green-50 p-4">
          <div><p className="flex items-center gap-2 font-semibold"><CheckCircle2 size={18} className="text-green-600" aria-hidden="true" /> {t("quote.draft.savedTitle")}</p><p className="text-sm text-gray-700">{t("quote.draft.savedText")}</p></div>
          <Link to="/profil#devis" className="rounded-[10px] border border-gray-300 bg-white px-4 py-2.5 font-medium hover:bg-gray-50">{t("quote.draft.open")}</Link>
        </div>
      )}
      {draftState === "restored" && <p role="status" className="mt-4 rounded-[12px] bg-white px-4 py-3 text-sm shadow-card">{t("quote.draft.restored")}</p>}
      {draftState === "expired" && <p role="alert" className="mt-4 rounded-[12px] border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">{t("quote.draft.expired")} <Link to="/sign" className="font-medium underline">{t("quote.draft.login")}</Link></p>}

      <div className="mt-5 grid gap-3 xl:grid-cols-2">
        <form id="quote-form" onSubmit={onSubmit} className="overflow-hidden rounded-[12px] bg-white shadow-card" aria-labelledby="quote-details">
          <div className="flex items-center justify-between border-b border-gray-100 px-5 py-4">
            <h2 id="quote-details" className="text-[15px] font-semibold">{t("quote.details")}</h2>
            <p className="text-xs text-gray-500"><span className="text-red-500">*</span> {t("quote.required")}</p>
          </div>

          <Section n="1" title={t("quote.sec.company")} data-guide="quote-contact">
            <div role="radiogroup" aria-label={t("quote.sec.company")} className="mb-4 grid grid-cols-2 gap-2 rounded-[12px] bg-gray-100 p-1">
              {[["particulier", User, "quote.kind.individual"], ["entreprise", Building2, "quote.kind.company"]].map(([k, Icon, key]) => (
                <button key={k} type="button" role="radio" aria-checked={kind === k} onClick={() => setKind(k)}
                  className={`flex items-center justify-center gap-2 rounded-[9px] px-3 py-2.5 text-sm font-medium transition ${kind === k ? "bg-white shadow-sm" : "text-gray-600 hover:text-black"}`}>
                  <Icon size={16} aria-hidden="true" /> {t(key)}
                </button>
              ))}
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              {company ? (
                <>
                  <div className="sm:col-span-2"><label htmlFor="q-company" className={lbl}>{t("quote.f.company")}<Req /></label><input id="q-company" required autoComplete="organization" placeholder={t("quote.ph.company")} value={f.company} onChange={set("company")} className={field} /></div>
                  <div className="sm:col-span-2"><label htmlFor="q-address" className={lbl}>{t("quote.f.address")}</label><input id="q-address" autoComplete="street-address" placeholder={t("quote.ph.address")} value={f.address} onChange={set("address")} className={field} /></div>
                  <div><label htmlFor="q-attention" className={lbl}>{t("quote.f.attention")}<Req /></label><input id="q-attention" required autoComplete="name" placeholder={t("quote.ph.attention")} value={f.attention} onChange={(e) => { set("attention")(e); setName(e.target.value); }} className={field} /></div>
                  <div><label htmlFor="q-job" className={lbl}>{t("quote.f.jobTitle")}</label><input id="q-job" autoComplete="organization-title" placeholder={t("quote.ph.jobTitle")} value={f.jobTitle} onChange={set("jobTitle")} className={field} /></div>
                </>
              ) : (
                <>
                  <div className="sm:col-span-2"><label htmlFor="q-fullname" className={lbl}>{t("quote.f.fullName")}<Req /></label><input id="q-fullname" required autoComplete="name" placeholder={t("quote.ph.fullName")} value={name} onChange={(e) => setName(e.target.value)} className={field} /></div>
                  <div className="sm:col-span-2"><label htmlFor="q-address" className={lbl}>{t("quote.f.addressInd")}</label><input id="q-address" autoComplete="street-address" placeholder={t("quote.ph.address")} value={f.address} onChange={set("address")} className={field} /></div>
                </>
              )}
              <div><label htmlFor="q-email" className={lbl}>{t("quote.f.email")}<Req /></label><input id="q-email" type="email" required autoComplete="email" placeholder={t("quote.ph.email")} value={f.email} onChange={set("email")} className={field} /></div>
              <div><label htmlFor="q-phone" className={lbl}>{t("quote.f.phone")}</label><input id="q-phone" type="tel" autoComplete="tel" placeholder={t("quote.ph.phone")} value={f.phone} onChange={set("phone")} className={field} /></div>
            </div>
          </Section>

          <Section n="2" title={t("quote.sec.subject")}>
            <div className="grid gap-4 sm:grid-cols-2">
              <div><label htmlFor="q-subject" className={lbl}>{t("quote.f.subject")}</label><input id="q-subject" placeholder={t("quote.ph.subject")} value={f.subject} onChange={set("subject")} className={field} /></div>
              <div><label htmlFor="q-ref" className={lbl}>{t("quote.f.reference")}</label><input id="q-ref" placeholder={t("quote.ph.reference")} value={f.reference} onChange={set("reference")} className={field} /></div>
            </div>
          </Section>

          <Section n="3" title={t("quote.sec.delivery")}>
            <div className="grid gap-4 sm:grid-cols-2">
              <div><label htmlFor="q-place" className={lbl}>{t("quote.f.deliveryPlace")}</label><input id="q-place" placeholder={t("quote.ph.deliveryPlace")} value={f.deliveryPlace} onChange={set("deliveryPlace")} className={field} /></div>
              <div><label htmlFor="q-delay" className={lbl}>{t("quote.f.deliveryDelay")}</label><input id="q-delay" placeholder={t("quote.ph.deliveryDelay")} value={f.deliveryDelay} onChange={set("deliveryDelay")} className={field} /></div>
              <div className="sm:col-span-2"><label htmlFor="q-pay" className={lbl}>{t("quote.f.paymentTerms")}</label><input id="q-pay" placeholder={t("quote.ph.paymentTerms")} value={f.paymentTerms} onChange={set("paymentTerms")} className={field} /></div>
            </div>
          </Section>

          <Section n="4" title={t("quote.sec.items")} data-guide="quote-items">
            <div className="overflow-hidden rounded-[10px] border border-gray-100">
              <div className="grid grid-cols-[1fr_84px_36px] gap-2 bg-gray-50 px-3 py-2 text-xs text-gray-500"><span>{t("quote.col.desc")}<Req /></span><span>{t("quote.qtyShort")}</span><span /></div>
              <ul className="divide-y divide-gray-100">
                {rows.map((row, i) => (
                  <li key={i} className="grid grid-cols-[1fr_84px_36px] items-center gap-2 px-3 py-2">
                    <input aria-label={`${t("quote.col.desc")} ${i + 1}`} required={i === 0} placeholder={t("quote.ph.item")} value={row.ref} onChange={(e) => setRow(i, { ref: e.target.value, productId: undefined })} className={field} />
                    <input aria-label={`${t("quote.qty")} ${i + 1}`} type="number" min="1" placeholder={t("quote.ph.qty")} value={row.qty} onChange={(e) => setRow(i, { qty: Math.max(1, Number(e.target.value) || 1) })} className={field} />
                    {rows.length > 1 ? <button type="button" aria-label={t("quote.removeRow", { n: i + 1 })} onClick={() => setRows((r) => r.filter((_, idx) => idx !== i))} className="flex h-9 w-9 items-center justify-center rounded-full text-gray-400 hover:bg-red-50 hover:text-red-600"><Trash2 size={16} /></button> : <span />}
                  </li>
                ))}
              </ul>
            </div>
            <button type="button" onClick={() => setRows((r) => [...r, NEW_ROW()])} className="mt-3 inline-flex items-center gap-1.5 text-sm font-medium text-gray-700 hover:text-black"><Plus size={14} aria-hidden="true" /> {t("quote.addRow")}</button>
            <div className="mt-5"><label htmlFor="q-notes" className={lbl}>{t("quote.f.notes")}</label><textarea id="q-notes" rows={4} placeholder={t("quote.ph.notes")} value={f.notes} onChange={set("notes")} className={`${field} resize-y`} /></div>
            {error && <p role="alert" className="mt-3 text-sm text-red-700">{error}</p>}
          </Section>
        </form>

        <aside data-guide="quote-aside" className="h-fit rounded-[12px] bg-white shadow-card xl:sticky xl:top-28" aria-label={t("quote.preview")}>
          <h2 className="border-b border-gray-100 px-5 py-4 text-[15px] font-semibold">{t("quote.preview")}</h2>
          <div className="p-4 sm:p-5">
            <div className="rounded-[10px] bg-gray-100 p-3 sm:p-5">
              <article className="overflow-hidden rounded-[6px] bg-white text-[12.5px] shadow-sm">
                <div className="h-1.5 bg-gradient-to-r from-[#8a3f10] via-[#c8601a] to-[#f4b183]" aria-hidden="true" />
                <div className="p-5 sm:p-6">
                  <header className="grid gap-4 sm:grid-cols-[1.1fr_1fr]">
                    <div className="flex gap-3">
                      <img src="/img/logo/favicon.svg" alt="" className="h-12 w-12 shrink-0" />
                      <div className="min-w-0 text-gray-600">
                        <p className="font-bold uppercase text-black">{SITE.name}</p>
                        {SITE.address && <p>{SITE.address}</p>}
                        {SITE.rccm && <p>{t("quote.p.rccm")} : {SITE.rccm}</p>}
                        {SITE.taxNumber && <p>{t("quote.p.tax")} {SITE.taxNumber}</p>}
                        {SITE.capital && <p>{t("quote.p.capital")} {SITE.capital}</p>}
                      </div>
                    </div>
                    <dl className="space-y-0.5 rounded-[4px] bg-gray-50 p-3 text-right">
                      <p className="text-center text-[15px] font-semibold text-black">{t("quote.docTitle").toUpperCase()}</p>
                      <div className="flex justify-between gap-2"><dt className="text-gray-500">{t("quote.p.ourRef")}</dt><dd>{t("quote.p.assigned")}</dd></div>
                      <div className="flex justify-between gap-2"><dt className="text-gray-500">{t("quote.date")}</dt><dd>{date}</dd></div>
                      <div className="flex justify-between gap-2"><dt className="text-gray-500">{t("quote.p.yourRef")}</dt><dd className="break-all">{f.reference || "—"}</dd></div>
                      <p className="pt-1 text-center font-medium text-black">{f.subject || "—"}</p>
                    </dl>
                  </header>

                  <dl className="mt-4 rounded-[4px] bg-gray-50 p-3 text-gray-700">
                    <div className="flex gap-2"><dt className="w-24 shrink-0 text-gray-500">{t("quote.p.contact")}</dt><dd>{SITE.publisher || SITE.name}</dd></div>
                    {SITE.email && <div className="flex gap-2"><dt className="w-24 shrink-0 text-gray-500">{t("contact.email")}</dt><dd>{SITE.email}</dd></div>}
                    {SITE.phone && <div className="flex gap-2"><dt className="w-24 shrink-0 text-gray-500">{t("contact.phone")}</dt><dd>{SITE.phone}</dd></div>}
                  </dl>

                  <div className="mt-4 grid gap-4 sm:grid-cols-2">
                    <section>
                      <h3 className="border-b-2 border-gray-300 pb-0.5 text-[10.5px] font-bold uppercase tracking-wide text-[#2f4f8f]">{t("quote.p.recipient")}</h3>
                      <p className="mt-1.5 font-bold text-black">{(company ? f.company : name) || "—"}</p>
                      {f.address && <p className="text-gray-600">{f.address}</p>}
                      {company && (f.attention || f.jobTitle) && <p className="mt-1 text-gray-700"><i>{t("quote.p.attention")} :</i> <b>{f.attention}</b>{f.jobTitle && <span> · {f.jobTitle}</span>}</p>}
                      <p className="text-gray-600">{[f.email, f.phone && `+225 ${f.phone}`].filter(Boolean).join(" · ")}</p>
                    </section>
                    <dl className="space-y-1 text-gray-700">
                      <P label={t("quote.p.delivery")}>{f.deliveryDelay}</P>
                      <P label={t("quote.p.place")}>{f.deliveryPlace}</P>
                      <P label={t("quote.p.payment")}>{f.paymentTerms}</P>
                    </dl>
                  </div>

                  <table className="mt-5 w-full text-left">
                    <thead className="bg-gray-100 text-[10.5px] font-bold uppercase text-gray-600">
                      <tr><th className="px-2 py-1.5">{t("quote.col.desc")}</th><th className="w-12 px-2 py-1.5 text-center">{t("quote.qtyShort")}</th><th className="w-20 px-2 py-1.5 text-right">{t("quote.col.unit")}</th><th className="w-20 px-2 py-1.5 text-right">{t("quote.col.total")}</th></tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 border-b border-gray-300">
                      {lines.length === 0 && <tr><td colSpan={4} className="px-2 py-4 text-center text-gray-400">{t("quote.noItems")}</td></tr>}
                      {lines.map((l, i) => (
                        <tr key={i}><td className="break-words px-2 py-1.5">{l.ref}</td><td className="px-2 py-1.5 text-center">{l.qty}</td><td className="px-2 py-1.5 text-right text-gray-400">{t("quote.p.toQuote")}</td><td className="px-2 py-1.5 text-right text-gray-400">—</td></tr>
                      ))}
                    </tbody>
                  </table>

                  <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:justify-between">
                    <p className="max-w-[16rem] text-[11px] leading-snug text-gray-500">{f.notes.trim() ? <span className="whitespace-pre-line break-words text-gray-700">{f.notes}</span> : t("quote.p.note")}</p>
                    <dl className="w-full text-[11.5px] sm:w-52">
                      {["subtotal", "discount", "shipping", "totalHt", "vat", "totalTtc"].map((k, i) => (
                        <div key={k} className={`flex justify-between gap-3 border-b border-gray-200 py-1 ${i === 5 ? "font-bold text-black" : "text-gray-600"}`}><dt>{t(`quote.p.${k}`)}</dt><dd className="text-gray-400">{t("quote.p.toQuote")}</dd></div>
                      ))}
                    </dl>
                  </div>
                </div>
                <div className="h-1.5 bg-gradient-to-r from-[#8fa8d8] via-[#2f4f8f] to-[#1c2e5a]" aria-hidden="true" />
              </article>
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
}
