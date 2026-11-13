import { getSystemInfo } from "./api";
import { StatusCard } from "@/components/ui/status-card";
export async function Connectivity() {
  const info = await getSystemInfo().catch(() => null);
  if (info) {
    return <StatusCard title="Koneksi backend"><p role="status" className="mb-4 font-medium text-emerald-700 dark:text-emerald-400">Terhubung</p>
      <dl className="grid grid-cols-2 gap-3 text-sm"><dt>Aplikasi</dt><dd>{info.app_name}</dd><dt>Environment</dt><dd>{info.environment}</dd><dt>Versi</dt><dd>{info.version}</dd></dl>
      <p className="mt-4 text-sm text-zinc-500">API dapat dihubungi. Kesiapan database diperiksa melalui health readiness.</p>
    </StatusCard>;
  } else {
    return <StatusCard title="Koneksi backend"><p role="status" className="font-medium text-amber-700 dark:text-amber-400">Backend belum tersedia</p><p className="mt-3 text-sm">Periksa layanan backend dan konfigurasi server, lalu muat ulang halaman.</p></StatusCard>;
  }
}
