package routes_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestTheCodeFormatIsReadAndSetOverTheWire(t *testing.T) {
	mgr := newManager(t)
	srv := newServer(t, routes.Routes(mgr))

	status, body := do(t, srv, http.MethodGet, "/code-format", nil)
	if status != http.StatusOK || body["group_count"] != float64(3) {
		t.Fatalf("read the code format = %d %v, want three groups", status, body)
	}

	status, body = do(t, srv, http.MethodPut, "/code-format", map[string]any{"group_count": 4})
	if status != http.StatusOK || body["group_count"] != float64(4) {
		t.Fatalf("set the code format = %d %v, want four groups", status, body)
	}

	status, body = do(t, srv, http.MethodPut, "/code-format", map[string]any{"group_count": 0})
	if status != http.StatusBadRequest {
		t.Errorf("set no groups = %d %v, want 400", status, body)
	}

	status, body = do(t, srv, http.MethodPost, "/actors", map[string]any{"display_name": "Ami", "public_code": "M-AAA-AAA-AAA"})
	if status != http.StatusCreated {
		t.Fatalf("create actor = %d %v", status, body)
	}
	code, _ := body["public_code"].(string)
	if code == "M-AAA-AAA-AAA" || !regexp.MustCompile(`^M(-[2-9A-HJKMNP-Z]{3}){4}$`).MatchString(code) {
		t.Errorf("created actor's public code = %q, want one drawn by the server with four groups", code)
	}
}
