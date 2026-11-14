import { useLocation } from "react-router-dom";
import { useNotify } from "../configurations/notificationContext.js";
import { CLAIMS, SITE, DEMO_NOTICE } from "../config/site.js";
import Todo from "./Todo.jsx";

export default function TopBar() {
  const notify = useNotify();
  const { pathname } = useLocation();
  if (pathname === "/sign" || pathname === "/reset") return null;

  const sep = <span className="hidden h-4 w-px bg-white/30 sm:block" aria-hidden="true" />;

  return (
    <div className="bg-navy text-sm text-white">
      <div className="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-x-4 gap-y-1 px-4 py-1.5 lg:px-6">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
          <button type="button" onClick={() => notify(DEMO_NOTICE, "demo")} className="hover:text-action">
            Chat with us
          </button>
          {sep}
          <span>
            {CLAIMS.freeShippingThreshold ? (
              <>
                <strong>FREE Shipping</strong> on orders over {CLAIMS.freeShippingThreshold}
              </>
            ) : (
              <Todo>Shipping offer</Todo>
            )}
          </span>
        </div>
        <div>
          {SITE.reviewsUrl ? (
            <a href={SITE.reviewsUrl} target="_blank" rel="noopener noreferrer" className="hover:underline">
              See our reviews on <span className="font-semibold text-trust">Trustpilot</span>
            </a>
          ) : (
            <Todo>Reviews link</Todo>
          )}
        </div>
      </div>
    </div>
  );
}
