import { useCallback, useRef, useState } from "react";
import { X } from "lucide-react";
import { NotificationContext } from "./notificationContext.js";

const STYLES = {
  info: "bg-slate-800 text-white",
  success: "bg-green-700 text-white",
  error: "bg-red-700 text-white",
  demo: "bg-amber-100 text-amber-900 border border-amber-300",
};

export function NotificationProvider({ children }) {
  const [items, setItems] = useState([]);
  const nextId = useRef(1);

  const dismiss = useCallback((id) => {
    setItems((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const notify = useCallback(
    // action (optionnelle) : { label, onClick } — ex. bouton « Annuler »
    (message, type = "info", duration = 5000, action = null) => {
      const id = nextId.current++;
      setItems((prev) => [...prev, { id, message, type, action }]);
      if (duration > 0) setTimeout(() => dismiss(id), duration);
      return id;
    },
    [dismiss],
  );

  return (
    <NotificationContext.Provider value={notify}>
      {children}
      <div
        className="fixed bottom-4 right-4 left-4 sm:left-auto sm:w-96 z-[2000] flex flex-col gap-2"
        role="region"
        aria-label="Notifications"
      >
        {items.map((n) => (
          <div
            key={n.id}
            role={n.type === "error" ? "alert" : "status"}
            className={`flex items-start gap-3 rounded-xl px-4 py-3 text-sm shadow-lg ${STYLES[n.type] ?? STYLES.info}`}
          >
            <p className="flex-1">{n.message}</p>
            {n.action && (
              <button
                type="button"
                onClick={() => { n.action.onClick(); dismiss(n.id); }}
                className="shrink-0 font-bold underline underline-offset-2 hover:no-underline"
              >
                {n.action.label}
              </button>
            )}
            <button
              type="button"
              aria-label="Fermer la notification"
              onClick={() => dismiss(n.id)}
              className="shrink-0 opacity-70 hover:opacity-100"
            >
              <X size={16} />
            </button>
          </div>
        ))}
      </div>
    </NotificationContext.Provider>
  );
}
