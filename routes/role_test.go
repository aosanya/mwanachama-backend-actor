package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestCreateRoleKind(t *testing.T) {
	um := newTestManager(t)
	handler := routes.CreateRoleKind(um)

	req := httptest.NewRequest(http.MethodPost, "/role-kinds", strings.NewReader(`{"name":"Coordinator"}`))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out mwanachamaactor.RoleKind
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.ID == "" || out.Name != "Coordinator" {
		t.Fatalf("unexpected role kind: %+v", out)
	}
}

func TestListRoleKinds(t *testing.T) {
	um := newTestManager(t)
	if _, err := um.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Coordinator"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := routes.ListRoleKinds(um)

	req := httptest.NewRequest(http.MethodGet, "/role-kinds", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []mwanachamaactor.RoleKind
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out) != 1 {
		t.Fatalf("got %d kinds, want 1", len(out))
	}
}

func TestGetRoleKind(t *testing.T) {
	um := newTestManager(t)
	created, err := um.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := routes.GetRoleKind(um)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/role-kinds/"+created.ID, nil), "kindID", created.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetRoleKind_NotFound(t *testing.T) {
	um := newTestManager(t)
	handler := routes.GetRoleKind(um)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/role-kinds/nope", nil), "kindID", "nope")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetRoleAssignment(t *testing.T) {
	um := newTestManager(t)
	ctx := context.Background()
	k, err := um.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("seed kind: %v", err)
	}
	a, err := um.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: "m-1", GroupID: "g-1", KindID: k.ID})
	if err != nil {
		t.Fatalf("seed grant: %v", err)
	}
	handler := routes.GetRoleAssignment(um)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/role-assignments/"+a.ID, nil), "assignmentID", a.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out mwanachamaactor.ActorRoleAssignment
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.ID != a.ID || out.ActorID != "m-1" {
		t.Fatalf("unexpected assignment: %+v", out)
	}
}

func TestGetRoleAssignment_NotFound(t *testing.T) {
	um := newTestManager(t)
	handler := routes.GetRoleAssignment(um)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/role-assignments/nope", nil), "assignmentID", "nope")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
