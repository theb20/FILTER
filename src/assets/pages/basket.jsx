import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Trash2 } from "lucide-react";
import { DEMO_PRODUCTS, formatUsd } from "../data/demoProducts.js";
import { DEMO_NOTICE } from "../config/site.js";
import LineArt from "../components/LineArt.jsx";

// Panier de démonstration : non relié aux boutons « Add to cart ».
const INITIAL = [
  { ...DEMO_PRODUCTS[0], quantity: 1 },
  { ...DEMO_PRODUCTS[2], quantity: 4 },
];

export default function Cart() {
  const [items, setItems] = useState(INITIAL);

  useEffect(() => {
    document.title = "Cart — FilterFinder";
  }, []);

  const setQty = (id, q) =>
    setItems((list) => list.map((i) => (i.id === id ? { ...i, quantity: Math.max(1, q) } : i)));
  const remove = (id) => setItems((list) => list.filter((i) => i.id !== id));
  const subtotal = items.reduce((s, i) => s + i.price * i.quantity, 0);

  return (
    <div className="mx-2.5 mt-2.5 grid gap-2.5 lg:grid-cols-[minmax(0,1fr)_340px]">
      <section className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="cart-title">
        <h1 id="cart-title" className="text-[40px] font-bold leading-tight">Cart</h1>
        <p className="mt-1 text-sm text-amber-800">Demo data: items and prices are fictional.</p>

        {items.length === 0 ? (
          <div className="py-10 text-center">
            <p>Your cart is empty.</p>
            <Link to="/category" className="mt-4 inline-flex rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">
              Browse products <span aria-hidden="true">&nbsp;→</span>
            </Link>
          </div>
        ) : (
          <ul className="mt-4 divide-y divide-gray-200">
            {items.map((i) => (
              <li key={i.id} className="flex flex-wrap items-center gap-4 py-4">
                <div className="flex h-20 w-20 shrink-0 items-center justify-center bg-gray-50">
                  <LineArt stroke="#9ca3af" className="h-16" />
                </div>
                <div className="min-w-0 flex-1">
                  <Link to={`/card?id=${i.id}`} className="text-[20px] font-semibold text-link hover:underline">{i.name}</Link>
                  <p className="text-sm text-gray-600">Ref. {i.reference}</p>
                </div>
                <label className="text-sm">
                  <span className="sr-only">Quantity for {i.name}</span>
                  <input
                    type="number"
                    min="1"
                    value={i.quantity}
                    onChange={(e) => setQty(i.id, Number(e.target.value) || 1)}
                    className="w-20 rounded-[3px] border border-gray-300 px-2 py-1.5"
                  />
                </label>
                <p className="w-28 text-right font-bold">{formatUsd(i.price * i.quantity)}</p>
                <button type="button" aria-label={`Remove ${i.name}`} onClick={() => remove(i.id)} className="text-gray-500 hover:text-red-600">
                  <Trash2 size={18} />
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <aside className="h-fit rounded-[4px] bg-white p-6 shadow-card" aria-label="Order summary">
        <h2 className="text-[24px] font-semibold">Order summary</h2>
        <dl className="mt-4 space-y-2">
          <div className="flex justify-between"><dt>Subtotal</dt><dd className="font-medium">{formatUsd(subtotal)}</dd></div>
          <div className="flex justify-between text-gray-600"><dt>Shipping</dt><dd>Calculated later</dd></div>
          <div className="flex justify-between border-t border-gray-200 pt-3 text-[20px] font-bold"><dt>Total</dt><dd>{formatUsd(subtotal)}</dd></div>
        </dl>
        <button type="button" disabled aria-describedby="checkout-note" className="mt-5 w-full cursor-not-allowed rounded-[3px] bg-gray-300 px-5 py-3 font-bold text-gray-600">
          Checkout
        </button>
        <p id="checkout-note" className="mt-3 rounded-[3px] bg-info p-2 text-sm">{DEMO_NOTICE}</p>
      </aside>
    </div>
  );
}
