import { Headphones, RotateCcw, ShieldCheck, Truck } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";

const ITEMS = [
  { key: "ship", Icon: Truck },
  { key: "returns", Icon: RotateCcw },
  { key: "pay", Icon: ShieldCheck },
  { key: "support", Icon: Headphones },
];

// Bandeau de réassurance (réutilisable : fiche produit, panier…).
export default function ReassuranceStrip({ className = "" }) {
  const { t } = useI18n();
  return (
    <ul className={`grid gap-4 rounded-[8px] bg-white p-5 shadow-card sm:grid-cols-2 lg:grid-cols-4 ${className}`}>
      {ITEMS.map(({ key, Icon }) => (
        <li key={key} className="flex items-start gap-3">
          <Icon size={26} className="mt-0.5 shrink-0 text-teal" aria-hidden="true" />
          <div>
            <p className="font-bold">{t(`reassure.${key}.title`)}</p>
            <p className="text-sm text-gray-600">{t(`reassure.${key}.text`)}</p>
          </div>
        </li>
      ))}
    </ul>
  );
}
