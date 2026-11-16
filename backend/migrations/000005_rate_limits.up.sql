BEGIN;
-- Fixed bucket slots bound cardinality without storing IP/contact/credential data.
CREATE TABLE rate_limit_buckets (
 scope varchar(20) NOT NULL CHECK(scope IN ('admin_login','report_submit','report_track')),
 bucket integer NOT NULL CHECK(bucket BETWEEN 0 AND 65535),
 hits integer NOT NULL CHECK(hits BETWEEN 1 AND 10001),
 expires_at timestamptz NOT NULL,
 PRIMARY KEY(scope,bucket)
);
COMMIT;
