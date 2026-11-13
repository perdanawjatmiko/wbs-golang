package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
	"wbs-golang/backend/migrations"
)

func Run(ctx context.Context, dsn, direction string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return errors.New("invalid migration database configuration")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(time.Minute)
	if err = db.PingContext(ctx); err != nil {
		return errors.New("migration database unavailable")
	}
	driver, err := pgx.WithInstance(db, &pgx.Config{StatementTimeout: 30 * time.Second})
	if err != nil {
		return errors.New("migration driver initialization failed")
	}
	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
	if err != nil {
		return errors.New("migration initialization failed")
	}
	defer m.Close()
	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		return fmt.Errorf("migration direction must be up or down")
	}
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	if err != nil {
		return errors.New("migration failed; inspect schema version/dirty state using a trusted database console")
	}
	return nil
}
