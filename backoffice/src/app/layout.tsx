import type { Metadata } from "next";
import { Roboto } from "next/font/google";
import "./globals.css";

const roboto = Roboto({ variable: "--font-roboto", subsets: ["latin"], weight: ["400", "500", "700"] });

export const metadata: Metadata = {
  title: { default: "FilterFinder · Backoffice", template: "%s · Backoffice FilterFinder" },
  description: "Administration FilterFinder",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="fr" className={`${roboto.variable} h-full`}>
      <body className="min-h-full">{children}</body>
    </html>
  );
}
