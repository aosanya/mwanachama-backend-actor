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
	if len(table) != 87 {
		t.Errorf("the declared table has %d routes, want 87", len(table))
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
	if len(tools) != 87 {
		t.Errorf("the declared table publishes %d tools, want 87", len(tools))
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

func TestJoiningOverTheWireTakesTheActorFromTheCaller(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	asker, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	caller := asker.ID
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return caller },
		CheckReferences: true,
	}))

	status, body := do(t, srv, http.MethodPost, "/groups/"+group.ID+"/join", map[string]any{
		"message":  "I live here.",
		"actor_id": "body-supplied-actor",
	})
	if status != http.StatusOK {
		t.Fatalf("join = %d, want 200 (body %v)", status, body)
	}
	request, _ := body["request"].(map[string]any)
	if body["status"] != "requested" || request["actor_id"] != asker.ID {
		t.Fatalf("join = %v, want a request from the caller", body)
	}

	seatLeader(t, mgr, group.ID, "leader-1")
	caller = "leader-1"
	status, body = do(t, srv, http.MethodPost, "/join-requests/"+request["id"].(string)+"/decline", map[string]any{
		"silent": true,
		"reason": "Not now.",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("a quiet decline with a reason = %d %v, want 400", status, body)
	}

	status, body = do(t, srv, http.MethodPost, "/join-requests/"+request["id"].(string)+"/decline", map[string]any{
		"silent": true,
	})
	if status != http.StatusOK || body["state"] != "declined" || body["silent"] != true || body["decided_by"] != "leader-1" {
		t.Fatalf("decline = %d %v, want a silent decline by the caller", status, body)
	}

	status, body = do(t, srv, http.MethodPost, "/join-requests/"+request["id"].(string)+"/approve", nil)
	if status != http.StatusConflict {
		t.Errorf("approving a declined request = %d %v, want 409", status, body)
	}

	status, body = do(t, srv, http.MethodPut, "/groups/"+group.ID+"/join-mode", map[string]any{"join_mode": "anyone"})
	if status != http.StatusBadRequest {
		t.Errorf("an unknown join mode = %d %v, want 400", status, body)
	}
}

func TestAnInvitationOverTheWireTakesIssuerAndAnswererFromTheCaller(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	invited, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Elders"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.SetGroupHidden(ctx, group.ID, true); err != nil {
		t.Fatalf("SetGroupHidden: %v", err)
	}
	seatLeader(t, mgr, group.ID, "leader-1")
	caller := "leader-1"
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return caller },
		CheckReferences: true,
	}))

	status, body := do(t, srv, http.MethodPost, "/groups/"+group.ID+"/invitations", map[string]any{
		"actor_id":   invited.ID,
		"invited_by": "body-supplied-leader",
	})
	if status != http.StatusCreated || body["invited_by"] != "leader-1" || body["state"] != "pending" {
		t.Fatalf("invite = %d %v, want a pending invitation from the caller", status, body)
	}
	id := body["id"].(string)

	status, _ = do(t, srv, http.MethodPost, "/invitations/"+id+"/accept", nil)
	if status != http.StatusNotFound {
		t.Errorf("the issuer accepting = %d, want 404", status)
	}

	caller = invited.ID
	status, body = do(t, srv, http.MethodPost, "/invitations/"+id+"/accept", map[string]any{"actor_id": "someone-else"})
	if status != http.StatusOK || body["state"] != "accepted" {
		t.Fatalf("accept = %d %v, want accepted", status, body)
	}
	status, _ = do(t, srv, http.MethodPost, "/invitations/"+id+"/decline", nil)
	if status != http.StatusConflict {
		t.Errorf("declining an accepted invitation = %d, want 409", status)
	}
	caller = "leader-1"
	status, body = do(t, srv, http.MethodPost, "/groups/"+group.ID+"/invitations", map[string]any{"actor_id": invited.ID})
	if status != http.StatusConflict {
		t.Errorf("inviting a registered actor = %d %v, want 409", status, body)
	}
}

func seatLeader(t *testing.T, mgr mwanachamaactor.UserManager, groupID, actorID string) {
	t.Helper()
	ctx := context.Background()
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Chair", LeadsGroup: true})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: actorID, GroupID: groupID, KindID: kind.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
}

func TestOverTheWireOnlyALeaderDecidesAndTheLeaderReadsWhatTheyLead(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	asker, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Secretary"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: "secretary", GroupID: group.ID, KindID: kind.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	caller := asker.ID
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return caller },
		CheckReferences: true,
	}))
	_, body := do(t, srv, http.MethodPost, "/groups/"+group.ID+"/join", map[string]any{})
	request, _ := body["request"].(map[string]any)
	requestID, _ := request["id"].(string)

	caller = "secretary"
	status, body := do(t, srv, http.MethodPost, "/join-requests/"+requestID+"/approve", nil)
	if status != http.StatusForbidden {
		t.Fatalf("a non-leader approving = %d %v, want 403", status, body)
	}
	status, body = do(t, srv, http.MethodPost, "/groups/"+group.ID+"/invitations", map[string]any{"actor_id": asker.ID})
	if status != http.StatusForbidden {
		t.Fatalf("a non-leader inviting = %d %v, want 403", status, body)
	}

	status, body = do(t, srv, http.MethodPut, "/role-kinds/"+kind.ID+"/leads-group", map[string]any{"leads_group": true})
	if status != http.StatusOK || body["leads_group"] != true {
		t.Fatalf("marking the kind = %d %v, want it leading", status, body)
	}
	status, body = do(t, srv, http.MethodPost, "/join-requests/"+requestID+"/approve", nil)
	if status != http.StatusOK || body["decided_by"] != "secretary" {
		t.Fatalf("the leader approving = %d %v, want approved by the caller", status, body)
	}

	resp, err := http.Get(srv.URL + "/groups/led?actor_id=" + asker.ID)
	if err != nil {
		t.Fatalf("GET /groups/led: %v", err)
	}
	defer resp.Body.Close()
	var led []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&led); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.StatusCode != http.StatusOK || len(led) != 1 || led[0]["id"] != group.ID {
		t.Fatalf("led = %d %v, want only %s for the caller, whatever the query names", resp.StatusCode, led, group.ID)
	}
}

func getList(t *testing.T, srv *httptest.Server, path string, body any) (int, string, []map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read: %v", err)
	}
	var out []map[string]any
	_ = json.Unmarshal(buf.Bytes(), &out)
	return resp.StatusCode, buf.String(), out
}

func TestOverTheWireACallerReadsOnlyTheirOwnGroups(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	amina, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	baraka, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	mine, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	theirs, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 5"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: amina.ID, GroupID: mine.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: baraka.ID, GroupID: theirs.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	caller := amina.ID
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return caller },
		CheckReferences: true,
	}))

	status, raw, groups := getList(t, srv, "/groups/mine?actor_id="+baraka.ID, map[string]any{"actor_id": baraka.ID})
	if status != http.StatusOK || len(groups) != 1 || groups[0]["id"] != mine.ID || groups[0]["name"] != "Ward 4" {
		t.Fatalf("GET /groups/mine = %d %s, want only %s for the caller, whatever the query or body names", status, raw, mine.ID)
	}

	stranger, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Chausiku"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	caller = stranger.ID
	status, raw, groups = getList(t, srv, "/groups/mine", nil)
	if status != http.StatusOK || len(groups) != 0 || bytes.TrimSpace([]byte(raw))[0] != '[' {
		t.Fatalf("GET /groups/mine with no registrations = %d %s, want an empty list", status, raw)
	}

	status, body := do(t, srv, http.MethodGet, "/groups/"+mine.ID, nil)
	if status != http.StatusOK || body["id"] != mine.ID {
		t.Fatalf("GET /groups/{id} beside /groups/mine = %d %v, want the group", status, body)
	}
}

func TestOverTheWireALeaderInvitesByTheCodeTheInviteeReadsAsTheirOwn(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	invited, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Elders"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.SetGroupHidden(ctx, group.ID, true); err != nil {
		t.Fatalf("SetGroupHidden: %v", err)
	}
	seatLeader(t, mgr, group.ID, "leader-1")
	caller := invited.ID
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return caller },
		CheckReferences: true,
	}))

	status, body := do(t, srv, http.MethodGet, "/actors/mine", nil)
	if status != http.StatusOK || body["id"] != invited.ID || body["public_code"] != invited.PublicCode {
		t.Fatalf("my actor = %d %v, want %s carrying %s", status, body, invited.ID, invited.PublicCode)
	}
	code := body["public_code"].(string)

	caller = "nobody-yet"
	status, _ = do(t, srv, http.MethodGet, "/actors/mine", nil)
	if status != http.StatusNotFound {
		t.Errorf("a caller with no actor reading their own = %d, want 404", status)
	}

	caller = "leader-1"
	status, body = do(t, srv, http.MethodPost, "/groups/"+group.ID+"/invitations", map[string]any{"actor_code": code})
	if status != http.StatusCreated || body["actor_id"] != invited.ID || body["invited_by"] != "leader-1" {
		t.Fatalf("invite by code = %d %v, want an invitation for %s from the caller", status, body, invited.ID)
	}

	status, internal := do(t, srv, http.MethodPost, "/groups/"+group.ID+"/invitations", map[string]any{"actor_code": invited.Code})
	if status != http.StatusNotFound {
		t.Errorf("invite by the internal code %s = %d %v, want 404", invited.Code, status, internal)
	}

	status, unknown := do(t, srv, http.MethodPost, "/groups/"+group.ID+"/invitations", map[string]any{"actor_code": "M-AAA-BBB-CCC"})
	statusByID, unknownByID := do(t, srv, http.MethodPost, "/groups/"+group.ID+"/invitations", map[string]any{"actor_id": "missing"})
	if status != http.StatusNotFound || statusByID != http.StatusNotFound || fmt.Sprint(unknown) != fmt.Sprint(unknownByID) {
		t.Errorf("unknown code = %d %v, unknown id = %d %v, want the same 404", status, unknown, statusByID, unknownByID)
	}

	status, _ = do(t, srv, http.MethodPost, "/groups/"+group.ID+"/invitations", map[string]any{"actor_id": invited.ID, "actor_code": code})
	if status != http.StatusBadRequest {
		t.Errorf("naming the invitee twice = %d, want 400", status)
	}
}

func TestADeclineReasonCrossesTheWireToTheApplicant(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	asker, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	caller := asker.ID
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return caller },
	}))
	_, body := do(t, srv, http.MethodPost, "/groups/"+group.ID+"/join", map[string]any{"message": "Please."})
	request, _ := body["request"].(map[string]any)
	id, _ := request["id"].(string)

	seatLeader(t, mgr, group.ID, "leader-1")
	caller = "leader-1"
	status, body := do(t, srv, http.MethodPost, "/join-requests/"+id+"/decline", map[string]any{
		"reason": "This group is for residents of Ward 4.",
	})
	if status != http.StatusOK || body["decline_reason"] != "This group is for residents of Ward 4." {
		t.Fatalf("decline = %d %v, want the reason returned", status, body)
	}

	caller = asker.ID
	status, body = do(t, srv, http.MethodGet, "/join-requests/"+id, nil)
	if status != http.StatusOK || body["decline_reason"] != "This group is for residents of Ward 4." {
		t.Errorf("the applicant's read = %d %v, want the reason", status, body)
	}
}
