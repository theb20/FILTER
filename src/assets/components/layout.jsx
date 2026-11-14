import { Outlet, useLocation } from 'react-router-dom';
import Header from './header.jsx';
import Footer from './Footer.jsx';
import Cookie from './Cookie.jsx';

export default function Layout() {
  const location = useLocation();

  // Routes où le header et le footer doivent être cachés
  const noHeaderFooterRoutes = ['/admin', '/login', '/dashboard'];

  const shouldHideHeaderFooter = noHeaderFooterRoutes.some((path) =>
    location.pathname.startsWith(path)
  );

  return (
    <div className="min-h-screen relative h-full">
      {!shouldHideHeaderFooter && <Header />}

      <main>
        <Outlet />
      </main>

      {!shouldHideHeaderFooter && <Cookie />}
      {!shouldHideHeaderFooter && <Footer />}
    </div>
  );
}
