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

const show = (a) => (typeof a.value === "boolean" ? (a.value ? "✓" : "✗") : `${a.value}${a.unit ? ` ${a.unit}` : ""}`);

// Colonne « Détails du produit » : description, caractéristiques, compatibilités, références croisées, secteurs.
export default function ProductDetails({ product }) {
  const { t } = useI18n();
  const specRows = [
    [t("prod.reference"), product.reference],
    ...(product.brand ? [[t("prod.spec.brand"), product.brand]] : []),
    [t("prod.spec.category"), product.categoryName],
    ...product.attributes.map((a) => [a.label, show(a)]),
  ];

  return (
    <section aria-labelledby="pd-details-title" className="rounded-[8px] bg-white p-5 shadow-card">
      <h2 id="pd-details-title" className="text-[20px] font-bold">{t("pd.details")}</h2>
      {product.description && <p className="mt-2 whitespace-pre-line text-[15px] text-gray-700">{product.description}</p>}

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
        {product.compat.length > 0 && (
          <Accordion title={t("pd.compat")}>
            <ul className="flex flex-wrap gap-2">
              {product.compat.map((c) => <li key={c} className="rounded-[3px] bg-gray-100 px-2 py-1 text-sm">{c}</li>)}
            </ul>
          </Accordion>
        )}
        <Accordion title={t("prod.crossRefs")}>
          {product.crossRefs.length ? (
            <ul className="flex flex-wrap gap-2">
              {product.crossRefs.map((r) => <li key={r} className="rounded-[3px] bg-gray-100 px-2 py-1 text-sm">{r}</li>)}
            </ul>
          ) : <p className="text-sm text-gray-600">{t("prod.noCross")}</p>}
        </Accordion>
        {product.sectors.length > 0 && (
          <Accordion title={t("pd.sectors")}>
            <ul className="flex flex-wrap gap-2">
              {product.sectors.map((s) => <li key={s.id} className="rounded-[3px] bg-gray-100 px-2 py-1 text-sm">{s.name}</li>)}
            </ul>
          </Accordion>
        )}
      </div>
    </section>
  );
}
