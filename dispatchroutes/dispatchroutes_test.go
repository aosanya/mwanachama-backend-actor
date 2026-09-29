package dispatchroutes_test

import (
	"sort"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/dispatchroutes"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func newTestManager(t *testing.T) mwanachamaactor.UserManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := mwanachamaactor.DefaultTableNames("test")
	if err := mwanachamaactor.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mgr, err := mwanachamaactor.NewUserManager(db, tables)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return mgr
}

func TestEveryDeclaredOperationBindsToTheManager(t *testing.T) {
	built, err := dispatchroutes.Build(newTestManager(t), nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(built) == 0 {
		t.Fatal("Build returned no routes")
	}
	for _, rt := range built {
		if rt.Handler == nil {
			t.Errorf("%s %s has no handler", rt.Method, rt.Path)
		}
		if rt.Action == "" {
			t.Errorf("%s %s declares no action", rt.Method, rt.Path)
		}
	}
}

func TestDeclaredTableMatchesTheHandWrittenOne(t *testing.T) {
	declared := addresses(shapeAddresses())
	handWritten := addresses(handWrittenAddresses(newTestManager(t)))

	if len(declared) != len(handWritten) {
		t.Errorf("declared %d addresses, hand-written serves %d", len(declared), len(handWritten))
	}
	for i := range declared {
		if i >= len(handWritten) {
			t.Errorf("declared but not served: %s", declared[i])
			continue
		}
		if declared[i] != handWritten[i] {
			t.Errorf("address %d: declared %q, hand-written %q", i, declared[i], handWritten[i])
		}
	}
	for i := len(declared); i < len(handWritten); i++ {
		t.Errorf("served but not declared: %s", handWritten[i])
	}
}

func shapeAddresses() []string {
	var out []string
	for _, rt := range dispatchroutes.Shape() {
		out = append(out, rt.Method+" "+rt.Path)
	}
	return out
}

func handWrittenAddresses(um mwanachamaactor.UserManager) []string {
	var out []string
	for _, rt := range routes.Routes(um, nil, routes.ResourceNames{}) {
		out = append(out, rt.Method+" "+rt.Path)
	}
	return out
}

func addresses(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
