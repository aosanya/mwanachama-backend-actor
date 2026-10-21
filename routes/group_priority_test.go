package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestSettingMyGroupPriorityBindsTheCallerNeverTheBody(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	me, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	other, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	notMine, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 2"})
	for _, who := range []string{me.ID, other.ID} {
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: who, GroupID: group.ID}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: other.ID, GroupID: notMine.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return me.ID },
		CheckReferences: true,
	}))

	status, body := do(t, srv, http.MethodPut, "/groups/"+group.ID+"/priority", map[string]any{
		"priority": "very_high",
		"actor_id": other.ID,
	})
	if status != http.StatusOK || body["actor_id"] != me.ID || body["priority"] != "very_high" {
		t.Fatalf("set priority = %d %v, want the caller's registration at very_high", status, body)
	}
	mine, _ := mgr.ListMyGroups(ctx, me.ID)
	theirs, _ := mgr.ListMyGroups(ctx, other.ID)
	if len(mine) != 1 || mine[0].Priority != "very_high" {
		t.Errorf("my groups = %+v, want very_high", mine)
	}
	for _, g := range theirs {
		if g.Priority != "normal" {
			t.Errorf("the body-named actor's %s = %q, want normal untouched", g.ID, g.Priority)
		}
	}

	if status, body := do(t, srv, http.MethodPut, "/groups/"+notMine.ID+"/priority", map[string]any{"priority": "high"}); status != http.StatusNotFound {
		t.Errorf("a group not mine = %d %v, want 404", status, body)
	}
	if status, body := do(t, srv, http.MethodPut, "/groups/"+group.ID+"/priority", map[string]any{"priority": "urgent"}); status != http.StatusBadRequest {
		t.Errorf("an unknown priority = %d %v, want 400", status, body)
	}
}
