"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import type { User, Resource } from "@/features/admin/contracts";
import { NotificationBell } from "@/features/notifications/bell";
import { LogoutButton } from "@/features/admin/logout-button";
import { Icon } from "@/components/ui/icon";

type NavigationItem = { href: string; label: string; icon: "home" | "reports" | "bell" | "users" | "settings"; group: string };

export function AdminShell({ children, user, csrf, resources }: { children: React.ReactNode; user: User; csrf: string; resources: Resource[] }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const items: NavigationItem[] = [
    { href: "/admin", label: "Ringkasan", icon: "home", group: "Ruang kerja" },
    ...(user.permissions.includes("report.view") || user.role_name === "Super Admin" ? [{ href: "/admin/reports", label: "Laporan", icon: "reports" as const, group: "Ruang kerja" }] : []),
    ...(user.permissions.includes("notifications.manage") || user.role_name === "Super Admin" ? [{ href: "/admin/notification-deliveries", label: "Pengiriman notifikasi", icon: "bell" as const, group: "Operasional" }] : []),
    ...resources.map((resource) => ({ href: `/admin/${resource.path}`, label: resource.title, icon: (resource.path === "users" ? "users" : "settings") as "users" | "settings", group: resource.security ? "Administrasi" : "Data master" })),
  ];
  const active = (href: string) => href === "/admin" ? pathname === href : pathname === href || pathname.startsWith(`${href}/`);
  const current = [...items].sort((a, b) => b.href.length - a.href.length).find(item => active(item.href));
  const groups = [...new Set(items.map(item => item.group))];
  return <div className="admin-shell">
    <button type="button" className={`admin-overlay ${mobileOpen ? "is-open" : ""}`} onClick={() => setMobileOpen(false)} aria-label="Tutup navigasi" tabIndex={mobileOpen ? 0 : -1} />
    <aside className={`admin-sidebar ${collapsed ? "is-collapsed" : ""} ${mobileOpen ? "mobile-open" : ""}`} aria-label="Navigasi administrasi">
      <div className="sidebar-brand"><Link href="/admin" className="brand-mark" aria-label="WBS Admin, beranda">W</Link><div className="brand-copy"><strong>WBS</strong><span>Ruang administrasi</span></div><button className="icon-button sidebar-close" aria-label="Tutup navigasi" onClick={() => setMobileOpen(false)}><Icon name="close" /></button></div>
      <nav className="sidebar-nav" aria-label="Menu utama">
        {groups.map(group => <div className="nav-group" key={group}><p className="nav-group-label">{group}</p>{items.filter(item => item.group === group).map(item => <Link key={item.href} href={item.href} title={collapsed ? item.label : undefined} aria-current={active(item.href) ? "page" : undefined} className={`nav-link ${active(item.href) ? "is-active" : ""}`} onClick={() => setMobileOpen(false)}><Icon name={item.icon} /><span>{item.label}</span></Link>)}</div>)}
      </nav>
      <div className="sidebar-footer"><span className="status-indicator" /><span className="sidebar-footer-copy">Sistem operasional</span><button className="icon-button collapse-button" aria-label={collapsed ? "Perluas navigasi" : "Ciutkan navigasi"} aria-expanded={!collapsed} onClick={() => setCollapsed(value => !value)}><Icon name="chevron" /></button></div>
    </aside>
    <div className="admin-main">
      <header className="admin-topbar"><button className="icon-button mobile-menu" aria-label="Buka navigasi" aria-expanded={mobileOpen} onClick={() => setMobileOpen(true)}><Icon name="menu" /></button><div className="topbar-context"><p className="breadcrumb">Administrasi <span>/</span> <strong>{current?.label ?? "Ringkasan"}</strong></p><p className="topbar-caption">Whistleblowing System</p></div><div className="topbar-actions">{(user.permissions.includes("report.view") || user.role_name === "Super Admin") && <NotificationBell csrf={csrf} />}<div className="profile-summary"><span className="avatar" aria-hidden="true">{user.name.trim().slice(0, 1).toUpperCase()}</span><div className="profile-copy"><strong>{user.name}</strong><span>{user.role_name}</span></div></div><LogoutButton csrf={csrf} /></div></header>
      <main id="main-content" className="admin-content">{children}</main>
      <footer className="admin-footer">WBS · Administrasi internal</footer>
    </div>
  </div>;
}
