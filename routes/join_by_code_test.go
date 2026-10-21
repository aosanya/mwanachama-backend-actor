package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestFindingAndJoiningByCodeOverTheWireTakesTheActorFromTheCaller(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	asker, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Council staff"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.SetGroupHidden(ctx, group.ID, true); err != nil {
		t.Fatalf("SetGroupHidden: %v", err)
	}
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return asker.ID },
		CheckReferences: true,
	}))

	status, body := do(t, srv, http.MethodGet, "/groups/by-code?code="+group.PublicCode+"&actor_id=body-supplied-actor", nil)
	found, _ := body["group"].(map[string]any)
	if status != http.StatusOK || found["id"] != group.ID || found["public_code"] != group.PublicCode || body["join_mode"] != "confirmed" || body["state"] != "none" {
		t.Fatalf("find by code = %d %v, want the hidden group, confirmed, state none", status, body)
	}

	status, body = do(t, srv, http.MethodPost, "/groups/by-code/join", map[string]any{
		"code":     group.PublicCode,
		"message":  "I work here.",
		"actor_id": "body-supplied-actor",
	})
	request, _ := body["request"].(map[string]any)
	if status != http.StatusOK || body["status"] != "requested" || request["actor_id"] != asker.ID || request["group_id"] != group.ID {
		t.Fatalf("join by code = %d %v, want a request from the caller to the group the code found", status, body)
	}

	status, body = do(t, srv, http.MethodGet, "/groups/by-code?code="+group.PublicCode, nil)
	if status != http.StatusOK || body["state"] != "requested" {
		t.Errorf("find after asking = %d %v, want requested", status, body)
	}

	for _, code := range []string{group.Code, "G-999", "nobody"} {
		status, body = do(t, srv, http.MethodGet, "/groups/by-code?code="+code, nil)
		if status != http.StatusNotFound {
			t.Errorf("find %q = %d %v, want 404", code, status, body)
		}
		status, body = do(t, srv, http.MethodPost, "/groups/by-code/join", map[string]any{"code": code})
		if status != http.StatusNotFound {
			t.Errorf("join %q = %d %v, want 404", code, status, body)
		}
	}

	status, body = do(t, srv, http.MethodPost, "/groups/"+group.ID+"/join", map[string]any{})
	if status != http.StatusNotFound {
		t.Errorf("joining the hidden group by id = %d %v, want 404", status, body)
	}
}
