import { Link } from "react-router-dom";

export default function Logo({ to = "/", className = "" }) {
  return (
    <Link
      to={to}
      aria-label="FilterFinder — home"
      className={`flex items-center gap-2 text-2xl font-bold leading-none select-none ${className}`}
    >
      <span className="inline-flex h-9 w-9 items-center justify-center rounded-full border-2 border-white/90">
        <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="#fff" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
          <circle cx="12" cy="12" r="8" />
          <path d="M12 4v16M6 8h12M6 16h12" strokeWidth="1.4" />
        </svg>
      </span>
      <span className="text-white">Filter</span>
      <span className="-ml-1.5 text-brand-red">Finder</span>
    </Link>
  );
}
