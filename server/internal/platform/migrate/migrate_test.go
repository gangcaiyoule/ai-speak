package migrate

import ("context"; "errors"; "strings"; "testing")
type fakeDB struct { applied map[int]bool; statements []string; fail bool }
func (d *fakeDB) AppliedMigrations(context.Context) (map[int]bool,error) { return d.applied,nil }
func (d *fakeDB) ExecContext(_ context.Context, q string, _ ...any) error { d.statements=append(d.statements,q); if d.fail{return errors.New("database unavailable")}; return nil }
func TestRunnerSortsAndSkipsApplied(t *testing.T){ d:=&fakeDB{applied:map[int]bool{1:true}}; err:=(Runner{Migrations:[]Migration{{2,"second","SQL2"},{1,"first","SQL1"}}}).Run(context.Background(),d); if err!=nil{t.Fatal(err)}; if len(d.statements)!=2||!strings.Contains(d.statements[1],"schema_migrations"){t.Fatalf("statements=%v",d.statements)} }
func TestRunnerReportsFailure(t *testing.T){ d:=&fakeDB{applied:map[int]bool{},fail:true}; err:=(Runner{Migrations:[]Migration{{1,"first","SQL1"}}}).Run(context.Background(),d); if err==nil||!strings.Contains(err.Error(),"apply migration 1"){t.Fatalf("err=%v",err)} }
