import { lazy, Suspense } from "react";
import { Outlet, useLocation } from "react-router-dom";
import TopBar from "./TopBar.jsx";
import Header from "./header.jsx";
import GeoBanner from "./GeoBanner.jsx";
import Newsletter from "./Newsletter.jsx";
import Footer from "./Footer.jsx";
import HelpBubble from "./HelpBubble.jsx";
import Cookie from "./Cookie.jsx";
import MotionProvider from "../animation/MotionProvider.jsx";

// Bande « À la une » : chargée à la demande, uniquement sur les pages du blog.
const FeaturedBar = lazy(() => import("./blog/FeaturedBar.jsx"));

// Pages plein écran sans header ni pied de page (connexion/inscription).
const NO_HEADER = ["/sign"];

export default function Layout() {
  const { pathname } = useLocation();
  const showHeader = !NO_HEADER.includes(pathname);
  const isBlog = pathname === "/blog" || pathname.startsWith("/blog/");

  return (
    <MotionProvider>
    <div className="min-h-screen bg-page">
      {showHeader && (
        <>
          {isBlog && <Suspense fallback={null}><FeaturedBar /></Suspense>}
          <Header />
          <TopBar />
          <GeoBanner />
        </>
      )}

      <main>
        <Outlet />
      </main>

      
      {showHeader && (
        <>
          <Newsletter />
          <Footer />
        </>
      )}
      <HelpBubble />
      <Cookie />
    </div>
    </MotionProvider>
  );
}
