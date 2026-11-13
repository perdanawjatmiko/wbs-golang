"use client";
export default function AdminError({ reset }: {
    reset: () => void;
}) { return <main className="p-10"><h1 className="text-xl font-semibold">Administrasi belum tersedia</h1><p className="my-4">Periksa koneksi layanan, lalu coba kembali.</p><button className="admin-button" onClick={reset}>Coba lagi</button></main>; }
