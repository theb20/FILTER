import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Trash2 } from "lucide-react";
import { DEMO_PRODUCTS } from "../data/demoProducts.js";
import { useI18n } from "../i18n/i18nContext.js";
import LineArt from "../components/LineArt.jsx";

// Panier de démonstration : non relié aux boutons « Add to cart ».
const INITIAL = [
  { ...DEMO_PRODUCTS[0], quantity: 1 },
  { ...DEMO_PRODUCTS[2], quantity: 4 },
];

export default function Cart() {
  const { t, formatPrice } = useI18n();
  const [items, setItems] = useState(INITIAL);

  useEffect(() => {
    document.title = t("meta.cart");
  }, [t]);

  const setQty = (id, q) =>
    setItems((list) => list.map((i) => (i.id === id ? { ...i, quantity: Math.max(1, q) } : i)));
  const remove = (id) => setItems((list) => list.filter((i) => i.id !== id));
  const subtotal = items.reduce((s, i) => s + i.price * i.quantity, 0);

  return (
    <div className="mx-2.5 mt-2.5 grid gap-2.5 lg:grid-cols-[minmax(0,1fr)_340px]">
      <section className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="cart-title">
        <h1 id="cart-title" className="text-[40px] font-bold leading-tight">{t("cart.title")}</h1>
        <p className="mt-1 text-sm text-amber-800">{t("cart.demo")}</p>

        {items.length === 0 ? (
          <div className="py-10 text-center">
            <p>{t("cart.empty")}</p>
            <Link to="/category" className="mt-4 inline-flex rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">
              {t("cart.browse")} <span aria-hidden="true">&nbsp;→</span>
            </Link>
          </div>
        ) : (
          <ul className="mt-4 divide-y divide-gray-200">
            {items.map((i) => {
              const name = t(`prod.${i.id}`);
              return (
                <li key={i.id} className="flex flex-wrap items-center gap-4 py-4">
                  <div className="flex h-20 w-20 shrink-0 items-center justify-center bg-gray-50">
                    <LineArt stroke="#9ca3af" className="h-16" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <Link to={`/card?id=${i.id}`} className="text-[20px] font-semibold text-link hover:underline">{name}</Link>
                    <p className="text-sm text-gray-600">{t("list.ref")} {i.reference}</p>
                  </div>
                  <label className="text-sm">
                    <span className="sr-only">{t("cart.qtyFor", { name })}</span>
                    <input
                      type="number"
                      min="1"
                      value={i.quantity}
                      onChange={(e) => setQty(i.id, Number(e.target.value) || 1)}
                      className="w-20 rounded-[3px] border border-gray-300 px-2 py-1.5"
                    />
                  </label>
                  <p className="w-28 text-right font-bold">{formatPrice(i.price * i.quantity)}</p>
                  <button type="button" aria-label={t("cart.remove", { name })} onClick={() => remove(i.id)} className="text-gray-500 hover:text-red-600">
                    <Trash2 size={18} />
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <aside className="h-fit rounded-[4px] bg-white p-6 shadow-card" aria-label={t("cart.summary")}>
        <h2 className="text-[24px] font-semibold">{t("cart.summary")}</h2>
        <dl className="mt-4 space-y-2">
          <div className="flex justify-between"><dt>{t("cart.subtotal")}</dt><dd className="font-medium">{formatPrice(subtotal)}</dd></div>
          <div className="flex justify-between text-gray-600"><dt>{t("cart.shipping")}</dt><dd>{t("cart.later")}</dd></div>
          <div className="flex justify-between border-t border-gray-200 pt-3 text-[20px] font-bold"><dt>{t("cart.total")}</dt><dd>{formatPrice(subtotal)}</dd></div>
        </dl>
        <button type="button" disabled aria-describedby="checkout-note" className="mt-5 w-full cursor-not-allowed rounded-[3px] bg-gray-300 px-5 py-3 font-bold text-gray-600">
          {t("cart.checkout")}
        </button>
        <p id="checkout-note" className="mt-3 rounded-[3px] bg-info p-2 text-sm">{t("demo.notice")}</p>
      </aside>
    </div>
  );
}
