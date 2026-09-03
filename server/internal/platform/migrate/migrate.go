// Package migrate provides deterministic migration planning and execution.
package migrate

import (
	"context"
	"fmt"
	"sort"
)

// Executor is the database operation required by the migration runner.
type Executor interface {
	ExecContext(context.Context, string, ...any) error
	AppliedMigrations(context.Context) (map[int]bool, error)
}

// Migration is one ordered schema change.
type Migration struct { Version int; Name, SQL string }

// Runner applies pending migrations in ascending version order.
type Runner struct { Migrations []Migration }

// Run creates bookkeeping through the executor and applies each pending migration.
func (r Runner) Run(ctx context.Context, db Executor) error {
	if db == nil { return fmt.Errorf("migration executor is nil") }
	applied, err := db.AppliedMigrations(ctx)
	if err != nil { return fmt.Errorf("read applied migrations: %w", err) }
	migrations := append([]Migration(nil), r.Migrations...)
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	for _, m := range migrations {
		if m.Version < 1 || m.Name == "" || m.SQL == "" { return fmt.Errorf("invalid migration %d", m.Version) }
		if applied[m.Version] { continue }
		if err := db.ExecContext(ctx, m.SQL); err != nil { return fmt.Errorf("apply migration %d (%s): %w", m.Version, m.Name, err) }
		if err := db.ExecContext(ctx, "INSERT INTO schema_migrations(version, name) VALUES ($1, $2)", m.Version, m.Name); err != nil { return fmt.Errorf("record migration %d: %w", m.Version, err) }
	}
	return nil
}
