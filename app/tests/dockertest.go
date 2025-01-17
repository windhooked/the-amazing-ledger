package tests

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/ory/dockertest/v3"
)

type PostgresDocker struct {
	DB       *pgxpool.Pool
	Pool     *dockertest.Pool
	Resource *dockertest.Resource
}

func SetupTest(migrationsPath string) *PostgresDocker {
	var conn *pgxpool.Pool

	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(fmt.Errorf("failed to connect to docker: %w", err))
	}

	database := "dev"

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13.2",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=" + database,
		},
		// ...so there are no differences in time between host and vm
		Mounts: []string{"/etc/localtime:/etc/localtime"},
	})
	if err != nil {
		panic(fmt.Errorf("failed to start resource: %w", err))
	}

	connString := fmt.Sprintf(
		"postgres://postgres:postgres@localhost:%s/%s?sslmode=disable",
		resource.GetPort("5432/tcp"),
		database)

	if err = pool.Retry(func() error {
		var rErr error
		ctx := context.Background()
		conn, rErr = pgxpool.Connect(ctx, connString)
		if rErr != nil {
			return fmt.Errorf("failed to retry pgxpool connect: %w", rErr)
		}
		_, rErr = conn.Acquire(ctx)
		if rErr != nil {
			return fmt.Errorf("failed to acquire pgxpool connection: %w", rErr)
		}

		return nil
	}); err != nil {
		panic(fmt.Errorf("failed to connect to docker: %w", err))
	}

	if err := runMigrations(migrationsPath, connString); err != nil {
		panic(fmt.Errorf("failed to run migrations: %w", err))
	}

	return &PostgresDocker{
		DB:       conn,
		Pool:     pool,
		Resource: resource,
	}
}

func RemoveContainer(pgDocker *PostgresDocker) {
	if err := pgDocker.Pool.Purge(pgDocker.Resource); err != nil {
		panic(fmt.Errorf("failed to remove container: %w", err))
	}
}

func TruncateTables(ctx context.Context, db *pgxpool.Pool, tables ...string) {
	if _, err := db.Exec(ctx, "truncate "+strings.Join(tables, ", ")); err != nil {
		panic(fmt.Errorf("failed to truncate table(s) %v: %w", tables, err))
	}
}

func runMigrations(migrationsPath, connString string) error {
	if migrationsPath != "" {
		mig, err := migrate.New("file://"+migrationsPath, connString)
		if err != nil {
			return fmt.Errorf("failed to start migrate struct: %w", err)
		}
		defer mig.Close()
		if err = mig.Up(); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return nil
}
