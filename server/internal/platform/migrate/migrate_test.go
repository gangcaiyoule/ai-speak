package migrate

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

type fakeStore struct {
	applied []int
	fail    bool
}

func (*fakeStore) Prepare(context.Context) error { return nil }
func (*fakeStore) AppliedVersions(context.Context) (map[int]bool, error) {
	return map[int]bool{1: true}, nil
}
func (s *fakeStore) Apply(_ context.Context, m Migration) error {
	if s.fail {
		return errors.New("database unavailable")
	}
	s.applied = append(s.applied, m.Version)
	return nil
}
func TestLoadSorts(t *testing.T) {
	m, e := Load(fstest.MapFS{"0002_second.sql": {Data: []byte("SQL2")}, "0001_first.sql": {Data: []byte("SQL1")}})
	if e != nil || len(m) != 2 || m[0].Version != 1 {
		t.Fatalf("m=%v e=%v", m, e)
	}
}
func TestRunSkipsApplied(t *testing.T) {
	s := &fakeStore{}
	if e := Run(context.Background(), s, []Migration{{1, "first", "SQL1"}, {2, "second", "SQL2"}}); e != nil || len(s.applied) != 1 || s.applied[0] != 2 {
		t.Fatalf("s=%v e=%v", s, e)
	}
}
func TestRunFailure(t *testing.T) {
	e := Run(context.Background(), &fakeStore{fail: true}, []Migration{{2, "broken", "SQL"}})
	if e == nil || !strings.Contains(e.Error(), "0002_broken") {
		t.Fatal(e)
	}
}
