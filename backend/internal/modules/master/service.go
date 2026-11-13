package master

import (
	"context"
	"encoding/json"
	"time"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
)

type Service struct {
	repo    *Repository
	timeout time.Duration
}

func NewService(r *Repository, timeout time.Duration) *Service { return &Service{r, timeout} }
func (s *Service) List(ctx context.Context, e Entity, q Query) (Page, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.repo.List(ctx, e, q)
}
func (s *Service) Get(ctx context.Context, e Entity, id string) (json.RawMessage, error) {
	if !security.UUID.MatchString(id) {
		return nil, invalid("ID harus UUID.")
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.repo.Get(ctx, e, id)
}
func authorize(p auth.Principal, e Entity) error {
	if (e.SuperOnly && !p.Super()) || !(p.Can(e.Permission) || (!e.SuperOnly && p.Can("master.manage"))) {
		return problem.New(403, "FORBIDDEN", "Akses tidak diizinkan.")
	}
	return nil
}
func (s *Service) Mutate(ctx context.Context, actor auth.Principal, e Entity, id string, input Input, remove bool) (json.RawMessage, error) {
	if id != "" && !security.UUID.MatchString(id) {
		return nil, invalid("ID harus UUID.")
	}
	values := Values{}
	var err error
	if !remove {
		values, err = Validate(e, input, id == "")
		if err != nil {
			return nil, err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	fresh, err := tx.Principal(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	if err = authorize(fresh, e); err != nil {
		return nil, err
	}
	existing := map[string]any{}
	if id != "" {
		raw, err := tx.Get(ctx, e, id)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &existing); err != nil {
			return nil, err
		}
	}
	if e.Path == "roles" && existing["name"] == "Super Admin" {
		if remove {
			return nil, problem.New(409, "PROTECTED_ROLE", "Role Super Admin dilindungi.")
		}
		if name, ok := values["name"]; ok && name != "Super Admin" {
			return nil, problem.New(409, "PROTECTED_ROLE", "Nama role Super Admin dilindungi.")
		}
	}
	if e.Path == "users" && id != "" {
		if id == fresh.ID {
			if _, ok := values["role_id"]; ok && values["role_id"] != existing["role_id"] {
				return nil, problem.New(403, "SELF_ROLE_CHANGE", "Role akun sendiri tidak dapat diubah.")
			}
		}
		if err = protectLast(ctx, tx, id, values, remove); err != nil {
			return nil, err
		}
	}
	if remove {

		if err = tx.Delete(ctx, e, id); err != nil {
			return nil, err
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
		return json.RawMessage(`{"deleted":true}`), nil
	}
	if e.Path == "email-templates" {
		if err = ValidateTemplate(values, existing); err != nil {
			return nil, err
		}
	}
	saved, err := tx.Save(ctx, e, id, values)
	if err != nil {
		return nil, err
	}
	if e.Path == "roles" {
		if ids, ok := values["permission_ids"].([]string); ok {
			if err = tx.Assign(ctx, saved, ids); err != nil {
				return nil, err
			}
		}
	}
	if e.Path == "users" && id != "" {
		_, password := values["password_hash"]
		inactive, changed := values["is_active"].(bool)
		if password || (changed && !inactive) {
			if err = tx.Revoke(ctx, id); err != nil {
				return nil, err
			}
		}
	}
	raw, err := tx.Get(ctx, e, saved)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return raw, nil
}
func protectLast(ctx context.Context, tx *Transaction, id string, values Values, remove bool) error {
	role, active, err := tx.UserRole(ctx, id)
	if err != nil {
		return err
	}
	if role != "Super Admin" || !active {
		return nil
	}
	leaving := remove
	if v, ok := values["is_active"].(bool); ok && !v {
		leaving = true
	}
	if roleID, ok := values["role_id"].(string); ok {
		name, err := tx.RoleName(ctx, roleID)
		if err != nil {
			return err
		}
		if name != "Super Admin" {
			leaving = true
		}
	}
	if leaving {
		n, err := tx.SuperCount(ctx)
		if err != nil {
			return err
		}
		if n <= 1 {
			return problem.New(409, "LAST_SUPER_ADMIN", "Super Admin aktif terakhir dilindungi.")
		}
	}
	return nil
}
