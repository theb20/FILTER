import { Outlet } from "react-router-dom";
import TopBar from "./TopBar.jsx";
import Header from "./header.jsx";
import GeoBanner from "./GeoBanner.jsx";
import Newsletter from "./Newsletter.jsx";
import Footer from "./Footer.jsx";
import HelpBubble from "./HelpBubble.jsx";
import Cookie from "./Cookie.jsx";

export default function Layout() {
  return (
    <div className="min-h-screen bg-page">
      <Header />
      <TopBar />
      <GeoBanner />

      <main>
        <Outlet />
      </main>

      <Newsletter />
      <Footer />
      <HelpBubble />
      <Cookie />
    </div>
  );
}
