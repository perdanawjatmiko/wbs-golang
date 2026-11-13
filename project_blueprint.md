# WBS (Whistleblowing System) & Ticketing Application Blueprint

This document serves as the master reference and technical specification guide for AI assistants collaborating on the development of the WBS/Ticketing application.

---

## 1. Tech Stack & Architecture

- **Backend:** Go (Golang) using a modern framework (e.g., Fiber, Gin, or Echo) with PostgreSQL/MySQL.
- **Frontend:** Next.js (App Router) with Tailwind CSS.
- **Communication Protocols:** RESTful APIs, Server-Sent Events (SSE) or WebSockets for real-time updates, SMTP for email delivery, and Telegram Bot API.

---

## 2. Complete Database Schema Specification

### A. Authentication & RBAC (Role-Based Access Control)
1. **`roles`**
   - `id` (PK, UUID / Auto-increment)
   - `name` (VARCHAR, e.g., "Super Admin", "Investigator")
   - `description` (TEXT)
   - `created_at`, `updated_at`

2. **`permissions`**
   - `id` (PK)
   - `slug` (VARCHAR, e.g., `report.view`, `report.update`, `master.manage`)
   - `module_name` (VARCHAR)
   - `description` (TEXT)

3. **`role_permissions`**
   - `role_id` (FK -> `roles.id`)
   - `permission_id` (FK -> `permissions.id`)

4. **`admin_users`**
   - `id` (PK)
   - `role_id` (FK -> `roles.id`)
   - `name` (VARCHAR)
   - `email` (VARCHAR, Unique)
   - `password_hash` (VARCHAR)
   - `is_active` (BOOLEAN)
   - `created_at`, `updated_at`

### B. Master Data (Dynamic Configuration)
5. **`report_types`**
   - `id` (PK)
   - `name` (VARCHAR, e.g., "Fraud", "Gratifikasi", "Pelanggaran Etik")
   - `description` (TEXT)
   - `is_active` (BOOLEAN)
   - `created_at`, `updated_at`

6. **`faqs`**
   - `id` (PK)
   - `report_type_id` (FK -> `report_types.id`, Nullable for General FAQs)
   - `question` (TEXT)
   - `answer` (TEXT)
   - `sort_order` (INT)
   - `is_active` (BOOLEAN)

7. **`email_templates`**
   - `id` (PK)
   - `event_key` (VARCHAR, Unique, e.g., `REPORT_CREATED`, `REPORT_ON_PROGRESS`, `REPORT_DONE`, `REPORT_REJECTED`)
   - `subject` (VARCHAR)
   - `body_html` (TEXT - supports placeholders like `{{report_code}}`, `{{user_name}}`, `{{status}}`, `{{external_note}}`)
   - `updated_at`

### C. Core Transactional Tables
8. **`reports`**
   - `id` (PK)
   - `code` (VARCHAR, Unique, format: `WBS-{DDMMYY}-{2RAND}{SEQ}`)
   - `name` (VARCHAR)
   - `contact` (VARCHAR - Email or Phone number stored in plain text)
   - `contact_type` (ENUM/VARCHAR: `EMAIL` or `PHONE`)
   - `report_type_id` (FK -> `report_types.id`)
   - `detail` (TEXT)
   - `attachment_url` (VARCHAR, Nullable)
   - `status` (ENUM: `Open`, `On Progress`, `Done`, `Rejected`)
   - `created_at`, `updated_at`

9. **`report_histories`**
   - `id` (PK)
   - `report_id` (FK -> `reports.id`)
   - `admin_id` (FK -> `admin_users.id`)
   - `old_status` (VARCHAR)
   - `new_status` (VARCHAR)
   - `internal_note` (TEXT - visible only to admins)
   - `external_note` (TEXT - visible to users during tracking)
   - `created_at`

---

## 3. Core Business Rules & Validation Logic

1. **Active Report Limit:**
   - A single credential (`contact` matching email or phone) can have a **maximum of 5 Active Reports** simultaneously.
   - Active status condition: `status NOT IN ('Done', 'Rejected')`.
   - Backend must validate this count before inserting a new report.

2. **Report Code Generation:**
   - Format: `WBS-{DDMMYY}-{2_RANDOM_CHARS}{SEQUENCE}` (Example: `WBS-141126-X9001`).
   - Must use database transactions and atomic counters to prevent race conditions or duplicate codes.

3. **Status Rejection Rule:**
   - If a report is marked as `Rejected` (e.g., Spam), the admin **must** provide an `external_note` explaining the reason. This reduces the active quota count for that user.

4. **Tracking Security:**
   - Users check report status using a combination of **Report Code + Contact (Email/Phone)**.
   - Backend query must verify both parameters to prevent IDOR vulnerabilities.

---

## 4. 5-Phase Development Roadmap

- **Fase 1: Master Data, RBAC & Database Setup**
  - Setup DB migrations and seeders.
  - Implement Admin Auth (JWT/HttpOnly Cookies) and RBAC middleware.
  - CRUD for Roles, Permissions, Admin Users, Report Types, FAQs, and Email Templates.
- **Fase 2: Public Reporting & Tracking (User Side)**
  - Public form with dynamic Report Types and interactive FAQs.
  - Report submission with active limit check (max 5) and code generator.
  - Secure tracking page showing timeline and external notes.
- **Fase 3: Admin Workflow & Ticket Management**
  - Admin dashboard with metrics and filtered list by permissions.
  - Status update workflow (`On Progress`, `Done`, `Rejected`) with `report_histories` logging (Internal & External notes).
- **Fase 4: Multi-Channel Notifications**
  - Async worker (Goroutine) for email dispatch using DB email templates.
  - Telegram bot integration for group alerts on new `Open` tickets.
  - Real-time in-app notification via SSE/WebSocket.
- **Fase 5: Security Hardening & Deployment**
  - Rate limiting, CAPTCHA integration, and file upload security validation.
  - Dockerization and production server configuration (Nginx + Go Binary + Next.js).