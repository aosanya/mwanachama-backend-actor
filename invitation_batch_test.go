package mwanachamaactor_test

import (
	"context"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestCheckingAnInvitationBatchGivesEachRowItsReason(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	member, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina", Attributes: map[string]any{"email": "amina@example.org"}})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: member.ID, GroupID: g.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	outsider, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka", Attributes: map[string]any{"email": "baraka@example.org"}})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	code := " " + g.PublicCode + " "

	rows := []mwanachamaactor.InvitationBatchRow{
		{FirstName: "Chege", LastName: "Kamau", Email: "Chege@Example.org", GroupCode: code},
		{FirstName: "Baraka", LastName: "Otieno", Email: "baraka@example.org", GroupCode: g.PublicCode},
		{FirstName: "Amina", LastName: "Wanjiru", Email: "amina@example.org", GroupCode: g.PublicCode},
		{FirstName: "Chege", LastName: "Kamau", Email: "chege@example.org", GroupCode: g.PublicCode},
		{FirstName: "Dida", LastName: "Halake", Email: "dida@", GroupCode: g.PublicCode},
		{FirstName: "Esi", LastName: "Mensah", Email: "esi@example.org", GroupCode: "G-NOPE-NOPE"},
		{FirstName: "", LastName: "Mensah", Email: "f@example.org", GroupCode: g.PublicCode},
	}
	got, err := mgr.CheckInvitationBatch(ctx, rows)
	if err != nil {
		t.Fatalf("CheckInvitationBatch: %v", err)
	}
	want := []string{
		models.BatchRowReady,
		models.BatchRowReady,
		models.BatchRowAlreadyMember,
		models.BatchRowDuplicate,
		models.BatchRowBadEmail,
		models.BatchRowUnknownGroup,
		models.BatchRowMissingName,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d checks, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Row != i+1 || got[i].Status != w {
			t.Errorf("row %d = %+v, want status %q", i+1, got[i], w)
		}
	}
	if got[0].GroupID != g.ID || got[0].ActorID != "" || got[0].Input.Email != "chege@example.org" {
		t.Errorf("ready row with no account = %+v", got[0])
	}
	if got[1].ActorID != outsider.ID {
		t.Errorf("ready row naming an account = %+v, want actor %s", got[1], outsider.ID)
	}
}
