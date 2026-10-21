package mwanachamaactor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestALeaderInvitesByTheInviteesMemberCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)
	if !strings.HasPrefix(a.PublicCode, "M-") {
		t.Fatalf("seeded actor public code = %q, want M- and random groups", a.PublicCode)
	}

	typed := "  " + strings.ToLower(a.PublicCode) + " "
	inv, err := mgr.InviteToGroup(ctx, g.ID, "", typed, "leader-1", "")
	if err != nil {
		t.Fatalf("InviteToGroup by code: %v", err)
	}
	if inv.ActorID != a.ID || inv.GroupID != g.ID {
		t.Fatalf("invitation = %+v, want one for %s at %s", inv, a.ID, g.ID)
	}
	if _, err := mgr.AcceptInvitation(ctx, inv.ID, a.ID); err != nil {
		t.Fatalf("the code's actor accepting: %v", err)
	}
	if !registered(t, mgr, a.ID, g.ID) {
		t.Error("accepting an invitation issued by code did not register the actor")
	}
}

func TestInvitingByIDAndByCodeRenewTheSameInvitation(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)

	byID, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "", "leader-1", "")
	if err != nil {
		t.Fatalf("InviteToGroup by id: %v", err)
	}
	byCode, err := mgr.InviteToGroup(ctx, g.ID, "", a.PublicCode, "leader-2", "")
	if err != nil {
		t.Fatalf("InviteToGroup by code: %v", err)
	}
	if byCode.ID != byID.ID || byCode.InvitedBy != "leader-2" {
		t.Errorf("by code = %+v, want the same invitation %s renewed by leader-2", byCode, byID.ID)
	}
}

func TestAnUnknownMemberCodeIsRefusedLikeAnUnknownActor(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)

	for _, code := range []string{"M-222-333-444", a.Code, "AC-9999"} {
		if _, err := mgr.InviteToGroup(ctx, g.ID, "", code, "leader-1", ""); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
			t.Errorf("code %q err = %v, want ErrActorNotFound", code, err)
		}
	}
	if _, err := mgr.InviteToGroup(ctx, g.ID, "missing", "", "leader-1", ""); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
		t.Errorf("an unknown id err = %v, want ErrActorNotFound", err)
	}
}

func TestAnInviteeIsNamedByExactlyOneOfIDAndCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)

	if _, err := mgr.InviteToGroup(ctx, g.ID, "", "  ", "leader-1", ""); !errors.Is(err, mwanachamaactor.ErrInvalidInvitation) {
		t.Errorf("no invitee err = %v, want ErrInvalidInvitation", err)
	}
	if _, err := mgr.InviteToGroup(ctx, g.ID, a.ID, a.PublicCode, "leader-1", ""); !errors.Is(err, mwanachamaactor.ErrInvalidInvitation) {
		t.Errorf("both id and code err = %v, want ErrInvalidInvitation", err)
	}
}

func TestACodeInviteStillNeedsALeader(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)

	if _, err := mgr.InviteToGroup(ctx, g.ID, "", a.PublicCode, "stranger", ""); !errors.Is(err, mwanachamaactor.ErrNotGroupLeader) {
		t.Errorf("a non-leader inviting by code err = %v, want ErrNotGroupLeader", err)
	}
}
