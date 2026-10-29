package routes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func getRoster(t *testing.T, base, query string) []map[string]any {
	t.Helper()
	resp, err := http.Get(base + query)
	if err != nil {
		t.Fatalf("GET %s: %v", query, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", query, resp.StatusCode)
	}
	var rows []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatalf("decode %s: %v", query, err)
	}
	return rows
}

func TestTheRosterRouteAnswersFiftyAtATimeAndTakesTheLastKey(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	for i := 0; i < 70; i++ {
		a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: fmt.Sprintf("Member %02d", i)})
		if err != nil {
			t.Fatalf("CreateActor: %v", err)
		}
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: group.ID}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
	}
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{CheckReferences: true}))
	path := srv.URL + "/groups/" + group.ID + "/assignments"

	first := getRoster(t, path, "")
	if len(first) != 50 {
		t.Fatalf("first page = %d rows, want 50", len(first))
	}
	last := first[49]["actor_id"].(string)
	rest := getRoster(t, path, "?after="+last)
	if len(rest) != 20 {
		t.Fatalf("second page = %d rows, want the remaining 20", len(rest))
	}
	if rest[0]["actor_id"].(string) <= last {
		t.Fatalf("second page starts at %v, not after %s", rest[0]["actor_id"], last)
	}
	small := getRoster(t, path, "?limit=5")
	if len(small) != 5 {
		t.Fatalf("limit=5 = %d rows, want 5", len(small))
	}
}
