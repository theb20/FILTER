"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { GuideButton } from "@/components/guide/guide-button";
import {
  Boxes, Building2, ClipboardList, FolderTree, KeyRound, LayoutDashboard, LogOut, Menu, Package, Settings, ShieldCogIcon,
  Tags, UserCog, UserRound, Users, X, Factory, Truck, BadgePercent, FileText, HelpCircle, Newspaper, Mail, Star, ReceiptText, Landmark, BookOpen, Scale, ShieldCheck, CalendarDays, BookMarked, Wallet, Repeat, Banknote, TrendingUp, type LucideIcon,
} from "lucide-react";

const ICONS: Record<string, LucideIcon> = {
  dashboard: LayoutDashboard, customers: Users, companies: Building2, products: Package, categories: FolderTree, brands: Tags,
  sectors: Factory, staff: UserCog, roles: KeyRound, audit: ClipboardList, settings: Settings, account: UserRound, attributes: Boxes,
  finance: Landmark, ledger: BookOpen, scale: Scale, approvals: ShieldCheck, calendar: CalendarDays, accounts: BookMarked, wallet: Wallet, bank: Banknote, forecast: TrendingUp, recurring: Repeat, shipping: Truck, promo: BadgePercent, pages: FileText, faq: HelpCircle, articles: Newspaper, mail: Mail, reviews: Star, quotes: ReceiptText,
};

export type NavItem = { href: string; label: string; icon: string };
export type NavGroup = { title: string; items: NavItem[] };

const initials = (name: string) => name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]!.toUpperCase()).join("") || "?";

export function AdminShell({ groups, user, org, logout, children }: {
  groups: NavGroup[]; user: { name: string; roles: string[] }; org: string; logout: () => Promise<void>; children: React.ReactNode;
}) {
  const pathname = usePathname();
  // Le tiroir est « ouvert pour la page où il a été ouvert » : naviguer le referme sans effet de bord.
  const [openAt, setOpenAt] = useState<string | null>(null);
  const open = openAt === pathname;
  const setOpen = (v: boolean) => setOpenAt(v ? pathname : null);
  const flat = groups.flatMap((g) => g.items);
  const matches = (href: string) => pathname === href || pathname.startsWith(`${href}/`);
  // Entrée la plus spécifique : « /finance » ne doit pas rester actif sur « /finance/entries ».
  const current = flat.filter((i) => matches(i.href)).sort((a, b) => b.href.length - a.href.length)[0];
  const isActive = (href: string) => current?.href === href;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpenAt(null);
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  const sidebar = (
    <div className="flex h-full flex-col bg-night text-white">
      <div className="flex h-16 shrink-0 items-center gap-3 border-b border-white/10 px-5">
        <span className="grid h-9 w-9 place-items-center rounded-lg bg-action text-night"><ShieldCogIcon className="h-5 w-5" aria-hidden /></span>
        <div className="leading-tight">
          <p className="font-semibold">FilterFinder</p>
          <p className="text-xs text-white/50">Backoffice</p>
        </div>
      </div>
      <nav aria-label="Navigation principale" data-guide="nav" className="flex-1 space-y-6 overflow-y-auto px-3 py-5">
        {groups.map((g) => (
          <div key={g.title}>
            <p className="px-3 pb-2 text-[11px] font-semibold uppercase tracking-wider text-white/40">{g.title}</p>
            <ul className="space-y-0.5">
              {g.items.map((n) => {
                const Icon = ICONS[n.icon] ?? LayoutDashboard;
                const active = isActive(n.href);
                return (
                  <li key={n.href}>
                    <Link
                      href={n.href}
                      aria-current={active ? "page" : undefined}
                      className={`group relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition ${active ? "bg-white/10 font-medium text-white" : "text-white/70 hover:bg-white/5 hover:text-white"}`}
                    >
                      {active && <span className="absolute -left-3 top-1/2 h-5 w-1 -translate-y-1/2 rounded-r bg-action" aria-hidden />}
                      <Icon className={`h-[18px] w-[18px] shrink-0 ${active ? "text-action" : "text-white/50 group-hover:text-white/80"}`} aria-hidden />
                      {n.label}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>
      <div className="shrink-0 border-t border-white/10 p-3">
        <div className="flex items-center gap-3 rounded-lg px-2 py-2">
          <span className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-action/90 text-sm font-bold text-night" aria-hidden>{initials(user.name)}</span>
          <div className="min-w-0 flex-1 leading-tight">
            <p className="truncate text-sm font-medium">{user.name}</p>
            <p className="truncate text-xs text-white/50">{user.roles.join(", ") || "—"}</p>
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen">
      <a href="#contenu" className="sr-only z-[100] rounded bg-action px-3 py-2 font-medium text-night focus:not-sr-only focus:fixed focus:left-3 focus:top-3">Aller au contenu</a>

      {/* Barre latérale fixe (desktop) */}
      <aside className="fixed inset-y-0 left-0 z-30 hidden w-64 lg:block">{sidebar}</aside>

      {/* Tiroir (mobile) */}
      {open && (
        <div className="fixed inset-0 z-40 lg:hidden" role="dialog" aria-modal="true" aria-label="Menu">
          <button type="button" aria-label="Fermer le menu" className="absolute inset-0 bg-night/60 backdrop-blur-sm" onClick={() => setOpen(false)} />
          <aside className="relative h-full w-72 max-w-[85vw] shadow-2xl">
            {sidebar}
            <button type="button" onClick={() => setOpen(false)} aria-label="Fermer le menu" className="absolute right-3 top-4 rounded-lg p-1.5 text-white/70 hover:bg-white/10 hover:text-white"><X className="h-5 w-5" /></button>
          </aside>
        </div>
      )}

      <div className="flex min-h-screen flex-col lg:pl-64">
        <header className="sticky top-0 z-20 flex h-16 items-center gap-3 border-b border-line bg-white/85 px-4 backdrop-blur lg:px-8">
          <button type="button" onClick={() => setOpen(true)} aria-label="Ouvrir le menu" aria-expanded={open} className="rounded-lg p-2 text-night hover:bg-page lg:hidden"><Menu className="h-5 w-5" /></button>
          <div className="min-w-0 flex-1 leading-tight">
            <p className="truncate text-sm font-semibold text-night">{current?.label ?? "Backoffice"}</p>
            <p className="truncate text-xs text-slate">{org}</p>
          </div>
          <div className="hidden items-center gap-3 text-sm sm:flex">
            <span className="grid h-8 w-8 place-items-center rounded-full bg-night text-xs font-bold text-action" aria-hidden>{initials(user.name)}</span>
            <span className="max-w-[12rem] truncate text-night">{user.name}</span>
          </div>
          <GuideButton />
          <form action={logout}>
            <button type="submit" className="inline-flex items-center gap-2 rounded-lg border border-line px-3 py-1.5 text-sm font-medium text-night transition hover:bg-page">
              <LogOut className="h-4 w-4" aria-hidden /><span className="hidden sm:inline">Déconnexion</span>
              <span className="sr-only sm:hidden">Déconnexion</span>
            </button>
          </form>
        </header>
        <main id="contenu" className="mx-auto w-full max-w-[1400px] flex-1 p-4 sm:p-6 lg:p-8">{children}</main>
        <footer className="border-t border-line px-6 py-4 text-center text-xs text-slate/70">© {new Date().getFullYear()} FilterFinder · Espace réservé au personnel</footer>
      </div>
    </div>
  );
}
