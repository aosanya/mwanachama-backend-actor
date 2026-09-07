package mwanachamaactor_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestGroupDashboardComposesRolesAndCounts(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "County"})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	if _, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward A", ParentID: root.ID}); err != nil {
		t.Fatalf("create ward a: %v", err)
	}
	if _, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward B", ParentID: root.ID}); err != nil {
		t.Fatalf("create ward b: %v", err)
	}

	kind1, _ := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator"})
	kind2, _ := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Organizer"})

	actorIDs := map[string]string{}
	for _, label := range []string{"m-1", "m-2", "m-3", "m-4"} {
		created, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: label})
		if err != nil {
			t.Fatalf("create actor %s: %v", label, err)
		}
		actorIDs[label] = created.ID
	}

	a1, _ := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: actorIDs["m-1"], GroupID: root.ID, KindID: kind1.ID})
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: actorIDs["m-2"], GroupID: root.ID, KindID: kind2.ID}); err != nil {
		t.Fatalf("grant m-2: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: actorIDs["m-3"], GroupID: root.ID, KindID: kind2.ID}); err != nil {
		t.Fatalf("grant m-3: %v", err)
	}
	if err := mgr.StepDownRole(ctx, a1.ID); err != nil {
		t.Fatalf("stepdown: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: "m-99", GroupID: "elsewhere-nonexistent-group", KindID: kind1.ID}); err != nil {
		t.Fatalf("grant elsewhere: %v", err)
	}

	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: actorIDs["m-1"], GroupID: root.ID}); err != nil {
		t.Fatalf("register m-1: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: actorIDs["m-2"], GroupID: root.ID}); err != nil {
		t.Fatalf("register m-2: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: actorIDs["m-3"], GroupID: root.ID}); err != nil {
		t.Fatalf("register m-3: %v", err)
	}

	dash, err := mgr.GroupDashboard(ctx, root.ID)
	if err != nil {
		t.Fatalf("GroupDashboard: %v", err)
	}
	if dash.GroupID != root.ID {
		t.Fatalf("wrong group id: %q", dash.GroupID)
	}
	if dash.MemberCount != 3 {
		t.Fatalf("expected 3 members at root, got %d", dash.MemberCount)
	}
	if dash.ChildCount != 2 {
		t.Fatalf("expected 2 direct children, got %d", dash.ChildCount)
	}
	if dash.ActiveRoles != 2 {
		t.Fatalf("expected 2 active roles after step-down, got %d", dash.ActiveRoles)
	}

	names := make([]string, 0, len(dash.Roles))
	membersByName := map[string][]string{}
	for _, r := range dash.Roles {
		names = append(names, r.KindName)
		membersByName[r.KindName] = append([]string(nil), r.Members...)
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "Coordinator" || names[1] != "Organizer" {
		t.Fatalf("unexpected role names: %v", names)
	}
	if len(membersByName["Coordinator"]) != 0 {
		t.Fatalf("expected Coordinator to have 0 members after step-down, got %v", membersByName["Coordinator"])
	}
	if len(membersByName["Organizer"]) != 2 {
		t.Fatalf("unexpected Organizer members: %v", membersByName["Organizer"])
	}
}

func TestGroupDashboardMissingGroup(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GroupDashboard(context.Background(), "no-such-group"); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound for missing group, got %v", err)
	}
}

// TestGroupDashboardMemberIDsNeverNil pins DEV-1093, ported: a kind held by
// nobody at a group must serialise its member list as [], not null.
func TestGroupDashboardMemberIDsNeverNil(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "County"})
	if _, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Treasurer"}); err != nil {
		t.Fatalf("kind: %v", err)
	}

	d, err := mgr.GroupDashboard(ctx, root.ID)
	if err != nil {
		t.Fatalf("GroupDashboard: %v", err)
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.Contains(b, []byte(`"actor_ids":null`)) {
		t.Fatalf("actor_ids marshalled as null: %s", b)
	}
	if !bytes.Contains(b, []byte(`"actor_ids":[]`)) {
		t.Fatalf("want an empty array for an unheld kind, got: %s", b)
	}
}
