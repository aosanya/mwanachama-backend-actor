package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestAssignGroup(t *testing.T) {
	um := newTestManager(t)
	actor, err := um.CreateActor(context.Background(), models.Actor{DisplayName: "Amos"})
	if err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	group := newGroup(t, um, "h1")
	handler := routes.AssignGroup(um)

	body := `{"group_id":"` + group.ID + `","attributes":{"is_home":true}}`
	req := withPathValue(httptest.NewRequest(http.MethodPost, "/actors/"+actor.ID+"/assignments", strings.NewReader(body)), "actorID", actor.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.ActorGroupAssignment
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	isHome, _ := out.Attributes["is_home"].(bool)
	if out.ActorID != actor.ID || out.GroupID != group.ID || !isHome {
		t.Fatalf("unexpected assignment: %+v", out)
	}
}

func TestAssignGroup_UnknownActor(t *testing.T) {
	um := newTestManager(t)
	group := newGroup(t, um, "h1")
	handler := routes.AssignGroup(um)

	req := withPathValue(httptest.NewRequest(http.MethodPost, "/actors/nope/assignments", strings.NewReader(`{"group_id":"`+group.ID+`"}`)), "actorID", "nope")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeregister_Idempotent(t *testing.T) {
	um := newTestManager(t)
	actor, err := um.CreateActor(context.Background(), models.Actor{DisplayName: "Amos"})
	if err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	group := newGroup(t, um, "h1")
	handler := routes.Deregister(um)

	req := withPathValue(withPathValue(httptest.NewRequest(http.MethodDelete, "/actors/"+actor.ID+"/assignments/"+group.ID, nil), "actorID", actor.ID), "groupID", group.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListGroupsForActor(t *testing.T) {
	um := newTestManager(t)
	actor, err := um.CreateActor(context.Background(), models.Actor{DisplayName: "Amos"})
	if err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	group := newGroup(t, um, "h1")
	if _, err := um.AssignGroup(context.Background(), models.ActorGroupAssignment{ActorID: actor.ID, GroupID: group.ID}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	handler := routes.ListGroupsForActor(um)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/actors/"+actor.ID+"/assignments", nil), "actorID", actor.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []models.ActorGroupAssignment
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out) != 1 || out[0].GroupID != group.ID {
		t.Fatalf("unexpected assignments: %+v", out)
	}
}

func TestListActorsForGroup(t *testing.T) {
	um := newTestManager(t)
	actor, err := um.CreateActor(context.Background(), models.Actor{DisplayName: "Amos"})
	if err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	group := newGroup(t, um, "h1")
	if _, err := um.AssignGroup(context.Background(), models.ActorGroupAssignment{ActorID: actor.ID, GroupID: group.ID}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	handler := routes.ListActorsForGroup(um)

	req := withPathValue(httptest.NewRequest(http.MethodGet, "/groups/"+group.ID+"/assignments", nil), "groupID", group.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []models.ActorGroupAssignment
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out) != 1 || out[0].ActorID != actor.ID {
		t.Fatalf("unexpected assignments: %+v", out)
	}
}

func TestHomeCounts(t *testing.T) {
	um := newTestManager(t)
	actor, err := um.CreateActor(context.Background(), models.Actor{DisplayName: "Amos"})
	if err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	group := newGroup(t, um, "h1")
	if _, err := um.AssignGroup(context.Background(), models.ActorGroupAssignment{ActorID: actor.ID, GroupID: group.ID, Attributes: map[string]any{"is_home": true}}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	handler := routes.HomeCounts(um)

	req := httptest.NewRequest(http.MethodGet, "/assignments/home-counts", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out map[string]int
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out[group.ID] != 1 {
		t.Fatalf("unexpected counts: %+v", out)
	}
}
