
import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Dictionary Translator",
  description: "Look up Hebrew letters and phrases and translate text.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
