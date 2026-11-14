import { Outlet } from "react-router-dom";
import TopBar from "./TopBar.jsx";
import Header from "./header.jsx";
import GeoBanner from "./GeoBanner.jsx";
import Newsletter from "./Newsletter.jsx";
import Footer from "./Footer.jsx";
import HelpBubble from "./HelpBubble.jsx";
import Cookie from "./Cookie.jsx";
import { useLocation } from "react-router-dom";

export default function Layout() {
  const { pathname } = useLocation();
  const bare = pathname === "/sign" || pathname === "/reset";

  return (
    <div className="min-h-screen bg-page">
      <Header />
      <TopBar />
      <GeoBanner />

      <main>
        <Outlet />
      </main>

      {!bare && <Newsletter />}
      <Footer />
      {!bare && <HelpBubble />}
      <Cookie />
    </div>
  );
}
