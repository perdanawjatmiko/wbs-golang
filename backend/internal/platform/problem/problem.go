package problem

import (
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string                   { return e.Code }
func New(status int, code, message string) error { return &Error{status, code, message} }
func Map(err error) *Error {
	var p *Error
	if errors.As(err, &p) {
		return p
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return &Error{404, "NOT_FOUND", "Data tidak ditemukan."}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return &Error{409, "DUPLICATE", "Data unik sudah digunakan."}
		case "23503", "23001":
			return &Error{409, "REFERENCE_CONFLICT", "Data direferensikan atau referensi tidak valid. Nonaktifkan data yang masih digunakan."}
		case "23514", "23502", "22P02", "22001":
			return &Error{422, "VALIDATION_ERROR", "Data tidak valid."}
		}
	}
	return &Error{500, "INTERNAL_ERROR", "Terjadi kesalahan internal."}
}
