package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/jackc/pgx/v4"
)

//go:embed migration/*.sql
var migrationFS embed.FS

type DB struct {
	db *pgx.Conn

	// Datasource name.
	connStr string

	// Now returns the current time.
	// Used to ensure a consistent time value for multiple inserts/updates in a single transaction
	now func() time.Time
}

// NewDB returns a new instance of DB associated with the given datasource name.
func NewDB(connStr string) *DB {
	db := &DB{
		connStr: connStr,
		now:     time.Now,
	}

	return db
}

// Open opens the database connection.
func (db *DB) Open(ctx context.Context) (err error) {
	// Ensure a DSN is set before attempting to open the database.
	if db.connStr == "" {
		return fmt.Errorf("db connection string required")
	}

	// Connect to the database.
	if db.db, err = pgx.Connect(ctx, db.connStr); err != nil {
		return err
	}

	if err := db.migrate(ctx); err != nil {
		return fmt.Errorf("error whilst migrating: %w", err)
	}

	return nil
}

func (db *DB) migrate(ctx context.Context) error {
	// Ensure the 'migrations' table exists so we don't duplicate migrations.
	if _, err := db.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY);`); err != nil {
		return fmt.Errorf("cannot create migrations table: %w", err)
	}

	names, err := fs.Glob(migrationFS, "migration/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	// Loop over all migration files and execute them in order.
	for _, name := range names {
		if err := db.migrateFile(ctx, name); err != nil {
			return fmt.Errorf("migration error: name=%q err=%w", name, err)
		}
	}
	return nil
}

func (db *DB) migrateFile(ctx context.Context, name string) error {
	tx, err := db.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Ensure migration has not already been run.
	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM migrations WHERE name = $1`, name).Scan(&n); err != nil {
		return err
	} else if n != 0 {
		fmt.Println("no migrations to run")
		return nil
	}

	// Read and execute migration file.
	if buf, err := fs.ReadFile(migrationFS, name); err != nil {
		return err
	} else if _, err := tx.Exec(ctx, string(buf)); err != nil {
		return err
	}

	// Insert record into migrations to prevent re-running migration.
	if _, err := tx.Exec(ctx, `INSERT INTO migrations (name) VALUES ($1)`, name); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (db *DB) Close(ctx context.Context) error {
	if db.db != nil {
		return db.db.Close(ctx)
	}
	return nil
}

type Tx struct {
	pgx.Tx
	db  *DB
	now time.Time
}

func (db *DB) BeginTx(ctx context.Context) (*Tx, error) {
	tx, err := db.db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Tx{
		Tx:  tx,
		db:  db,
		now: db.now().UTC().Truncate(time.Second),
	}, nil
}
