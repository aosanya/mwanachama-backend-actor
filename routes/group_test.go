package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

// stubHierarchy is a HierarchyChecker fed by two literal sets — good enough
// for these tests, which never need a real hierarchy/level store.
type stubHierarchy struct {
	hierarchies map[string]bool
	// levels maps levelID -> hierarchyID it belongs to.
	levels map[string]string
}

func (s stubHierarchy) HierarchyExists(_ context.Context, id string) (bool, error) {
	return s.hierarchies[id], nil
}

func (s stubHierarchy) LevelInHierarchy(_ context.Context, levelID, hierarchyID string) (bool, error) {
	return s.levels[levelID] == hierarchyID, nil
}

func newTestManager(t *testing.T) mwanachamaactor.UserManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := mwanachamaactor.DefaultTableNames("routes_test")
	if err := mwanachamaactor.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mgr, err := mwanachamaactor.NewUserManager(db, tables)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return mgr
}

func TestCreateGroup(t *testing.T) {
	um := newTestManager(t)
	hc := stubHierarchy{hierarchies: map[string]bool{"h1": true}, levels: map[string]string{"l1": "h1"}}
	handler := routes.CreateGroup(um, hc)

	body := `{"hierarchy_id":"h1","level_id":"l1","name":"Ward One"}`
	req := httptest.NewRequest(http.MethodPost, "/chapters", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Group
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Name != "Ward One" || !out.Discoverable {
		t.Fatalf("unexpected group: %+v", out)
	}
}

func TestCreateGroup_DiscoverableExplicitFalse(t *testing.T) {
	um := newTestManager(t)
	hc := stubHierarchy{hierarchies: map[string]bool{"h1": true}}
	handler := routes.CreateGroup(um, hc)

	body := `{"hierarchy_id":"h1","name":"Covert Ward","discoverable":false}`
	req := httptest.NewRequest(http.MethodPost, "/chapters", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Group
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Discoverable {
		t.Fatalf("expected discoverable=false to be honoured, got %+v", out)
	}
}

func TestCreateGroup_InvalidHierarchyReference(t *testing.T) {
	um := newTestManager(t)
	hc := stubHierarchy{}
	handler := routes.CreateGroup(um, hc)

	body := `{"hierarchy_id":"missing","name":"Ward"}`
	req := httptest.NewRequest(http.MethodPost, "/chapters", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateGroup_MissingHierarchyID(t *testing.T) {
	um := newTestManager(t)
	handler := routes.CreateGroup(um, stubHierarchy{})

	req := httptest.NewRequest(http.MethodPost, "/chapters", strings.NewReader(`{"name":"Ward"}`))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func newGroup(t *testing.T, um mwanachamaactor.UserManager, hierarchyID string) models.Group {
	t.Helper()
	g, err := um.CreateGroup(context.Background(), models.Group{HierarchyID: hierarchyID, Name: "seed"})
	if err != nil {
		t.Fatalf("seed CreateGroup: %v", err)
	}
	return g
}

func withPathValue(req *http.Request, key, value string) *http.Request {
	req.SetPathValue(key, value)
	return req
}

func TestEditGroup(t *testing.T) {
	um := newTestManager(t)
	g := newGroup(t, um, "h1")
	hc := stubHierarchy{hierarchies: map[string]bool{"h1": true}, levels: map[string]string{"l2": "h1"}}
	handler := routes.EditGroup(um, hc)

	body := `{"name":"Renamed Ward","anchor_level_override":"l2"}`
	req := withPathValue(httptest.NewRequest(http.MethodPatch, "/chapters/"+g.ID, strings.NewReader(body)), "groupID", g.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Group
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Name != "Renamed Ward" || out.AnchorLevelOverrideID != "l2" {
		t.Fatalf("unexpected group: %+v", out)
	}
}

func TestEditGroup_AnchorOverrideWrongHierarchy(t *testing.T) {
	um := newTestManager(t)
	g := newGroup(t, um, "h1")
	// l2 belongs to h2, not h1 — the cross-hierarchy override the gateway's
	// old migration-000034 FK used to reject; checkHierarchyRefs rejects it
	// the same way, in Go rather than at the database.
	hc := stubHierarchy{hierarchies: map[string]bool{"h1": true, "h2": true}, levels: map[string]string{"l2": "h2"}}
	handler := routes.EditGroup(um, hc)

	req := withPathValue(httptest.NewRequest(http.MethodPatch, "/chapters/"+g.ID, strings.NewReader(`{"anchor_level_override":"l2"}`)), "groupID", g.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestEditGroup_NotFound(t *testing.T) {
	um := newTestManager(t)
	handler := routes.EditGroup(um, stubHierarchy{})

	req := withPathValue(httptest.NewRequest(http.MethodPatch, "/chapters/nope", strings.NewReader(`{"name":"x"}`)), "groupID", "nope")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestMoveGroup(t *testing.T) {
	um := newTestManager(t)
	root := newGroup(t, um, "h1")
	child, err := um.CreateGroup(context.Background(), models.Group{HierarchyID: "h1", Name: "child", ParentID: root.ID})
	if err != nil {
		t.Fatalf("seed child: %v", err)
	}
	newParent := newGroup(t, um, "h1")
	handler := routes.MoveGroup(um)

	req := withPathValue(httptest.NewRequest(http.MethodPost, "/chapters/"+child.ID+"/move", strings.NewReader(`{"parent_id":"`+newParent.ID+`"}`)), "groupID", child.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Group
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.ParentID != newParent.ID {
		t.Fatalf("unexpected group: %+v", out)
	}
}

func TestEditGroup_NameRequired(t *testing.T) {
	um := newTestManager(t)
	g := newGroup(t, um, "h1")
	handler := routes.EditGroup(um, stubHierarchy{})

	req := withPathValue(httptest.NewRequest(http.MethodPatch, "/chapters/"+g.ID, strings.NewReader(`{"name":"   "}`)), "groupID", g.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestMoveGroup_EmptyParentRefused(t *testing.T) {
	um := newTestManager(t)
	root := newGroup(t, um, "h1")
	child, err := um.CreateGroup(context.Background(), models.Group{HierarchyID: "h1", Name: "child", ParentID: root.ID})
	if err != nil {
		t.Fatalf("seed child: %v", err)
	}
	handler := routes.MoveGroup(um)

	req := withPathValue(httptest.NewRequest(http.MethodPost, "/chapters/"+child.ID+"/move", strings.NewReader(`{"parent_id":""}`)), "groupID", child.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestMoveGroup_ParentIsSelf(t *testing.T) {
	um := newTestManager(t)
	g := newGroup(t, um, "h1")
	handler := routes.MoveGroup(um)

	req := withPathValue(httptest.NewRequest(http.MethodPost, "/chapters/"+g.ID+"/move", strings.NewReader(`{"parent_id":"`+g.ID+`"}`)), "groupID", g.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
