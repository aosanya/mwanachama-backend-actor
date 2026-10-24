package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestADelegationGrantBindsTheGrantorFromTheCaller(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	a, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Delegate"})
	b, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Subject"})
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return "granter" },
	}))

	if status, _ := do(t, srv, http.MethodPost, "/delegations", map[string]any{
		"delegate_id": a.ID, "subject_id": b.ID, "granted_by": "someone-else",
	}); status != http.StatusBadRequest {
		t.Fatalf("a body naming the grantor = %d, want 400", status)
	}
	if status, _ := do(t, srv, http.MethodPost, "/delegations", map[string]any{
		"delegate_id": a.ID, "subject_id": a.ID,
	}); status != http.StatusBadRequest {
		t.Fatalf("a delegation to oneself = %d, want 400", status)
	}
	status, body := do(t, srv, http.MethodPost, "/delegations", map[string]any{
		"delegate_id": a.ID, "subject_id": b.ID,
	})
	if status != http.StatusOK || body["granted_by"] != "granter" {
		t.Fatalf("grant = %d %v, want the caller as grantor", status, body)
	}
	id, _ := body["id"].(string)
	if status, body := do(t, srv, http.MethodPost, "/delegations/"+id+"/revoke", nil); status != http.StatusOK || body["revoked_by"] != "granter" {
		t.Fatalf("revoke = %d %v", status, body)
	}
	if status, _ := do(t, srv, http.MethodPost, "/delegations/"+id+"/revoke", nil); status != http.StatusConflict {
		t.Fatalf("second revoke = %d, want 409", status)
	}
	if ok, _ := mgr.ActsFor(ctx, a.ID, b.ID); ok {
		t.Error("a revoked delegation still acts")
	}
}
