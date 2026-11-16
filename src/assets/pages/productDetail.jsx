import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { DEMO_PRODUCTS } from "../data/demoProducts.js";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import LineArt from "../components/LineArt.jsx";
import Todo from "../components/Todo.jsx";

export default function ProductPage() {
  const notify = useNotify();
  const { t, formatPrice } = useI18n();
  const [params] = useSearchParams();
  const product = DEMO_PRODUCTS.find((p) => String(p.id) === params.get("id")) ?? DEMO_PRODUCTS[0];
  const name = t(`prod.${product.id}`);
  const [qty, setQty] = useState(1);

  useEffect(() => {
    document.title = t("meta.product", { name });
  }, [t, name]);

  const specs = [
    [t("prod.reference"), product.reference],
    [t("prod.spec.brand"), product.brand],
    [t("prod.spec.category"), t(`cat.${product.category}`)],
    [t("prod.spec.dimensions"), null],
    [t("prod.spec.rating"), null],
    [t("prod.spec.materials"), null],
  ];

  return (
    <div className="mx-2.5 mt-2.5 space-y-2.5">
      <nav aria-label={t("prod.breadcrumb")} className="px-1 text-sm text-gray-600">
        <Link to="/" className="text-link hover:underline">{t("nav.home")}</Link> /{" "}
        <Link to="/category" className="text-link hover:underline">{t("nav.products")}</Link> / {name}
      </nav>

      <section className="grid gap-6 rounded-[4px] bg-white p-6 shadow-card md:grid-cols-2">
        <div className="flex aspect-square items-center justify-center bg-gray-50" title={t("prod.photo")}>
          <LineArt stroke="#9ca3af" className="h-4/5" />
        </div>

        <div>
          <h1 className="text-[40px] font-bold leading-tight">{name}</h1>
          <p className="mt-2">{t("prod.reference")}: <strong>{product.reference}</strong> · {product.brand}</p>

          <div className="mt-4">
            <h2 className="text-[24px] font-semibold">{t("prod.crossRefs")}</h2>
            {product.crossRefs.length ? (
              <ul className="mt-1 flex flex-wrap gap-2">
                {product.crossRefs.map((r) => (
                  <li key={r} className="rounded-[3px] bg-gray-100 px-2 py-1 text-sm">{r}</li>
                ))}
              </ul>
            ) : (
              <p className="mt-1 text-sm text-gray-600">{t("prod.noCross")}</p>
            )}
          </div>

          <p className="mt-6 text-[37px] font-bold">{formatPrice(product.price)}</p>
          <p className={`font-medium ${product.inStock ? "text-green-700" : "text-orange-700"}`}>
            {product.inStock ? t("stock.in") : t("stock.back")}
          </p>

          <div className="mt-5 flex flex-wrap items-end gap-4">
            <div>
              <label htmlFor="qty" className="block text-sm font-medium">{t("prod.quantity")}</label>
              <input
                id="qty"
                type="number"
                min="1"
                value={qty}
                onChange={(e) => setQty(Math.max(1, Number(e.target.value) || 1))}
                className="mt-1 w-24 rounded-[3px] border border-gray-300 px-3 py-2"
              />
            </div>
            <button
              type="button"
              onClick={() => notify(t("demo.notice"), "demo")}
              className="inline-flex items-center gap-2 rounded-[3px] bg-action px-6 py-2.5 font-bold text-black hover:bg-action-dark"
            >
              {t("list.addToCart")} <span aria-hidden="true">→</span>
            </button>
          </div>
          <p className="mt-3 text-sm text-amber-800">{t("prod.demoNote")} {t("demo.notice")}</p>
        </div>
      </section>

      <section className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="specs-title">
        <h2 id="specs-title" className="text-[37px] font-bold leading-tight">{t("prod.specs")}</h2>
        <table className="mt-4 w-full text-left">
          <tbody>
            {specs.map(([label, value]) => (
              <tr key={label} className="border-t border-gray-200">
                <th scope="row" className="w-1/3 py-2.5 pr-4 font-medium">{label}</th>
                <td className="py-2.5">{value ?? <Todo>{t("todo.toComplete")}</Todo>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
