package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func squareJSON(lon, lat, size float64) map[string]any {
	return map[string]any{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{lon, lat}, {lon + size, lat}, {lon + size, lat + size}, {lon, lat + size}, {lon, lat},
		}},
	}
}

func TestAnAreaIsMarkedOutlinedAndResolvedThroughTheDeclaredTable(t *testing.T) {
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
	status, ward := do(t, srv, http.MethodPost, "/levels", map[string]any{"hierarchy_id": h.ID, "name": "Ward", "depth": 2, "is_area": true})
	if status != http.StatusCreated {
		t.Fatalf("create ward level = %d %v", status, ward)
	}

	council, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{HierarchyID: h.ID, LevelID: top["id"].(string), Name: "Council"})
	north, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{HierarchyID: h.ID, LevelID: district["id"].(string), Name: "North"})
	inner, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{HierarchyID: h.ID, LevelID: ward["id"].(string), Name: "Inner"})

	if status, body := do(t, srv, http.MethodPut, "/groups/"+north.ID+"/outline", map[string]any{"outline": squareJSON(10, 10, 4)}); status != http.StatusOK || body["outline"] == nil {
		t.Fatalf("set district outline = %d %v", status, body)
	}
	if status, body := do(t, srv, http.MethodPut, "/groups/"+inner.ID+"/outline", map[string]any{"outline": squareJSON(11, 11, 1)}); status != http.StatusOK {
		t.Fatalf("set ward outline = %d %v", status, body)
	}
	if status, body := do(t, srv, http.MethodPut, "/groups/"+council.ID+"/outline", map[string]any{"outline": squareJSON(0, 0, 50)}); status != http.StatusConflict {
		t.Errorf("an outline on a group not at an area level = %d %v, want 409", status, body)
	}
	if status, body := do(t, srv, http.MethodPut, "/groups/"+north.ID+"/outline", map[string]any{"outline": map[string]any{"type": "Point", "coordinates": []float64{1, 1}}}); status != http.StatusBadRequest {
		t.Errorf("a point as an outline = %d %v, want 400", status, body)
	}
	if status, body := do(t, srv, http.MethodPut, "/groups/missing/outline", map[string]any{"outline": squareJSON(0, 0, 1)}); status != http.StatusNotFound {
		t.Errorf("an outline on a missing group = %d %v, want 404", status, body)
	}

	resolve := func(lon, lat string) map[string]any {
		t.Helper()
		status, body := do(t, srv, http.MethodGet, "/hierarchies/"+h.ID+"/resolve-point?lon="+lon+"&lat="+lat, nil)
		if status != http.StatusOK {
			t.Fatalf("resolve %s,%s = %d %v", lon, lat, status, body)
		}
		return body
	}
	deep := resolve("11.5", "11.5")
	if deep["found"] != true || deep["group"].(map[string]any)["id"] != inner.ID {
		t.Errorf("a pin inside the ward = %v, want the deepest area, Inner", deep)
	}
	shallow := resolve("13.5", "13.5")
	if shallow["found"] != true || shallow["group"].(map[string]any)["id"] != north.ID {
		t.Errorf("a pin in the district outside the ward = %v, want North", shallow)
	}
	if outside := resolve("-20", "-20"); outside["found"] != false {
		t.Errorf("a pin in no area = %v, want found false", outside)
	}

	if status, body := do(t, srv, http.MethodPut, "/groups/"+inner.ID+"/outline", map[string]any{}); status != http.StatusOK {
		t.Fatalf("clear ward outline = %d %v", status, body)
	}
	if cleared := resolve("11.5", "11.5"); cleared["group"].(map[string]any)["id"] != north.ID {
		t.Errorf("after clearing the ward's outline = %v, want North", cleared)
	}
	if status, body := do(t, srv, http.MethodGet, "/hierarchies/"+h.ID+"/resolve-point?lon=east&lat=1", nil); status != http.StatusBadRequest {
		t.Errorf("a longitude that is not a number = %d %v, want 400", status, body)
	}
}
