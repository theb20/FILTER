import { Link } from "react-router-dom";

export default function Logo({ to = "/", className = "" }) {
  return (
    <Link
      to={to}
      aria-label="FilterFinder — home"
      className={`flex items-center gap-2 text-2xl font-bold leading-none select-none ${className}`}
    >
      <img src="/img/logo/favicon.svg" alt="Logo" className="h-9 w-9" />
      <div className="flex flex-col leading-none">
          <span className="text-brand-action">HOLDING</span>
          <span className=" text-[0.8rem] text-brand-action">Ledan & Associe</span>
      </div>
    </Link>
  );
}
