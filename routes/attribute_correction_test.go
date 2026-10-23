package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestASelfCorrectionBindsTheCallerAndRecordsSelfProvenance(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	me, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina", Attributes: map[string]any{"phone": "+254700000001"}})
	other, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return me.ID },
		CheckReferences: true,
	}))

	if status, _ := do(t, srv, http.MethodPatch, "/actors/mine/attributes", map[string]any{
		"attributes": map[string]any{"email": "amina@example.org"},
		"actor_id":   other.ID,
	}); status != http.StatusBadRequest {
		t.Fatalf("a body naming an actor = %d, want 400", status)
	}
	status, body := do(t, srv, http.MethodPatch, "/actors/mine/attributes", map[string]any{
		"attributes": map[string]any{"email": "amina@example.org"},
	})
	if status != http.StatusOK || body["id"] != me.ID {
		t.Fatalf("correct = %d %v, want the caller's record", status, body)
	}
	held, _ := mgr.GetActor(ctx, me.ID)
	if held.Attributes["email"] != "amina@example.org" || held.Attributes["phone"] != "+254700000001" {
		t.Fatalf("attributes = %v, want email added and phone kept", held.Attributes)
	}
	src := held.AttributeSources["email"]
	if src.SetBy != "self" || src.SetByID != me.ID || src.SetAt == "" {
		t.Errorf("email source = %+v, want self by the caller", src)
	}
	if _, ok := held.AttributeSources["phone"]; ok {
		t.Errorf("phone gained a source it was never given")
	}
	if theirs, _ := mgr.GetActor(ctx, other.ID); len(theirs.Attributes) != 0 {
		t.Errorf("the body-named actor changed: %v", theirs.Attributes)
	}
}

func TestAStaffEditRecordsStaffProvenanceAndASelfCorrectionOverridesIt(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	clerk, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Clerk"})
	resident, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Resident"})
	asClerk := newServer(t, routes.RoutesFor(mgr, routes.Mount{Caller: func(context.Context) string { return clerk.ID }}))
	asResident := newServer(t, routes.RoutesFor(mgr, routes.Mount{Caller: func(context.Context) string { return resident.ID }}))

	status, body := do(t, asClerk, http.MethodPatch, "/actors/"+resident.ID+"/attributes", map[string]any{
		"attributes": map[string]any{"email": "wrong@example.org"},
	})
	if status != http.StatusOK {
		t.Fatalf("staff edit = %d %v", status, body)
	}
	held, _ := mgr.GetActor(ctx, resident.ID)
	if s := held.AttributeSources["email"]; s.SetBy != "staff" || s.SetByID != clerk.ID {
		t.Fatalf("source = %+v, want staff by the clerk", s)
	}

	status, body = do(t, asResident, http.MethodPatch, "/actors/mine/attributes", map[string]any{
		"attributes": map[string]any{"email": "right@example.org"},
	})
	if status != http.StatusOK {
		t.Fatalf("self correction = %d %v", status, body)
	}
	held, _ = mgr.GetActor(ctx, resident.ID)
	if held.Attributes["email"] != "right@example.org" || held.AttributeSources["email"].SetBy != "self" {
		t.Errorf("after correction = %v %+v, want the resident's value marked self", held.Attributes, held.AttributeSources)
	}
}

func TestASelfCorrectionIsValidatedLikeACreate(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	taken, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "A", Attributes: map[string]any{"email": "a@example.org"}})
	me, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "B", Attributes: map[string]any{"email": "b@example.org"}})
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{Caller: func(context.Context) string { return me.ID }}))

	if status, _ := do(t, srv, http.MethodPatch, "/actors/mine/attributes", map[string]any{
		"attributes": map[string]any{"email": "a@example.org"},
	}); status != http.StatusBadRequest {
		t.Errorf("another's email = %d, want 400", status)
	}
	if status, _ := do(t, srv, http.MethodPatch, "/actors/mine/attributes", map[string]any{
		"attributes": map[string]any{"ward": "7"},
	}); status != http.StatusBadRequest {
		t.Errorf("undeclared attribute = %d, want 400", status)
	}
	if status, _ := do(t, srv, http.MethodPatch, "/actors/mine/attributes", map[string]any{
		"attributes": map[string]any{"email": "b@example.org"},
	}); status != http.StatusOK {
		t.Errorf("keeping my own email = %d, want 200", status)
	}
	held, _ := mgr.GetActor(ctx, taken.ID)
	if held.Attributes["email"] != "a@example.org" {
		t.Errorf("the other actor's email changed")
	}
}

func TestCorrectingMyAttributesWithNoRecordAnswersNotFound(t *testing.T) {
	mgr := newManager(t)
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{Caller: func(context.Context) string { return "nobody" }}))
	if status, _ := do(t, srv, http.MethodPatch, "/actors/mine/attributes", map[string]any{
		"attributes": map[string]any{"email": "x@example.org"},
	}); status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}
}
