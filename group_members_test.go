package mwanachamaactor_test

import (
	"context"
	"fmt"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestAGroupsMembersComeByNameWithTheTotal(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, _ := rosterOf(t, mgr, 200)
	other := mustGroup(t, mgr, "Ward B")
	stranger := mustActor(t, mgr, "Aaa Stranger")
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: stranger.ID, GroupID: other.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}

	page, err := mgr.ListGroupMembers(ctx, g.ID, 0, 3, "")
	if err != nil {
		t.Fatalf("ListGroupMembers: %v", err)
	}
	if page.Total != 200 {
		t.Fatalf("total = %d, want 200", page.Total)
	}
	for i, item := range page.Items {
		if want := fmt.Sprintf("Member %03d", i); item.DisplayName != want || item.ActorID == "" {
			t.Fatalf("item %d = %+v, want %q with an actor id", i, item, want)
		}
	}
	if len(page.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(page.Items))
	}
}

func TestAGroupsMembersPageByOffset(t *testing.T) {
	mgr := newTestManager(t)
	g, _ := rosterOf(t, mgr, 120)

	page, err := mgr.ListGroupMembers(context.Background(), g.ID, 100, 50, "")
	if err != nil {
		t.Fatalf("ListGroupMembers: %v", err)
	}
	if len(page.Items) != 20 || page.Items[0].DisplayName != "Member 100" || page.Items[19].DisplayName != "Member 119" || page.Total != 120 {
		t.Fatalf("page = %d items from %q, total %d, want the last 20 from Member 100 of 120", len(page.Items), page.Items[0].DisplayName, page.Total)
	}
}

func TestAGroupsMembersSearchCountsOnlyTheMatches(t *testing.T) {
	mgr := newTestManager(t)
	g, _ := rosterOf(t, mgr, 120)

	page, err := mgr.ListGroupMembers(context.Background(), g.ID, 0, 5, "member 05")
	if err != nil {
		t.Fatalf("ListGroupMembers: %v", err)
	}
	if page.Total != 10 || len(page.Items) != 5 || page.Items[0].DisplayName != "Member 050" {
		t.Fatalf("search = %d items first %q, total %d, want 5 items from Member 050 of 10", len(page.Items), page.Items[0].DisplayName, page.Total)
	}
}

func TestAGroupsMembersAreCappedAndAnUnknownGroupIsEmpty(t *testing.T) {
	mgr := newTestManager(t)
	g, _ := rosterOf(t, mgr, mwanachamaactor.MaxRosterPageSize+5)

	page, err := mgr.ListGroupMembers(context.Background(), g.ID, 0, 10000, "")
	if err != nil || len(page.Items) != mwanachamaactor.MaxRosterPageSize || page.Total != mwanachamaactor.MaxRosterPageSize+5 {
		t.Fatalf("page = %d items, total %d, err %v, want the cap of %d of %d", len(page.Items), page.Total, err, mwanachamaactor.MaxRosterPageSize, mwanachamaactor.MaxRosterPageSize+5)
	}
	none, err := mgr.ListGroupMembers(context.Background(), "no-such-group", 0, 3, "")
	if err != nil || len(none.Items) != 0 || none.Total != 0 {
		t.Fatalf("unknown group = %+v, err %v, want empty", none, err)
	}
}
