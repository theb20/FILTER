import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";

// Champ mot de passe avec bouton œil (afficher / masquer). Reçoit les props d'un <input>.
export default function PasswordInput({ className = "", ...props }) {
  const { t } = useI18n();
  const [visible, setVisible] = useState(false);
  return (
    <div className="relative w-full">
      <input {...props} type={visible ? "text" : "password"} className={`${className} pr-11`} />
      <button
        type="button"
        onClick={() => setVisible((v) => !v)}
        aria-label={visible ? t("auth.hidePassword") : t("auth.showPassword")}
        aria-pressed={visible}
        className="absolute inset-y-0 right-0 flex items-center px-3 text-gray-500 hover:text-black"
      >
        {visible ? <EyeOff size={18} aria-hidden="true" /> : <Eye size={18} aria-hidden="true" />}
      </button>
    </div>
  );
}
