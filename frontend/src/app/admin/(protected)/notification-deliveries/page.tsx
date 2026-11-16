import { requireSession } from '@/features/admin/server';
import { hasPermission } from '@/features/admin/contracts';
import { DeliveryMonitoring } from '@/features/notifications/deliveries';
export default async function DeliveriesPage() { const s = await requireSession(); if (!hasPermission(s.user, 'notifications.manage'))
    return <section className="admin-panel permission-state" role="alert"><span className="permission-icon" aria-hidden="true">!</span><div><h1>Akses terbatas</h1><p>Akun Anda tidak memiliki permission notifications.manage.</p></div></section>; return <DeliveryMonitoring csrf={s.csrf_token}/>; }
