package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func boundaryServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, err := actor.SpecFor("routestest")
	if err != nil {
		t.Fatal(err)
	}
	if err := actor.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	um, err := actor.NewUserManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	caller := func(context.Context) string { return "u-1" }
	for _, rt := range routes.RoutesFor(um, routes.Mount{Caller: caller}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func send(t *testing.T, method, url string, body any) (int, []byte) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, url, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(res.Body)
	return res.StatusCode, buf.Bytes()
}

func TestBoundaryRouteSetsAndTheLookupRouteFindsIt(t *testing.T) {
	srv := boundaryServer(t)
	status, body := send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Északi kerület"})
	if status >= 300 {
		t.Fatalf("create group = %d %s", status, body)
	}
	var g struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &g)
	square := map[string]any{"type": "Polygon", "coordinates": [][][]float64{{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}}}
	if status, body := send(t, http.MethodPut, srv.URL+"/groups/"+g.ID+"/boundary", map[string]any{"boundary": square}); status != http.StatusOK {
		t.Fatalf("set boundary = %d %s", status, body)
	}
	status, body = send(t, http.MethodGet, srv.URL+"/groups/at-point?lon=1&lat=1", nil)
	var found []struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &found)
	if status != http.StatusOK || len(found) != 1 || found[0].ID != g.ID {
		t.Fatalf("at-point inside = %d %s", status, body)
	}
	status, body = send(t, http.MethodGet, srv.URL+"/groups/at-point?lon=9&lat=9", nil)
	if status != http.StatusOK || string(bytes.TrimSpace(body)) != "[]" {
		t.Fatalf("at-point outside = %d %s", status, body)
	}
	if status, _ := send(t, http.MethodGet, srv.URL+"/groups/at-point?lon=999&lat=0", nil); status != http.StatusBadRequest {
		t.Fatalf("at-point off the earth = %d, want 400", status)
	}
}

func TestBoundaryRouteAnswers400ForAPoint(t *testing.T) {
	srv := boundaryServer(t)
	_, body := send(t, http.MethodPost, srv.URL+"/groups", map[string]any{"name": "Déli kerület"})
	var g struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &g)
	point := map[string]any{"type": "Point", "coordinates": []float64{1, 1}}
	if status, body := send(t, http.MethodPut, srv.URL+"/groups/"+g.ID+"/boundary", map[string]any{"boundary": point}); status != http.StatusBadRequest {
		t.Fatalf("a point as a boundary = %d %s, want 400", status, body)
	}
}
