import { lazy, Suspense, useEffect } from "react";
import { Routes, Route } from "react-router-dom";
import { NotificationProvider } from "./assets/configurations/NotificationProvider.jsx";

// Layout principal
import Layout from "./assets/components/layout.jsx";

// Pages chargées à la demande (découpage du bundle par route)
const Sign = lazy(() => import("./assets/pages/sign.jsx"));
const Reset = lazy(() => import("./assets/pages/reset.jsx"));
const Contact = lazy(() => import("./assets/pages/contact.jsx"));
const Basket = lazy(() => import("./assets/pages/basket.jsx"));
const Category = lazy(() => import("./assets/pages/category.jsx"));
const Card = lazy(() => import("./assets/pages/card.jsx"));
const Home = lazy(() => import("./assets/pages/home.jsx"));
const Profil = lazy(() => import("./assets/pages/profil.jsx"));
const Quote = lazy(() => import("./assets/pages/quote.jsx"));
const Privacy = lazy(() => import("./assets/pages/privacy.jsx"));

const InfoPage = lazy(() => import("./assets/pages/info.jsx"));

const NotFound = () => {
  useEffect(() => {
    document.title = "Page introuvable — Filter Finder";
  }, []);
  return (
    <div className="flex flex-col items-center justify-center h-screen text-center">
      <h1 className="text-4xl font-bold mb-4 dark:text-white">404 - Page non trouvée</h1>
      <p className="text-gray-600 dark:text-gray-400">
        La page que vous cherchez n'existe pas.
      </p>
    </div>
  );
};

const PageLoader = () => (
  <div className="min-h-[50vh] flex items-center justify-center text-gray-500" role="status">
    Chargement…
  </div>
);

function App() {
  return (
    <div className="min-h-screen relative h-full bg-white dark:bg-gray-900 transition-colors duration-300">
      <NotificationProvider>
          <Suspense fallback={<PageLoader />}>
            <Routes>
              {/* Routes avec layout global */}
              <Route element={<Layout />}>
                <Route index element={<Home />} />
                <Route path="/sign" element={<Sign />} />
                <Route path="/reset" element={<Reset />} />
                <Route path="/basket" element={<Basket />} />
                <Route path="/category" element={<Category />} />
                <Route path="/contact" element={<Contact />} />
                <Route path="/card" element={<Card />} />
                <Route path="/profil" element={<Profil />} />
                <Route path="/quote" element={<Quote />} />
                <Route path="/privacy" element={<Privacy />} />
                <Route path="/blog" element={<InfoPage title="Blog" />} />
                <Route path="/brands" element={<InfoPage title="Brands" />} />
                <Route path="/terms" element={<InfoPage title="Terms & Conditions" />} />
                <Route path="/refund" element={<InfoPage title="Refund policy" />} />
              </Route>

              {/* Page 404 */}
              <Route path="*" element={<NotFound />} />
            </Routes>
          </Suspense>
      </NotificationProvider>
    </div>
  );
}

export default App;
