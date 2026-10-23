package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestOverTheWireCheckingAnInvitationListWritesNothingAndAnswersEachRow(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return "operator-1" },
	}))

	raw, err := json.Marshal(map[string]any{"rows": []map[string]any{
		{"first_name": "Chege", "last_name": "Kamau", "email": "Chege@Example.org", "group_code": group.PublicCode},
		{"first_name": "Chege", "last_name": "Kamau", "email": "chege@example.org", "group_code": group.PublicCode},
		{"first_name": "Esi", "last_name": "Mensah", "email": "esi@", "group_code": group.PublicCode},
	}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(srv.URL+"/invitations/batch/check", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	var got []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.StatusCode != http.StatusOK || len(got) != 3 {
		t.Fatalf("check = %d %v, want 200 with three rows", resp.StatusCode, got)
	}
	want := []string{"ready", "duplicate", "bad_email"}
	for i, w := range want {
		if got[i]["status"] != w || got[i]["row"] != float64(i+1) {
			t.Errorf("row %d = %v, want status %s", i+1, got[i], w)
		}
	}
	if got[0]["group_id"] != group.ID {
		t.Errorf("ready row group_id = %v, want %s", got[0]["group_id"], group.ID)
	}

	invitations, err := mgr.ListInvitationsForGroup(ctx, group.ID, false)
	if err != nil {
		t.Fatalf("ListInvitationsForGroup: %v", err)
	}
	if len(invitations) != 0 {
		t.Errorf("checking wrote %d invitations, want none", len(invitations))
	}
}
