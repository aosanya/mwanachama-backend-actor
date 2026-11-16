package routes_test

import (
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
