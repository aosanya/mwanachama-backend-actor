package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestAGroupPopulationIsRecordedByTheCallerAndReadBackOverTheWire(t *testing.T) {
	mgr := newManager(t)
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return "op-1" },
	}))
	g, err := mgr.CreateGroup(context.Background(), mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	status, _ := do(t, srv, http.MethodGet, "/groups/"+g.ID+"/population", nil)
	if status != http.StatusNotFound {
		t.Fatalf("read before any figure = %d, want 404", status)
	}
	status, body := do(t, srv, http.MethodPost, "/groups/"+g.ID+"/population", map[string]any{
		"population": 12000, "as_of": "2022-08-01", "source": "2022 census", "recorded_by": "someone-else",
	})
	if status != http.StatusCreated || body["recorded_by"] != "op-1" {
		t.Fatalf("record = %d %v, want 201 with the caller as recorder", status, body)
	}
	status, body = do(t, srv, http.MethodGet, "/groups/"+g.ID+"/population", nil)
	if status != http.StatusOK || body["population"] != float64(12000) || body["source"] != "2022 census" {
		t.Errorf("latest = %d %v, want the recorded figure", status, body)
	}
	status, _ = do(t, srv, http.MethodPost, "/groups/"+g.ID+"/population", map[string]any{
		"population": 10, "as_of": "2022-08-01",
	})
	if status != http.StatusBadRequest {
		t.Errorf("unsourced figure = %d, want 400", status)
	}
}

func TestAWithdrawnFigureIsSkippedByTheLatestReadAndWithdrawnByTheCaller(t *testing.T) {
	mgr := newManager(t)
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return "op-1" },
	}))
	ctx := context.Background()
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.RecordGroupPopulation(ctx, g.ID, 9000, "2019-08-01", "2019 census", "op-1"); err != nil {
		t.Fatalf("record: %v", err)
	}
	wrong, err := mgr.RecordGroupPopulation(ctx, g.ID, 90000, "2022-08-01", "typo", "op-1")
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	status, body := do(t, srv, http.MethodPost, "/groups/"+g.ID+"/population/"+wrong.ID+"/withdraw", map[string]any{})
	if status != http.StatusCreated || body["recorded_by"] != "op-1" || body["withdraws"] != wrong.ID {
		t.Fatalf("withdraw = %d %v, want 201 naming the figure and the caller", status, body)
	}
	status, body = do(t, srv, http.MethodGet, "/groups/"+g.ID+"/population", nil)
	if status != http.StatusOK || body["population"] != float64(9000) {
		t.Errorf("latest after withdraw = %d %v, want the 2019 figure", status, body)
	}
	status, _ = do(t, srv, http.MethodPost, "/groups/"+g.ID+"/population/"+wrong.ID+"/withdraw", map[string]any{})
	if status != http.StatusNotFound {
		t.Errorf("second withdraw = %d, want 404", status)
	}
}
