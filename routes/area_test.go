package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestAGroupNamesItsAreaNodeThroughTheDeclaredTable(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	srv := newServer(t, routes.Routes(mgr))

	h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Tree"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	status, top := do(t, srv, http.MethodPost, "/levels", map[string]any{"hierarchy_id": h.ID, "name": "Council", "depth": 0})
	if status != http.StatusCreated {
		t.Fatalf("create top level = %d %v", status, top)
	}
	status, district := do(t, srv, http.MethodPost, "/levels", map[string]any{"hierarchy_id": h.ID, "name": "District", "depth": 1, "is_area": true})
	if status != http.StatusCreated || district["is_area"] != true || district["name"] != "District" {
		t.Fatalf("create area level = %d %v, want is_area true keeping its name", status, district)
	}

	council, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{HierarchyID: h.ID, LevelID: top["id"].(string), Name: "Council"})
	north, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{HierarchyID: h.ID, LevelID: district["id"].(string), Name: "North"})

	if status, body := do(t, srv, http.MethodPut, "/groups/"+north.ID+"/area-node", map[string]any{"area_node_id": "node-north"}); status != http.StatusOK || body["area_node_id"] != "node-north" {
		t.Fatalf("set area node = %d %v", status, body)
	}
	if status, body := do(t, srv, http.MethodGet, "/groups/"+north.ID, nil); status != http.StatusOK || body["area_node_id"] != "node-north" {
		t.Errorf("read area node = %d %v, want node-north", status, body)
	}
	if status, body := do(t, srv, http.MethodPut, "/groups/"+council.ID+"/area-node", map[string]any{"area_node_id": "node-council"}); status != http.StatusConflict {
		t.Errorf("an area node on a group not at an area level = %d %v, want 409", status, body)
	}
	if status, body := do(t, srv, http.MethodPut, "/groups/missing/area-node", map[string]any{"area_node_id": "node-x"}); status != http.StatusNotFound {
		t.Errorf("an area node on a missing group = %d %v, want 404", status, body)
	}
	if status, body := do(t, srv, http.MethodPut, "/groups/"+north.ID+"/area-node", map[string]any{}); status != http.StatusOK || body["area_node_id"] != nil {
		t.Errorf("clear area node = %d %v", status, body)
	}
}
