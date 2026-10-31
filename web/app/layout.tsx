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
          <Link href="/" className="brand" aria-label="it-intelligence, overview">
            <svg viewBox="0 0 64 64" width="26" height="26" aria-hidden="true">
              <rect x="2" y="2" width="60" height="60" rx="15" fill="var(--brand-tile)" />
              <line x1="18" y1="44" x2="32" y2="18" stroke="var(--brand-fg)" strokeWidth="3.6" strokeLinecap="round" />
              <line x1="34.1" y1="17.6" x2="48.1" y2="43.6" stroke="var(--brand-derived)" strokeWidth="2" strokeLinecap="round" />
              <line x1="30.9" y1="19.2" x2="44.9" y2="45.2" stroke="var(--brand-derived)" strokeWidth="2" strokeLinecap="round" />
              <line x1="18" y1="44" x2="46" y2="44" stroke="var(--brand-inferred)" strokeWidth="3" strokeDasharray="3.6 3.4" strokeLinecap="round" />
              <circle cx="32" cy="18" r="6" fill="var(--brand-fg)" />
              <circle cx="18" cy="44" r="5.4" fill="var(--brand-fg)" />
              <circle cx="46" cy="44" r="5.4" fill="var(--brand-fg)" />
            </svg>
            <span>it-intelligence</span>
          </Link>
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
