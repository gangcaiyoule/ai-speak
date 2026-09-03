package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

var filenamePattern = regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.sql$`)

type Store interface {
	Prepare(context.Context) error
	AppliedVersions(context.Context) (map[int]bool, error)
	Apply(context.Context, Migration) error
}
type Migration struct {
	Version   int
	Name, SQL string
}

func Load(source fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, fmt.Errorf("read migration directory: %w", err)
	}
	var out []Migration
	seen := map[int]bool{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		m := filenamePattern.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("invalid migration filename %q", e.Name())
		}
		v, _ := strconv.Atoi(m[1])
		if v == 0 || seen[v] {
			return nil, fmt.Errorf("duplicate migration version %04d", v)
		}
		b, err := fs.ReadFile(source, e.Name())
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			return nil, fmt.Errorf("migration %q is empty", e.Name())
		}
		seen[v] = true
		out = append(out, Migration{v, m[2], string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}
func Run(ctx context.Context, store Store, ms []Migration) error {
	if store == nil {
		return fmt.Errorf("migration store is nil")
	}
	if err := store.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare migrations: %w", err)
	}
	applied, err := store.AppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}
	for _, m := range ms {
		if applied[m.Version] {
			continue
		}
		if err := store.Apply(ctx, m); err != nil {
			return fmt.Errorf("apply migration %04d_%s: %w", m.Version, m.Name, err)
		}
	}
	return nil
}
