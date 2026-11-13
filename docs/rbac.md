# Database-driven RBAC

Relationship: admin_users -> roles -> role_permissions -> permissions. Go resolve permission setiap request, lalu cek lagi di dalam transaksi mutasi. Default deny. Navigation frontend hanya membantu UX; backend selalu memvalidasi.

## Permission matrix

| Resource / capability | Required permission | Additional restriction |
| --- | --- | --- |
| /auth/me, /auth/logout | authenticated active session | Mutation wajib CSRF |
| Admin users CRUD | users.manage | Super Admin only |
| Roles CRUD + permission assignment | roles.manage | Super Admin only |
| Permissions CRUD | permissions.manage | Super Admin only |
| Report types CRUD | report-types.manage OR master.manage | Authenticated |
| FAQs CRUD | faqs.manage OR master.manage | Authenticated |
| Email templates CRUD | email-templates.manage OR master.manage | Authenticated |
| Public report types / FAQs | none | Read-only, active records only |
| report.view / report.update | seeded for future phases | No report endpoint in Phase 1 |

Super Admin secara efektif memiliki semua permission, termasuk permission baru; seeder juga membuat assignment eksplisit untuk semua permission yang ada. Nama role ini reserved dan tidak dapat diubah/dihapus. Role lain tetap dapat dibuat tanpa hierarchy. Investigator awal tidak mendapat permission otomatis; Super Admin dapat mengassign sesuai kebutuhan. Role/permission administratif tambahan adalah keputusan teknis Phase 1, bukan business requirement baru.

## Assignment

POST/PATCH role menerima `permission_ids: UUID[]`, maksimal 200, unik. PATCH mengganti seluruh assignment dalam satu transaksi. Field yang tidak dikirim tetap dipertahankan. Unknown UUID 409; duplicate UUID dalam request 422; composite PK mencegah duplikat database. List/get role menyertakan permission_ids dalam query teragregasi, bukan N+1 query.

## Privilege escalation prevention

- Pengelolaan users/roles/permissions dibatasi Super Admin selain permission check. Memberi `users.manage` kepada role biasa tidak membuka account administration. Pembatasan konservatif ini mencegah delegated admin mengangkat dirinya atau mengubah role tempat ia menjadi anggota.
- User tidak dapat mengganti role akun sendiri. Super Admin lain boleh mengelola role akun tersebut selama invariant last active Super Admin terpenuhi.
- Mutasi dan login creation memakai PostgreSQL transaction advisory lock yang sama. Aktor dibaca ulang setelah lock; check last Super Admin dan perubahan user dilakukan secara atomik.
- Last active Super Admin tidak dapat dihapus, dinonaktifkan, atau diturunkan role-nya. Test concurrent memastikan dua request bersamaan tidak dapat menonaktifkan keduanya.
- Perubahan password/is_active merevoke session; role/permission dibaca ulang sehingga perubahan permission langsung berlaku di request berikutnya.
- Referential conflicts dikembalikan sebagai 409. Tidak ada cascade yang menghilangkan historical business records.


## Phase 3 report permission matrix

| Capability | Permission |
| --- | --- |
| Report list/detail, type filter options | report.view |
| Dashboard metrics/recent reports | report.view |
| Private attachment download | report.view |
| Status change/rejection/reactivation | report.update + CSRF |
| Internal/external notes-only | report.update + CSRF |

Permissions are independent. API writer-only roles cannot read report detail, contact, notes or attachment and receive only id/updated_at acknowledgment. UI detail page requires report.view; action controls additionally check report.update. No report-type permission hierarchy, ownership/assignment filtering or implicit master.manage override. Super Admin effective bypass remains. Actor is_active and permissions re-resolved after workflow locks; revoked/expired sessions rejected at request entry. Existing report.view/report.update seeds reused, no new permission names.
