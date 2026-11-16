package routes_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestOverlongAndBlankNamesAreAnswered400OverTheWire(t *testing.T) {
	srv := newServer(t, routes.Routes(newManager(t)))
	for _, create := range []struct{ label, path, field string }{
		{"group", "/groups", "name"},
		{"actor", "/actors", "display_name"},
		{"hierarchy", "/hierarchies", "name"},
	} {
		for name, value := range map[string]string{
			"201 letters":    strings.Repeat("a", 201),
			"100000 letters": strings.Repeat("a", 100000),
			"five spaces":    "     ",
			"empty":          "",
		} {
			if value == "" && create.label == "actor" {
				continue
			}
			status, body := do(t, srv, http.MethodPost, create.path, map[string]any{create.field: value})
			if status != http.StatusBadRequest {
				t.Errorf("create %s with %s = %d %.120v, want 400", create.label, name, status, body)
			}
		}
		status, body := do(t, srv, http.MethodPost, create.path, map[string]any{create.field: strings.Repeat("a", 200)})
		if status != http.StatusCreated {
			t.Errorf("create %s with 200 letters = %d %.120v, want 201", create.label, status, body)
		}
	}
}

func TestEditingAGroupTrimsBeforeCountingAndStoring(t *testing.T) {
	srv := newServer(t, routes.Routes(newManager(t)))
	status, made := do(t, srv, http.MethodPost, "/groups", map[string]any{"name": "Ward 1"})
	if status != http.StatusCreated {
		t.Fatalf("create = %d %v", status, made)
	}
	id, _ := made["id"].(string)
	for label, c := range map[string]struct{ typed, stored string }{
		"short":  {"  Padded  ", "Padded"},
		"200":    {" " + strings.Repeat("a", 200) + " ", strings.Repeat("a", 200)},
		"accent": {" " + strings.Repeat("é", 200) + " ", strings.Repeat("é", 200)},
	} {
		status, body := do(t, srv, http.MethodPatch, "/groups/"+id, map[string]any{"name": c.typed})
		if status != http.StatusOK || body["name"] != c.stored {
			t.Errorf("%s: PATCH = %d name %.40v, want 200 and %.40q", label, status, body["name"], c.stored)
		}
	}
}

func TestAnOverlongGroupNameIsRefusedBeforeTheGroupIsLookedUp(t *testing.T) {
	overlong := strings.Repeat("a", 201)
	for label, name := range map[string]string{
		"plain":  overlong,
		"padded": " " + overlong + " ",
	} {
		srv := newServer(t, routes.Routes(newManager(t)))
		status, answer := do(t, srv, http.MethodPatch, "/groups/nowhere", map[string]any{"name": name})
		if status != http.StatusBadRequest || !strings.Contains(fmt.Sprint(answer), "a name may be at most 200 characters") {
			t.Errorf("%s: PATCH /groups/nowhere = %d %v, want 400 and \"a name may be at most 200 characters\"", label, status, answer)
		}
	}
}

func TestAnOverlongGroupNameIsRefusedBeforeTheGroupIsLookedUpWithReferencesChecked(t *testing.T) {
	srv := serverFor(t, managerOnMemory(t), routes.Mount{CheckReferences: true, Caller: func(context.Context) string { return "u-1" }})
	status, answer := send(t, http.MethodPatch, srv.URL+"/groups/nowhere", map[string]any{"name": strings.Repeat("a", 201)})
	if status != http.StatusBadRequest || !strings.Contains(string(answer), "a name may be at most 200 characters") {
		t.Errorf("PATCH /groups/nowhere = %d %s, want 400 and \"a name may be at most 200 characters\"", status, answer)
	}
}
