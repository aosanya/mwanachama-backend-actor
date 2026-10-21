package routes_test

import (
	"context"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestTheFormsWeekSplitBindsTheCallerNeverTheBody(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	me, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	other, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller:          func(context.Context) string { return me.ID },
		CheckReferences: true,
	}))

	status, body := do(t, srv, http.MethodGet, "/actors/mine/forms-week-split", nil)
	if status != http.StatusOK || body["forms_week_split"] != true || body["actor_id"] != me.ID {
		t.Fatalf("read = %d %v, want the caller's split on", status, body)
	}

	status, body = do(t, srv, http.MethodPut, "/actors/mine/forms-week-split", map[string]any{
		"forms_week_split": false,
		"actor_id":         other.ID,
	})
	if status != http.StatusOK || body["actor_id"] != me.ID || body["forms_week_split"] != false {
		t.Fatalf("set = %d %v, want the caller's split off", status, body)
	}
	if mine, _ := mgr.GetMyFormsWeekSplit(ctx, me.ID); mine.FormsWeekSplit {
		t.Errorf("my split = on, want off")
	}
	if theirs, _ := mgr.GetMyFormsWeekSplit(ctx, other.ID); !theirs.FormsWeekSplit {
		t.Errorf("the body-named actor's split = off, want on untouched")
	}

	status, body = do(t, srv, http.MethodGet, "/actors/mine/forms-week-split", nil)
	if status != http.StatusOK || body["forms_week_split"] != false {
		t.Errorf("read after set = %d %v, want off", status, body)
	}
}

func TestSettingTheFormsWeekSplitWithNoRecordAnswersNotFound(t *testing.T) {
	mgr := newManager(t)
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return "no-record" },
	}))
	if status, body := do(t, srv, http.MethodPut, "/actors/mine/forms-week-split", map[string]any{"forms_week_split": false}); status != http.StatusNotFound {
		t.Errorf("set with no record = %d %v, want 404", status, body)
	}
}
