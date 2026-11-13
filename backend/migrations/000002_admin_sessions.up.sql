BEGIN;
CREATE TABLE admin_sessions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), admin_id uuid NOT NULL REFERENCES admin_users(id) ON DELETE RESTRICT,
 token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash)=32),
 expires_at timestamptz NOT NULL, revoked_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(),
 CHECK (expires_at > created_at)
);
CREATE INDEX admin_sessions_admin_idx ON admin_sessions(admin_id);
CREATE INDEX admin_sessions_expiry_idx ON admin_sessions(expires_at);
COMMIT;
