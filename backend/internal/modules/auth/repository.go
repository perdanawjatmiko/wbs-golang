package auth

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

const AdminLock int64 = 761983221

// Querier lets effective permissions be re-resolved inside mutation transactions.
type Querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Principal struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	RoleID      string   `json:"role_id"`
	RoleName    string   `json:"role_name"`
	Permissions []string `json:"permissions"`
}

func (p Principal) Super() bool { return p.RoleName == "Super Admin" }
func (p Principal) Can(permission string) bool {
	if p.Super() {
		return true
	}
	for _, v := range p.Permissions {
		if v == permission {
			return true
		}
	}
	return false
}
func Resolve(ctx context.Context, q Querier, id string) (Principal, error) {
	var p Principal
	err := q.QueryRow(ctx, `SELECT u.id::text,u.name,u.email,u.role_id::text,r.name,COALESCE(array_agg(p.slug ORDER BY p.slug) FILTER(WHERE p.slug IS NOT NULL),'{}') FROM admin_users u JOIN roles r ON r.id=u.role_id LEFT JOIN role_permissions rp ON rp.role_id=r.id LEFT JOIN permissions p ON p.id=rp.permission_id WHERE u.id=$1 AND u.is_active GROUP BY u.id,r.id`, id).Scan(&p.ID, &p.Name, &p.Email, &p.RoleID, &p.RoleName, &p.Permissions)
	return p, err
}

type Repository struct{ Pool *pgxpool.Pool }

func NewRepository(p *pgxpool.Pool) *Repository { return &Repository{p} }
func (r *Repository) Credentials(ctx context.Context, email string) (id, hash string, active bool, err error) {
	err = r.Pool.QueryRow(ctx, `SELECT id::text,password_hash,is_active FROM admin_users WHERE lower(email)=lower($1)`, email).Scan(&id, &hash, &active)
	return
}
func (r *Repository) CreateSession(ctx context.Context, id, expectedHash string, hash, old []byte, expires time.Time) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, AdminLock); err != nil {
		return err
	}
	var active bool
	var currentHash string
	if err = tx.QueryRow(ctx, `SELECT is_active,password_hash FROM admin_users WHERE id=$1 FOR UPDATE`, id).Scan(&active, &currentHash); err != nil {
		return err
	}
	if !active || currentHash != expectedHash {
		return pgx.ErrNoRows
	}
	if len(old) > 0 {
		if _, err = tx.Exec(ctx, `UPDATE admin_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, old); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO admin_sessions(admin_id,token_hash,expires_at) VALUES($1,$2,$3)`, id, hash, expires)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) Session(ctx context.Context, hash []byte) (Principal, error) {
	var id string
	err := r.Pool.QueryRow(ctx, `SELECT s.admin_id::text FROM admin_sessions s JOIN admin_users u ON u.id=s.admin_id WHERE s.token_hash=$1 AND s.expires_at>now() AND s.revoked_at IS NULL AND u.is_active`, hash).Scan(&id)
	if err != nil {
		return Principal{}, err
	}
	return Resolve(ctx, r.Pool, id)
}
func (r *Repository) Revoke(ctx context.Context, hash []byte) error {
	_, err := r.Pool.Exec(ctx, `UPDATE admin_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, hash)
	return err
}
