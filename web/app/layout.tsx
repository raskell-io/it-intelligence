import type { Metadata } from "next";
import Link from "next/link";
import { ActAs } from "@/components/ActAs";
import "./globals.css";

export const metadata: Metadata = {
  title: "IT intelligence: the Kestrel Group",
  description: "The Context Model of a fictional company: what exists, who owns it, how it connects, and why the model thinks so.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <header className="top">
          <Link href="/" className="brand">IT intelligence</Link>
          <nav>
            <Link href="/">Overview</Link>
            <Link href="/review">Review queue</Link>
            <Link href="/scorecards">Scorecards</Link>
            <Link href="/artifacts">Artifacts</Link>
            <Link href="/runs">Runs</Link>
            <Link href="/ontology">Ontology</Link>
          </nav>
          <span className="spacer" />
          <ActAs />
        </header>
        <main>{children}</main>
      </body>
    </html>
  );
}
