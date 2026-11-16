import Link from "next/link";
import { Icon } from "@/components/ui/icon";
export default function PublicLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="public-shell">
      <a className="skip-link" href="#public-content">
        Lewati ke konten utama
      </a>
      <header className="public-header">
        <div className="public-header-inner">
          <Link className="public-brand" href="/">
            <span className="brand-mark" aria-hidden="true">
              W
            </span>
            <span className="public-brand-copy">
              <span>APN WBS</span>
              <small>Agrinas Pangan Whistleblowing System</small>
            </span>
          </Link>
          <nav aria-label="Navigasi publik" className="public-nav">
            <Link href="/tracking" className="flex gap-2 items-center justify-center bg-primary">
              <Icon className="text-white" name="file" size={16} />
              <span className="text-white">Buat Laporan</span>
            </Link>
            <Link href="/tracking" className="flex gap-2 items-center justify-center">
              <Icon className="text-primary" name="search" size={16} />
              <span className="text-primary">Lacak laporan</span>
            </Link>
          </nav>
        </div>
      </header>
      <main id="public-content" className="public-main">
        {children}
      </main>
      <footer className="public-footer">
        <div className="public-footer-inner">
          <span>WBS · Pelaporan dan penelusuran status</span>
          <span>
            Gunakan kode laporan dan contact yang sama untuk tracking.
          </span>
        </div>
      </footer>
    </div>
  );
}
