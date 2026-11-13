package seed

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/mail"
	"strings"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/security"
)

var Permissions = []string{"report.view", "report.update", "master.manage", "users.manage", "roles.manage", "permissions.manage", "report-types.manage", "faqs.manage", "email-templates.manage"}

type Admin struct{ Email, Password, Name string }

func Run(ctx context.Context, pool *pgxpool.Pool, admin Admin) error {
	configured := admin.Email != "" || admin.Password != "" || admin.Name != ""
	var hash string
	if configured {
		address, err := mail.ParseAddress(admin.Email)
		if err != nil || address.Address != admin.Email || strings.TrimSpace(admin.Name) == "" || len(admin.Name) > 150 {
			return fmt.Errorf("bootstrap requires valid ADMIN_SEED_EMAIL and ADMIN_SEED_NAME")
		}
		hash, err = security.HashPassword(admin.Password)
		if err != nil {
			return err
		}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, auth.AdminLock); err != nil {
		return err
	}
	for _, name := range []string{"Super Admin", "Investigator"} {
		if _, err = tx.Exec(ctx, `INSERT INTO roles(name) VALUES($1) ON CONFLICT(name) DO NOTHING`, name); err != nil {
			return err
		}
	}
	for _, slug := range Permissions {
		if _, err = tx.Exec(ctx, `INSERT INTO permissions(slug,module_name) VALUES($1,$2) ON CONFLICT(slug) DO NOTHING`, slug, strings.Split(slug, ".")[0]); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO role_permissions(role_id,permission_id) SELECT r.id,p.id FROM roles r CROSS JOIN permissions p WHERE r.name='Super Admin' ON CONFLICT DO NOTHING`); err != nil {
		return err
	}
	for _, key := range []string{"REPORT_CREATED", "REPORT_ON_PROGRESS", "REPORT_DONE", "REPORT_REJECTED"} {
		if _, err = tx.Exec(ctx, `INSERT INTO email_templates(event_key,subject,body_html) VALUES($1,$2,$3) ON CONFLICT(event_key) DO NOTHING`, key, "Laporan {{report_code}} — {{status}}", "<p>{{user_name}}</p><p>{{report_code}}: {{status}}</p><p>{{external_note}}</p>"); err != nil {
			return err
		}
	}
	if configured {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM admin_users WHERE lower(email)=lower($1))`, admin.Email).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			if _, err = tx.Exec(ctx, `INSERT INTO admin_users(role_id,name,email,password_hash) SELECT id,$1,lower($2),$3 FROM roles WHERE name='Super Admin'`, strings.TrimSpace(admin.Name), admin.Email, hash); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}
