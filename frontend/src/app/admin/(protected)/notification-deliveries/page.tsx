import { requireSession } from '@/features/admin/server';
import { hasPermission } from '@/features/admin/contracts';
import { DeliveryMonitoring } from '@/features/notifications/deliveries';
export default async function DeliveriesPage() { const s = await requireSession(); if (!hasPermission(s.user, 'notifications.manage'))
    return <p role="alert">Akses pengiriman tidak diizinkan.</p>; return <DeliveryMonitoring csrf={s.csrf_token}/>; }
