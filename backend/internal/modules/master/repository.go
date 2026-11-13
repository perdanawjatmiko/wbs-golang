package master

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/problem"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool} }

// SQL identifiers below originate only from the static entity catalog, never request input.
func selection(e Entity, public bool) string {
	if public {
		if e.Path == "report-types" {
			return `jsonb_build_object('id',t.id,'name',t.name,'description',t.description)`
		}
		return `jsonb_build_object('id',t.id,'report_type_id',t.report_type_id,'question',t.question,'answer',t.answer,'sort_order',t.sort_order)`
	}
	if e.Path == "users" {
		return `to_jsonb(t)-'password_hash'`
	}
	if e.Path == "roles" {
		return `to_jsonb(t)||jsonb_build_object('permission_ids',COALESCE((SELECT jsonb_agg(rp.permission_id ORDER BY rp.permission_id) FROM role_permissions rp WHERE rp.role_id=t.id),'[]'::jsonb))`
	}
	return `to_jsonb(t)`
}
func (r *Repository) List(ctx context.Context, e Entity, q Query) (Page, error) {
	args := []any{}
	where := []string{"true"}
	param := func(v any) string { args = append(args, v); return fmt.Sprintf("$%d", len(args)) }
	if q.Search != "" {
		p := param("%" + q.Search + "%")
		parts := []string{}
		for _, column := range e.Search {
			parts = append(parts, "t."+column+" ILIKE "+p)
		}
		where = append(where, "("+strings.Join(parts, " OR ")+")")
	}
	if q.Public {
		where = append(where, "t.is_active")
		if e.Path == "faqs" {
			where = append(where, `(t.report_type_id IS NULL OR EXISTS(SELECT 1 FROM report_types rt WHERE rt.id=t.report_type_id AND rt.is_active))`)
		}
	} else if q.Active != "" && (e.Path == "users" || e.Path == "report-types" || e.Path == "faqs") {
		where = append(where, "t.is_active="+param(q.Active == "true"))
	}
	if e.Path == "faqs" && q.ReportType != "" {
		if q.ReportType == "general" {
			where = append(where, "t.report_type_id IS NULL")
		} else {
			where = append(where, "t.report_type_id="+param(q.ReportType)+"::uuid")
		}
	}
	clause := strings.Join(where, " AND ")
	result := Page{Items: []json.RawMessage{}, Page: q.Page, PageSize: q.Size}
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM "+e.Table+" t WHERE "+clause, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	order := "t.id"
	if e.Path == "faqs" {
		order = "t.sort_order,t.id"
	}
	limit := param(q.Size)
	offset := param((q.Page - 1) * q.Size)
	rows, err := r.pool.Query(ctx, "SELECT "+selection(e, q.Public)+" FROM "+e.Table+" t WHERE "+clause+" ORDER BY "+order+" LIMIT "+limit+" OFFSET "+offset, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return result, err
		}
		result.Items = append(result.Items, json.RawMessage(raw))
	}
	return result, rows.Err()
}
func get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, e Entity, id string) (json.RawMessage, error) {
	var raw []byte
	err := q.QueryRow(ctx, "SELECT "+selection(e, false)+" FROM "+e.Table+" t WHERE t.id=$1", id).Scan(&raw)
	return raw, err
}
func (r *Repository) Get(ctx context.Context, e Entity, id string) (json.RawMessage, error) {
	return get(ctx, r.pool, e, id)
}
func write(ctx context.Context, tx pgx.Tx, e Entity, id string, values Values) (string, error) {
	columns := []string{}
	args := []any{}
	for _, field := range e.Fields {
		name := field.Name
		if name == "permission_ids" {
			continue
		}
		if name == "password" {
			name = "password_hash"
		}
		if v, ok := values[name]; ok {
			columns = append(columns, name)
			args = append(args, v)
		}
	}
	if id == "" {
		places := []string{}
		for i := range args {
			places = append(places, fmt.Sprintf("$%d", i+1))
		}
		err := tx.QueryRow(ctx, "INSERT INTO "+e.Table+"("+strings.Join(columns, ",")+") VALUES("+strings.Join(places, ",")+") RETURNING id::text", args...).Scan(&id)
		return id, err
	}
	if len(columns) > 0 {
		sets := []string{}
		for i, column := range columns {
			sets = append(sets, fmt.Sprintf("%s=$%d", column, i+1))
		}
		if e.Updated {
			sets = append(sets, "updated_at=now()")
		}
		args = append(args, id)
		_, err := tx.Exec(ctx, "UPDATE "+e.Table+" SET "+strings.Join(sets, ",")+fmt.Sprintf(" WHERE id=$%d", len(args)), args...)
		if err != nil {
			return "", err
		}
	}
	return id, nil
}

// Transaction owns persistence operations while the service enforces business guards.
type Transaction struct{ tx pgx.Tx }

func (r *Repository) Begin(ctx context.Context) (*Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, auth.AdminLock); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return &Transaction{tx}, nil
}
func (t *Transaction) Commit(ctx context.Context) error   { return t.tx.Commit(ctx) }
func (t *Transaction) Rollback(ctx context.Context) error { return t.tx.Rollback(ctx) }
func (t *Transaction) Principal(ctx context.Context, id string) (auth.Principal, error) {
	return auth.Resolve(ctx, t.tx, id)
}
func (t *Transaction) Get(ctx context.Context, e Entity, id string) (json.RawMessage, error) {
	return get(ctx, t.tx, e, id)
}
func (t *Transaction) Save(ctx context.Context, e Entity, id string, v Values) (string, error) {
	return write(ctx, t.tx, e, id, v)
}
func (t *Transaction) Delete(ctx context.Context, e Entity, id string) error {
	_, err := t.tx.Exec(ctx, "DELETE FROM "+e.Table+" WHERE id=$1", id)
	return err
}
func (t *Transaction) Assign(ctx context.Context, id string, ids []string) error {
	if _, err := t.tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id=$1`, id); err != nil {
		return err
	}
	for _, permission := range ids {
		if _, err := t.tx.Exec(ctx, `INSERT INTO role_permissions(role_id,permission_id) VALUES($1,$2)`, id, permission); err != nil {
			return err
		}
	}
	return nil
}
func (t *Transaction) Revoke(ctx context.Context, id string) error {
	_, err := t.tx.Exec(ctx, `UPDATE admin_sessions SET revoked_at=now() WHERE admin_id=$1 AND revoked_at IS NULL`, id)
	return err
}
func (t *Transaction) UserRole(ctx context.Context, id string) (string, bool, error) {
	var role string
	var active bool
	err := t.tx.QueryRow(ctx, `SELECT r.name,u.is_active FROM admin_users u JOIN roles r ON r.id=u.role_id WHERE u.id=$1`, id).Scan(&role, &active)
	return role, active, err
}
func (t *Transaction) RoleName(ctx context.Context, id string) (string, error) {
	var name string
	err := t.tx.QueryRow(ctx, `SELECT name FROM roles WHERE id=$1`, id).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", problem.New(409, "REFERENCE_CONFLICT", "Role tidak valid.")
	}
	return name, err
}
func (t *Transaction) SuperCount(ctx context.Context) (int, error) {
	var count int
	err := t.tx.QueryRow(ctx, `SELECT count(*) FROM admin_users u JOIN roles r ON r.id=u.role_id WHERE r.name='Super Admin' AND u.is_active`).Scan(&count)
	return count, err
}
