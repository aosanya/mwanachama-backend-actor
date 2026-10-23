package mwanachamaactor_test

import (
	"context"
	"errors"
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

func TestSendingAnInvitationBatchInvitesEachReadyRowWithoutALeaderAndOnlyOnce(t *testing.T) {
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
	chege, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Chege"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	rows := []mwanachamaactor.InvitationBatchRow{
		{FirstName: "Chege", LastName: "Kamau", Email: "chege@example.org", GroupCode: g.PublicCode, ActorID: chege.ID},
		{FirstName: "Amina", LastName: "Wanjiru", Email: "amina@example.org", GroupCode: g.PublicCode},
		{FirstName: "Chege", LastName: "Kamau", Email: "chege@example.org", GroupCode: g.PublicCode, ActorID: chege.ID},
		{FirstName: "Esi", LastName: "Mensah", Email: "esi@example.org", GroupCode: g.PublicCode},
		{FirstName: "Dida", LastName: "Halake", Email: "dida@example.org", GroupCode: g.PublicCode, ActorID: "nobody"},
		{FirstName: "Fatuma", LastName: "Ali", Email: "fatuma@", GroupCode: g.PublicCode},
	}

	first, err := mgr.SendInvitationBatch(ctx, rows, "owner-not-a-leader")
	if err != nil {
		t.Fatalf("SendInvitationBatch: %v", err)
	}
	want := []string{
		models.BatchRowSent,
		models.BatchRowAlreadyMember,
		models.BatchRowDuplicate,
		models.BatchRowNoActor,
		models.BatchRowNoActor,
		models.BatchRowBadEmail,
	}
	if len(first) != len(want) {
		t.Fatalf("got %d results, want %d", len(first), len(want))
	}
	for i, w := range want {
		if first[i].Row != i+1 || first[i].Status != w {
			t.Errorf("row %d = %+v, want status %q", i+1, first[i], w)
		}
		if (w == models.BatchRowSent) != (first[i].Invitation != nil) {
			t.Errorf("row %d invitation = %v, want one only on a sent row", i+1, first[i].Invitation)
		}
	}
	sent := first[0].Invitation
	if sent.ActorID != chege.ID || sent.GroupID != g.ID || sent.InvitedBy != "owner-not-a-leader" || sent.State != models.InvitationPending {
		t.Errorf("sent invitation = %+v, want a pending invitation of Chege to the group by the caller", sent)
	}

	second, err := mgr.SendInvitationBatch(ctx, rows[:1], "owner-not-a-leader")
	if err != nil {
		t.Fatalf("second SendInvitationBatch: %v", err)
	}
	if second[0].Status != models.BatchRowSent || second[0].Invitation == nil || second[0].Invitation.ID != sent.ID || second[0].Invitation.InvitedAt != sent.InvitedAt {
		t.Errorf("second send = %+v, want the invitation already pending, %+v", second[0], sent)
	}
	invitations, err := mgr.ListInvitationsForGroup(ctx, g.ID, false)
	if err != nil {
		t.Fatalf("ListInvitationsForGroup: %v", err)
	}
	if len(invitations) != 1 {
		t.Errorf("group holds %d invitations, want exactly one", len(invitations))
	}
}

func TestSendingAnInvitationBatchNeedsWhoeverSendsIt(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.SendInvitationBatch(context.Background(), nil, " "); !errors.Is(err, mwanachamaactor.ErrInvalidInvitation) {
		t.Errorf("err = %v, want ErrInvalidInvitation", err)
	}
}
