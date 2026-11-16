import { ChevronDown } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";

function Accordion({ title, children, open = false }) {
  return (
    <details open={open} className="group border-t border-gray-200 py-3">
      <summary className="flex cursor-pointer list-none items-center justify-between font-bold">
        {title}
        <ChevronDown size={18} className="text-gray-500 transition-transform group-open:rotate-180" aria-hidden="true" />
      </summary>
      <div className="mt-2">{children}</div>
    </details>
  );
}

// Colonne « Détails du produit » : description, caractéristiques, contenu du colis, références croisées.
export default function ProductDetails({ product, mock, name }) {
  const { t } = useI18n();
  const compat = mock.compat.map((k) => t(k)).join(", ");
  const bullets = [
    ...mock.specs.filter((s) => s.value !== "—").map((s) => `${t(`cart.spec.${s.key}`)} : ${s.value}`),
    t("cart.compat", { list: compat }),
  ];
  const specRows = [
    [t("prod.reference"), product.reference],
    [t("prod.spec.brand"), product.brand],
    [t("prod.spec.category"), t(`cat.${product.category}`)],
    ...mock.specs.map((s) => [t(`cart.spec.${s.key}`), s.value]),
  ];

  return (
    <section aria-labelledby="pd-details-title" className="rounded-[8px] bg-white p-5 shadow-card">
      <h2 id="pd-details-title" className="text-[20px] font-bold">{t("pd.details")}</h2>
      <p className="mt-2 text-[15px] text-gray-700">{t("pd.desc.intro", { name, ref: product.reference, brand: product.brand })}</p>
      <ul className="mt-3 list-disc space-y-1 pl-5 text-[15px]">
        {bullets.map((b) => <li key={b}>{b}</li>)}
      </ul>

      <div className="mt-4">
        <Accordion title={t("prod.specs")} open>
          <table className="w-full text-left text-sm">
            <tbody>
              {specRows.map(([label, value]) => (
                <tr key={label} className="border-t border-gray-100">
                  <th scope="row" className="w-2/5 py-1.5 pr-3 font-medium">{label}</th>
                  <td className="py-1.5">{value}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Accordion>
        <Accordion title={t("prod.crossRefs")}>
          {product.crossRefs.length ? (
            <ul className="flex flex-wrap gap-2">
              {product.crossRefs.map((r) => <li key={r} className="rounded-[3px] bg-gray-100 px-2 py-1 text-sm">{r}</li>)}
            </ul>
          ) : <p className="text-sm text-gray-600">{t("prod.noCross")}</p>}
        </Accordion>
        <Accordion title={t("pd.box")}>
          <ul className="list-disc space-y-1 pl-5 text-sm">
            <li>{t("pd.box.1", { name })}</li>
            <li>{t("pd.box.2")}</li>
            <li>{t("pd.box.3")}</li>
          </ul>
        </Accordion>
      </div>
    </section>
  );
}
