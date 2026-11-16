package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func managerOnMemory(t *testing.T) actor.UserManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, err := actor.SpecFor("routescover")
	if err != nil {
		t.Fatal(err)
	}
	if err := actor.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	um, err := actor.NewUserManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	return um
}

func serverFor(t *testing.T, um actor.UserManager, mount routes.Mount) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(um, mount) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func idOf(t *testing.T, body []byte) string {
	t.Helper()
	var got struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &got); err != nil || got.ID == "" {
		t.Fatalf("no id in %s (%v)", body, err)
	}
	return got.ID
}

func TestTheRouteShapeIsSortedAndEveryRouteNamesItsAction(t *testing.T) {
	shape := routes.Shape()
	if len(shape) < 50 {
		t.Fatalf("only %d routes declared", len(shape))
	}
	seen := map[string]bool{}
	for _, r := range shape {
		if r.Method == "" || !strings.HasPrefix(r.Path, "/") || r.Action == "" {
			t.Errorf("incomplete route %+v", r)
		}
		key := r.Method + " " + r.Path
		if seen[key] {
			t.Errorf("route %s is declared twice", key)
		}
		seen[key] = true
	}
	again := routes.Shape()
	actions := make([]string, len(shape))
	for i, r := range shape {
		actions[i] = r.Action
		if again[i].Method != r.Method || again[i].Path != r.Path || again[i].Action != r.Action {
			t.Fatalf("the shape changes between calls at %d: %+v then %+v", i, r, again[i])
		}
	}
	for _, want := range []string{"actor.group.create", "actor.group.update", "actor.assignment.create", "actor.invitation.create"} {
		found := false
		for _, a := range actions {
			found = found || a == want
		}
		if !found {
			t.Errorf("the shape lacks %q", want)
		}
	}
}

func TestBuildAndRoutesAgreeWithTheShape(t *testing.T) {
	um := managerOnMemory(t)
	built, err := routes.Build(um)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(built) != len(routes.Shape()) || len(routes.Routes(um)) != len(built) {
		t.Errorf("Build %d, Routes %d, Shape %d routes", len(built), len(routes.Routes(um)), len(routes.Shape()))
	}
}

func TestCreatingAGroupChecksItsHierarchyLevelAndParent(t *testing.T) {
	um := managerOnMemory(t)
	srv := serverFor(t, um, routes.Mount{CheckReferences: true, Caller: func(context.Context) string { return "u-1" }})
	status, body := send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Rossz hierarchia", "hierarchy_id": "no-such"})
	if status < 400 || !strings.Contains(string(body), "does not exist") {
		t.Errorf("unknown hierarchy: %d %s", status, body)
	}
	h, err := um.CreateHierarchy(context.Background(), actor.Hierarchy{Name: "Közigazgatás"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	status, body = send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Rossz szint", "hierarchy_id": h.ID, "level_id": "no-level"})
	if status < 400 || !strings.Contains(string(body), "is not in that hierarchy") {
		t.Errorf("level outside the hierarchy: %d %s", status, body)
	}
	status, body = send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Árva csoport", "parent_id": "no-parent"})
	if status != http.StatusNotFound && status != http.StatusBadRequest && status != http.StatusUnprocessableEntity {
		t.Errorf("unknown parent: %d %s", status, body)
	}
	if !strings.Contains(strings.ToLower(string(body)), "parent") {
		t.Errorf("unknown parent must say it is the parent that is missing: %s", body)
	}
	status, body = send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Rendben"})
	if status >= 300 {
		t.Fatalf("a plain group: %d %s", status, body)
	}
	status, body = send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Gyerek", "parent_id": idOf(t, body)})
	if status >= 300 {
		t.Errorf("a group under an existing parent: %d %s", status, body)
	}
}

func TestWithoutReferenceChecksTheStoreIsLeftToJudge(t *testing.T) {
	um := managerOnMemory(t)
	srv := serverFor(t, um, routes.Mount{Caller: func(context.Context) string { return "u-1" }})
	status, body := send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Ellenőrzés nélkül", "hierarchy_id": "no-such"})
	if strings.Contains(string(body), "does not exist") {
		t.Errorf("the reference check ran although it was not switched on: %d %s", status, body)
	}
}

func TestEditingAGroupNeedsAShortNameAndAnExistingAnchorLevel(t *testing.T) {
	um := managerOnMemory(t)
	srv := serverFor(t, um, routes.Mount{CheckReferences: true, Caller: func(context.Context) string { return "u-1" }})
	status, body := send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Régi név"})
	if status >= 300 {
		t.Fatalf("create: %d %s", status, body)
	}
	id := idOf(t, body)
	status, body = send(t, http.MethodPatch, srv.URL+"/groups/"+id, map[string]any{"name": "   "})
	if status < 400 || !strings.Contains(string(body), "a name is required") {
		t.Errorf("a blank name: %d %s", status, body)
	}
	status, body = send(t, http.MethodPatch, srv.URL+"/groups/"+id, map[string]any{"name": strings.Repeat("é", 201)})
	if status < 400 || !strings.Contains(string(body), "at most 200 characters") {
		t.Errorf("a 201 character name: %d %s", status, body)
	}
	status, body = send(t, http.MethodPatch, srv.URL+"/groups/"+id, map[string]any{"name": "Anker", "anchor_level_override": "no-level"})
	if status < 400 {
		t.Errorf("an unknown anchor level: %d %s", status, body)
	}
	status, body = send(t, http.MethodPatch, srv.URL+"/groups/"+id, map[string]any{"name": "  Új név  "})
	if status >= 300 || !strings.Contains(string(body), "Új név") {
		t.Errorf("a valid rename: %d %s", status, body)
	}
	g, _ := um.GetGroup(context.Background(), id)
	if g.Name != "Új név" {
		t.Errorf("stored name = %q, want it trimmed", g.Name)
	}
}

func TestAssigningAnActorToAnUnknownGroupIsRefusedBeforeTheStoreIsAsked(t *testing.T) {
	um := managerOnMemory(t)
	srv := serverFor(t, um, routes.Mount{CheckReferences: true, Caller: func(context.Context) string { return "u-1" }})
	a, _ := um.CreateActor(context.Background(), actor.Actor{DisplayName: "Nagy Nóra"})
	status, body := send(t, http.MethodPost, srv.URL+"/actors/"+a.ID+"/assignments", map[string]any{"group_id": "nowhere"})
	if status != http.StatusNotFound {
		t.Errorf("unknown group: %d %s, want 404", status, body)
	}
	g, _ := um.CreateGroup(context.Background(), actor.Group{Name: "Létező"})
	status, body = send(t, http.MethodPost, srv.URL+"/actors/"+a.ID+"/assignments", map[string]any{"group_id": g.ID})
	if status >= 300 {
		t.Errorf("a known group: %d %s", status, body)
	}
}

func TestTheAuthorizerSeesEveryCallAndCanRefuseIt(t *testing.T) {
	um := managerOnMemory(t)
	var asked []string
	mount := routes.Mount{
		Caller: func(context.Context) string { return "u-1" },
		Authorize: func(_ context.Context, action string) error {
			asked = append(asked, action)
			return context.Canceled
		},
	}
	srv := serverFor(t, um, mount)
	status, _ := send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Tiltott"})
	if status < 400 {
		t.Errorf("a refused call went through: %d", status)
	}
	if len(asked) != 1 || asked[0] != "actor.group.create" {
		t.Errorf("authorizer asked about %v, want actor.group.create once", asked)
	}
	if groups, _ := um.ListGroups(context.Background(), ""); len(groups) != 0 {
		t.Errorf("a refused call still created %d groups", len(groups))
	}
}

func TestToolsAreBuiltFromTheSameOperationsWithDocumentedFields(t *testing.T) {
	um := managerOnMemory(t)
	tools, err := routes.BuildTools(um)
	if err != nil || len(tools) == 0 {
		t.Fatalf("BuildTools = %d tools, %v", len(tools), err)
	}
	if len(routes.Tools(um)) != len(tools) {
		t.Errorf("Tools and BuildTools disagree on how many tools there are")
	}
	if len(routes.ToolsFor(um, routes.Mount{})) != len(tools) {
		t.Errorf("a mount without reference checks changes which tools exist")
	}
	fields, err := routes.Fields()
	if err != nil {
		t.Fatalf("Fields: %v", err)
	}
	name := fields["GroupEdit.Name"]
	if !name.Required || !strings.Contains(name.Description, "200") {
		t.Errorf("GroupEdit.Name doc = %+v", name)
	}
	if fields["GroupEdit.NodeType"].Required || fields["GroupEdit.AnchorLevelOverrideID"].Description == "" {
		t.Errorf("the optional edit fields are described wrongly: %+v", fields)
	}
	readOnly := 0
	for _, f := range fields {
		if f.ReadOnly {
			readOnly++
		}
	}
	if readOnly == 0 {
		t.Errorf("no field is marked read-only, although ids and timestamps are")
	}
}
