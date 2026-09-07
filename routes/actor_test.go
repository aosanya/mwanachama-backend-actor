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

func TestCreateActor(t *testing.T) {
	um := newTestManager(t)
	handler := routes.CreateActor(um)

	req := httptest.NewRequest(http.MethodPost, "/actors", strings.NewReader(`{"display_name":"Amos"}`))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out mwanachamaactor.Actor
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.ID == "" || out.DisplayName != "Amos" {
		t.Fatalf("unexpected actor: %+v", out)
	}
}

func TestGetActor(t *testing.T) {
	um := newTestManager(t)
	created, err := um.CreateActor(context.Background(), mwanachamaactor.Actor{DisplayName: "Amos"})
	if err != nil {
		t.Fatalf("seed CreateActor: %v", err)
	}
	handler := routes.GetActor(um)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/actors/"+created.ID, nil), "actorID", created.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetActor_NotFound(t *testing.T) {
	um := newTestManager(t)
	handler := routes.GetActor(um)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/actors/nope", nil), "actorID", "nope")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListActors(t *testing.T) {
	um := newTestManager(t)
	if _, err := um.CreateActor(context.Background(), mwanachamaactor.Actor{DisplayName: "Amos"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := routes.ListActors(um)

	req := httptest.NewRequest(http.MethodGet, "/actors", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []mwanachamaactor.Actor
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out) != 1 {
		t.Fatalf("got %d actors, want 1", len(out))
	}
}

func TestSetActorDisplayName(t *testing.T) {
	um := newTestManager(t)
	created, err := um.CreateActor(context.Background(), mwanachamaactor.Actor{DisplayName: "Amos"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := routes.SetActorDisplayName(um)

	req := withPathValue(httptest.NewRequest(http.MethodPatch, "/actors/"+created.ID+"/display-name", strings.NewReader(`{"display_name":"A. Sanya"}`)), "actorID", created.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out mwanachamaactor.Actor
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.DisplayName != "A. Sanya" {
		t.Fatalf("unexpected actor: %+v", out)
	}
}
