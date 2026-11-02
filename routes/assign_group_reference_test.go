package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestAssigningAnActorToAGroupThatDoesNotExistIsRefused(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	actor, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{CheckReferences: true}))

	if status, body := do(t, srv, http.MethodPost, "/actors/"+actor.ID+"/assignments", map[string]any{"group_id": "no-such-group"}); status != http.StatusNotFound {
		t.Fatalf("assigning to a group that does not exist = %d %v, want 404", status, body)
	}
	if held, _ := mgr.ListGroupsForActor(ctx, actor.ID); len(held) != 0 {
		t.Fatalf("the actor holds %v after the refused assignment, want none", held)
	}
	if status, body := do(t, srv, http.MethodPost, "/actors/"+actor.ID+"/assignments", map[string]any{"group_id": group.ID}); status != http.StatusCreated {
		t.Fatalf("assigning to a group that exists = %d %v, want 201", status, body)
	}
}
