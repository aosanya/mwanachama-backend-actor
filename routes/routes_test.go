package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func newManager(t *testing.T) mwanachamaactor.UserManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaactor.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaactor.NewUserManager(db, s)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return mgr
}

func newServer(t *testing.T, table []routes.Route) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range table {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, srv.URL+path, reader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestTheDeclaredTableBuilds(t *testing.T) {
	table, err := routes.Build(newManager(t))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(table) != 47 {
		t.Errorf("the declared table has %d routes, want 47", len(table))
	}
	for _, rt := range table {
		if rt.Action == "" {
			t.Errorf("%s %s carries no action, so a mount has nothing to gate on", rt.Method, rt.Path)
		}
		if rt.Handler == nil {
			t.Errorf("%s %s has no handler", rt.Method, rt.Path)
		}
	}
}

func TestShapeMatchesTheBuiltTable(t *testing.T) {
	built, err := routes.Build(newManager(t))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	shape := routes.Shape()
	if len(shape) != len(built) {
		t.Fatalf("Shape has %d routes, the built table has %d", len(shape), len(built))
	}
	for i := range shape {
		if shape[i].Method != built[i].Method || shape[i].Path != built[i].Path || shape[i].Action != built[i].Action {
			t.Errorf("route %d: shape is %s %s (%s), built is %s %s (%s)", i,
				shape[i].Method, shape[i].Path, shape[i].Action,
				built[i].Method, built[i].Path, built[i].Action)
		}
	}
}

func TestEveryToolComesFromTheSameDeclaration(t *testing.T) {
	tools, err := routes.BuildTools(newManager(t))
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	if len(tools) != 47 {
		t.Errorf("the declared table publishes %d tools, want 47", len(tools))
	}
	seen := map[string]bool{}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Errorf("a tool for %q has no name", tool.Action)
			continue
		}
		if seen[tool.Name] {
			t.Errorf("tool %q is published twice", tool.Name)
		}
		seen[tool.Name] = true
	}
}

func TestCreateActorOverTheWire(t *testing.T) {
	srv := newServer(t, routes.Routes(newManager(t)))

	status, body := do(t, srv, http.MethodPost, "/actors", map[string]any{
		"display_name": "Amina",
		"attributes":   map[string]any{"phone": "+15550001111"},
	})
	if status != http.StatusCreated {
		t.Fatalf("POST /actors = %d, want 201 (body %v)", status, body)
	}
	if body["display_name"] != "Amina" {
		t.Errorf("display_name = %v, want Amina", body["display_name"])
	}
	if body["code"] != "AC-1" {
		t.Errorf("code = %v, want AC-1", body["code"])
	}
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatal("the server minted no id")
	}

	status, read := do(t, srv, http.MethodGet, "/actors/"+id, nil)
	if status != http.StatusOK {
		t.Fatalf("GET /actors/%s = %d, want 200", id, status)
	}
	if read["id"] != id {
		t.Errorf("read back id %v, want %v", read["id"], id)
	}
}

func TestPinsACT4_CreateActorIgnoresCallerSuppliedServerOwnedFields(t *testing.T) {
	srv := newServer(t, routes.Routes(newManager(t)))

	status, body := do(t, srv, http.MethodPost, "/actors", map[string]any{
		"id":           "attacker-chosen-id",
		"created_at":   "1999-01-01T00:00:00Z",
		"deleted":      true,
		"display_name": "Ghost",
	})
	if status != http.StatusCreated {
		t.Fatalf("POST /actors = %d, want 201 (body %v)", status, body)
	}
	if body["id"] == "attacker-chosen-id" {
		t.Error("the caller chose the primary key, which is ACT4")
	}
	if body["created_at"] == "1999-01-01T00:00:00Z" {
		t.Error("the caller backdated created_at, which is ACT4")
	}
	if body["deleted"] == true {
		t.Error("the caller minted an already-deleted actor, which is ACT4")
	}

	status, _ = do(t, srv, http.MethodGet, "/actors/attacker-chosen-id", nil)
	if status != http.StatusNotFound {
		t.Errorf("GET /actors/attacker-chosen-id = %d, want 404 — the forged key must address nothing", status)
	}
}

func TestPinsACT4_CreateGroupIgnoresCallerSuppliedID(t *testing.T) {
	mgr := newManager(t)
	srv := newServer(t, routes.Routes(mgr))

	h, err := mgr.CreateHierarchy(context.Background(), mwanachamaactor.Hierarchy{Name: "Governance"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	lvl, err := mgr.CreateLevel(context.Background(), mwanachamaactor.Level{HierarchyID: h.ID, Name: "Branch"})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	status, body := do(t, srv, http.MethodPost, "/groups", map[string]any{
		"id":           "attacker-group-id",
		"hierarchy_id": h.ID,
		"level_id":     lvl.ID,
		"name":         "Ghost Group",
	})
	if status != http.StatusCreated {
		t.Fatalf("POST /groups = %d, want 201 (body %v)", status, body)
	}
	if body["id"] == "attacker-group-id" {
		t.Error("the caller chose the group's primary key, which is ACT4")
	}
}

func TestAGatedMountRefusesEveryActionItDoesNotAllow(t *testing.T) {
	allowed := "actor.actor.list"
	authorize := func(_ context.Context, action string) error {
		if action == allowed {
			return nil
		}
		return fmt.Errorf("refused")
	}
	srv := newServer(t, routes.RoutesFor(newManager(t), routes.Mount{
		Authorize:       authorize,
		CheckReferences: true,
	}))

	if status, _ := do(t, srv, http.MethodGet, "/actors", nil); status != http.StatusOK {
		t.Errorf("GET /actors = %d, want 200 — it is the one allowed action", status)
	}
	status, _ := do(t, srv, http.MethodPost, "/actors", map[string]any{"display_name": "Amina"})
	if status != http.StatusForbidden {
		t.Errorf("POST /actors = %d, want 403 — an action the mount does not allow", status)
	}
}

func TestRetireRoleKindTakesItsActorFromTheCallerNotTheBody(t *testing.T) {
	mgr := newManager(t)
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return "caller-actor" },
		CheckReferences: true,
	}))

	kind, err := mgr.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Chair"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}

	status, body := do(t, srv, http.MethodPost, "/role-kinds/"+kind.ID+"/retire", map[string]any{
		"actor_id": "body-supplied-actor",
	})
	if status != http.StatusOK {
		t.Fatalf("retire = %d, want 200 (body %v)", status, body)
	}
	if body["retired_by"] != "caller-actor" {
		t.Errorf("retired_by = %v, want caller-actor — the session decides, not the body", body["retired_by"])
	}
}

func TestAnUnknownSentinelIsNotLeakedAsAFiveHundredBody(t *testing.T) {
	srv := newServer(t, routes.Routes(newManager(t)))

	status, body := do(t, srv, http.MethodGet, "/actors/nobody", nil)
	if status != http.StatusNotFound {
		t.Fatalf("GET /actors/nobody = %d, want 404 (body %v)", status, body)
	}
}

func TestEveryDeclaredActionIsUniqueAndNamespaced(t *testing.T) {
	seen := map[string]bool{}
	for _, rt := range routes.Shape() {
		if !dispatch.ActionPattern.MatchString(rt.Action) {
			t.Errorf("%s %s: action %q is not <module>.<resource>.<verb>", rt.Method, rt.Path, rt.Action)
		}
		if seen[rt.Action] {
			t.Errorf("action %q is claimed twice", rt.Action)
		}
		seen[rt.Action] = true
	}
}
