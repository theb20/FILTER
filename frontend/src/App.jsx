import { lazy, Suspense, useEffect } from "react";
import { Routes, Route, Link } from "react-router-dom";
import { I18nProvider } from "./assets/i18n/I18nProvider.jsx";
import { useI18n } from "./assets/i18n/i18nContext.js";
import { NotificationProvider } from "./assets/configurations/NotificationProvider.jsx";
import { useAuth } from "./assets/store/authStore.js";

// Layout principal
import Layout from "./assets/components/layout.jsx";

// Pages chargées à la demande (découpage du bundle par route)
const Sign = lazy(() => import("./assets/pages/sign.jsx"));
const Reset = lazy(() => import("./assets/pages/reset.jsx"));
const Contact = lazy(() => import("./assets/pages/contact.jsx"));
const Basket = lazy(() => import("./assets/pages/basket.jsx"));
const Category = lazy(() => import("./assets/pages/category.jsx"));
const Card = lazy(() => import("./assets/pages/productDetail.jsx"));
const Home = lazy(() => import("./assets/pages/home.jsx"));
const Profil = lazy(() => import("./assets/pages/profil.jsx"));
const SecureAccount = lazy(() => import("./assets/pages/secureAccount.jsx"));
const Quote = lazy(() => import("./assets/pages/quote.jsx"));
const Privacy = lazy(() => import("./assets/pages/privacy.jsx"));
const OAuthCallback = lazy(() => import("./assets/pages/oauthCallback.jsx"));
const WebResult = lazy(() => import("./assets/pages/webResult.jsx"));
const VerifyEmail = lazy(() => import("./assets/pages/verifyEmail.jsx"));
const InfoPage = lazy(() => import("./assets/pages/info.jsx"));
const Brands = lazy(() => import("./assets/pages/brands.jsx"));
const Blog = lazy(() => import("./assets/pages/blog.jsx"));
const BlogPost = lazy(() => import("./assets/pages/blogDetail.jsx"));

function NotFound() {
  const { t } = useI18n();
  useEffect(() => {
    document.title = t("meta.notFound");
  }, [t]);
  return (
    <div className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-10 text-center shadow-card">
      <h1 className="text-[40px] font-bold">{t("notFound.title")}</h1>
      <p className="mt-2">{t("notFound.text")}</p>
      <Link to="/" className="mt-6 inline-flex rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">
        {t("notFound.home")} <span aria-hidden="true">&nbsp;→</span>
      </Link>
    </div>
  );
}

function PageLoader() {
  const { t } = useI18n();
  return (
    <div className="flex min-h-[50vh] items-center justify-center text-gray-500" role="status">
      {t("loading")}
    </div>
  );
}

function App() {
  useEffect(() => {
    useAuth.getState().init(); // restaure la session via le cookie de rafraîchissement
  }, []);
  return (
    <I18nProvider>
      <NotificationProvider>
        <Suspense fallback={<PageLoader />}>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<Home />} />
              <Route path="/sign" element={<Sign />} />
              <Route path="/reset" element={<Reset />} />
              <Route path="/oauth/callback" element={<OAuthCallback />} />
              <Route path="/web/:id" element={<WebResult />} />
              <Route path="/verify-email" element={<VerifyEmail />} />
              <Route path="/basket" element={<Basket />} />
              <Route path="/category" element={<Category />} />
              <Route path="/contact" element={<Contact />} />
              <Route path="/card" element={<Card />} />
              <Route path="/profil" element={<Profil />} />
              <Route path="/secure-account" element={<SecureAccount />} />
              <Route path="/quote" element={<Quote />} />
              <Route path="/privacy" element={<Privacy />} />
              <Route path="/blog" element={<Blog />} />
              <Route path="/blog/:slug" element={<BlogPost />} />
              <Route path="/brands" element={<Brands />} />
              <Route path="/about" element={<InfoPage titleKey="info.about" slug="about" />} />
              <Route path="/terms" element={<InfoPage titleKey="info.terms" slug="terms" />} />
              <Route path="/refund" element={<InfoPage titleKey="info.refund" slug="refund" />} />
              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </Suspense>
      </NotificationProvider>
    </I18nProvider>
  );
}

export default App;
