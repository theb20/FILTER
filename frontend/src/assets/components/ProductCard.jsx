import { Link } from "react-router-dom";
import LineArt from "./LineArt.jsx";
import { useCartStore } from "../store/cartStore.js";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import { flyToCart } from "../animation/gsapSetup.js";

const btnPrimary =
  "inline-flex items-center gap-2 rounded-[3px] bg-action px-4 py-2 font-bold text-black transition-colors hover:bg-action-dark";

// Carte produit réutilisable, reliée au store du panier.
// variant "row" : ligne horizontale (catalogue) ; variant "tile" : carte verticale (grilles, blog).
export default function ProductCard({ product: p, variant = "row" }) {
  const { t, formatPrice } = useI18n();
  const notify = useNotify();
  const addItem = useCartStore((s) => s.addItem);
  const name = p.name;
  const add = (e) => {
    flyToCart(e.currentTarget);
    addItem(p, 1);
    notify(t("cart.added", { name }), "success");
  };
  const stock = (
    <p className={`text-sm font-medium ${p.inStock ? "text-green-700" : "text-gray-700"}`}>
      {p.inStock ? t("stock.in") : p.availability === "made_to_order" ? t("stock.madeToOrder") : t("stock.request")}
    </p>
  );
  const price = p.onQuote ? t("price.onQuote") : formatPrice(p.price);
  const addButton = p.onQuote ? (
    <Link to="/quote" className={btnPrimary}>{t("price.askQuote")} <span aria-hidden="true">→</span></Link>
  ) : (
    <button type="button" onClick={add} className={btnPrimary}>
      {t("list.addToCart")} <span aria-hidden="true">→</span>
    </button>
  );
  const thumb = (cls) => (p.image ? <img src={p.image} alt="" loading="lazy" className={`${cls} object-contain`} /> : <LineArt stroke="#9ca3af" className={cls} />);

  if (variant === "tile") {
    return (
      <article className="flex h-full flex-col rounded-[8px] bg-white p-4 shadow-card">
        <Link to={`/card?id=${p.id}`} aria-label={name} className="flex h-32 items-center justify-center rounded-[6px] bg-gray-50">
          {thumb("h-24")}
        </Link>
        <h3 className="mt-3 text-[17px] font-semibold leading-snug">
          <Link to={`/card?id=${p.id}`} className="text-link hover:underline">{name}</Link>
        </h3>
        <p className="mt-0.5 text-sm text-gray-700">{t("list.ref")} <strong>{p.reference}</strong> · {p.brand}</p>
        <div className="mt-1">{stock}</div>
        <div className="mt-auto flex flex-wrap items-center justify-between gap-2 pt-3">
          <p className="text-[20px] font-bold">{price}</p>
          {addButton}
        </div>
      </article>
    );
  }

  return (
    <li className="flex flex-col gap-4 rounded-[4px] bg-white p-4 shadow-card sm:flex-row">
      <Link to={`/card?id=${p.id}`} className="flex h-36 w-full shrink-0 items-center justify-center bg-gray-50 sm:w-36" aria-label={name}>
        {thumb("h-28")}
      </Link>
      <div className="min-w-0 flex-1">
        <Link to={`/card?id=${p.id}`} className="text-[24px] font-semibold leading-tight text-link hover:underline">{name}</Link>
        <p className="mt-1 text-sm text-gray-700">{t("list.ref")} <strong>{p.reference}</strong> · {p.brand}</p>
        <div className="mt-1">{stock}</div>
      </div>
      <div className="flex shrink-0 flex-col items-start gap-3 sm:items-end">
        <p className="text-[24px] font-bold">{price}</p>
        {addButton}
      </div>
    </li>
  );
}
