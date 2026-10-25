package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestAMemberCodesScreenNameOverTheWireCarriesNothingElse(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina", FirstName: "Amina", LastName: "Odhiambo"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return "leader" },
	}))
	status, body := do(t, srv, http.MethodGet, "/actors/by-code/screen-name?code="+a.PublicCode, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body %s", status, body)
	}
	if len(body) != 1 || body["display_name"] != "Amina" {
		t.Errorf("body = %v, want only the screen name", body)
	}
	status, _ = do(t, srv, http.MethodGet, "/actors/by-code/screen-name?code=M-AAA-AAA-AAA", nil)
	if status != http.StatusNotFound {
		t.Errorf("unknown code status = %d, want 404", status)
	}
}
