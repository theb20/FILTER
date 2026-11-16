import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import { CLAIMS, SITE } from "../config/site.js";
import Todo from "./Todo.jsx";

export default function TopBar() {
  const notify = useNotify();
  const { t } = useI18n();

  const sep = <span className="hidden h-4 w-px bg-white/30 sm:block" aria-hidden="true" />;

  return (
    <div className="bg-navy text-sm text-white">
      <div className="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-x-4 gap-y-1 px-4 py-1.5 lg:px-6">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
          <button type="button" onClick={() => notify(t("demo.notice"), "demo")} className="hover:text-action">
            {t("topbar.chat")}
          </button>
          {sep}
          <span>
            {CLAIMS.freeShippingThreshold ? (
              <>
                <strong>{t("topbar.free")}</strong> {t("topbar.over", { amount: CLAIMS.freeShippingThreshold })}
              </>
            ) : (
              <Todo>{t("todo.shippingOffer")}</Todo>
            )}
          </span>
        </div>
        <div>
          {SITE.reviewsUrl ? (
            <a href={SITE.reviewsUrl} target="_blank" rel="noopener noreferrer" className="hover:underline">
              {t("topbar.reviews")} <span className="font-semibold text-trust">Trustpilot</span>
            </a>
          ) : (
            <Todo>{t("todo.reviewsLink")}</Todo>
          )}
        </div>
      </div>
    </div>
  );
}
