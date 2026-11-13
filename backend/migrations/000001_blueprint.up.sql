BEGIN;
CREATE TABLE roles (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name varchar(100) NOT NULL UNIQUE CHECK (btrim(name) <> ''),
 description text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE permissions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), slug varchar(100) NOT NULL UNIQUE CHECK (slug ~ '^[a-z][a-z-]*[.][a-z][a-z-]*$'),
 module_name varchar(100) NOT NULL CHECK (btrim(module_name) <> ''), description text NOT NULL DEFAULT ''
);
CREATE TABLE role_permissions (
 role_id uuid NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
 permission_id uuid NOT NULL REFERENCES permissions(id) ON DELETE RESTRICT,
 PRIMARY KEY(role_id, permission_id)
);
CREATE INDEX role_permissions_permission_idx ON role_permissions(permission_id);
CREATE TABLE admin_users (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), role_id uuid NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
 name varchar(150) NOT NULL CHECK (btrim(name) <> ''), email varchar(254) NOT NULL CHECK (btrim(email) <> ''),
 password_hash varchar(255) NOT NULL CHECK (btrim(password_hash) <> ''), is_active boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX admin_users_email_unique ON admin_users(lower(email));
CREATE INDEX admin_users_role_idx ON admin_users(role_id);
CREATE TABLE report_types (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name varchar(150) NOT NULL CHECK (btrim(name) <> ''),
 description text NOT NULL DEFAULT '', is_active boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE faqs (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), report_type_id uuid REFERENCES report_types(id) ON DELETE RESTRICT,
 question text NOT NULL CHECK (btrim(question) <> ''), answer text NOT NULL CHECK (btrim(answer) <> ''),
 sort_order integer NOT NULL DEFAULT 0 CHECK (sort_order >= 0), is_active boolean NOT NULL DEFAULT true
);
CREATE INDEX faqs_report_type_idx ON faqs(report_type_id);
CREATE INDEX faqs_order_idx ON faqs(sort_order,id);
CREATE TABLE email_templates (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), event_key varchar(100) NOT NULL UNIQUE CHECK (event_key ~ '^[A-Z][A-Z0-9_]*$'),
 subject varchar(255) NOT NULL CHECK (btrim(subject) <> ''), body_html text NOT NULL CHECK (btrim(body_html) <> ''),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE reports (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), code varchar(100) NOT NULL UNIQUE CHECK (btrim(code) <> ''),
 name varchar(150) NOT NULL CHECK (btrim(name) <> ''), contact varchar(254) NOT NULL CHECK (btrim(contact) <> ''),
 contact_type varchar(10) NOT NULL CHECK (contact_type IN ('EMAIL','PHONE')),
 report_type_id uuid NOT NULL REFERENCES report_types(id) ON DELETE RESTRICT, detail text NOT NULL CHECK (btrim(detail) <> ''),
 attachment_url varchar(2048), status varchar(20) NOT NULL DEFAULT 'Open' CHECK (status IN ('Open','On Progress','Done','Rejected')),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reports_contact_idx ON reports(contact);
CREATE INDEX reports_status_idx ON reports(status);
CREATE INDEX reports_report_type_idx ON reports(report_type_id);
CREATE TABLE report_histories (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), report_id uuid NOT NULL REFERENCES reports(id) ON DELETE RESTRICT,
 admin_id uuid NOT NULL REFERENCES admin_users(id) ON DELETE RESTRICT,
 old_status varchar(20) NOT NULL CHECK (old_status IN ('Open','On Progress','Done','Rejected')),
 new_status varchar(20) NOT NULL CHECK (new_status IN ('Open','On Progress','Done','Rejected')),
 internal_note text NOT NULL DEFAULT '', external_note text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX report_histories_report_idx ON report_histories(report_id);
CREATE INDEX report_histories_admin_idx ON report_histories(admin_id);
COMMIT;
