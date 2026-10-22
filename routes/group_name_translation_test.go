package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestAGroupNameIsTranslatedReviewedAndReadInALanguageOverTheWire(t *testing.T) {
	mgr := newManager(t)
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return "op-1" },
	}))
	g, err := mgr.CreateGroup(context.Background(), mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	status, body := do(t, srv, http.MethodPut, "/groups/"+g.ID+"/names/sw", map[string]any{"name": "Kata 1"})
	if status != http.StatusOK || body["reviewed"] != false {
		t.Fatalf("set translation = %d %v, want it unreviewed", status, body)
	}
	status, body = do(t, srv, http.MethodGet, "/groups/"+g.ID+"/in/sw", nil)
	if status != http.StatusOK || body["name"] != "Ward 1" {
		t.Fatalf("read before review = %d %v, want the original", status, body)
	}
	status, body = do(t, srv, http.MethodPost, "/groups/"+g.ID+"/names/sw/review", map[string]any{"reviewed_by": "someone-else"})
	if status != http.StatusOK || body["reviewed_by"] != "op-1" {
		t.Fatalf("review = %d %v, want the caller as reviewer", status, body)
	}
	status, body = do(t, srv, http.MethodGet, "/groups/"+g.ID+"/in/sw", nil)
	if status != http.StatusOK || body["name"] != "Kata 1" {
		t.Errorf("read after review = %d %v, want the translation", status, body)
	}
	status, _ = do(t, srv, http.MethodPut, "/groups/"+g.ID+"/names/not%20a%20tag", map[string]any{"name": "x"})
	if status != http.StatusBadRequest {
		t.Errorf("bad language = %d, want 400", status)
	}
	status, _ = do(t, srv, http.MethodPost, "/groups/"+g.ID+"/names/hu/review", nil)
	if status != http.StatusNotFound {
		t.Errorf("review a missing translation = %d, want 404", status)
	}
}
