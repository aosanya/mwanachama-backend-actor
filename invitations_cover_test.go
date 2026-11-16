package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func newManagerWith(t *testing.T, opts ...actor.Option) actor.UserManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := actor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := actor.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	um, err := actor.NewUserManager(db, s, opts...)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return um
}

func TestAnInvitationLifetimeMustBePositive(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	s, _ := actor.SpecFor("test")
	if _, err := actor.NewUserManager(db, s, actor.WithInvitationLifetime(0)); err == nil {
		t.Error("a zero lifetime was accepted")
	}
	if _, err := actor.NewUserManager(nil, s); err == nil {
		t.Error("a nil database was accepted")
	}
}

func TestOnlyALeaderInvitesAndExactlyOneWayOfNamingTheInvitee(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.person.ID, ""); !errors.Is(err, actor.ErrNotGroupLeader) {
		t.Errorf("a non-leader inviting: %v", err)
	}
	if _, err := w.um.InviteToGroup(ctx, "", w.person.ID, "", w.leader.ID, ""); !errors.Is(err, actor.ErrInvalidInvitation) {
		t.Errorf("no group: %v", err)
	}
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", "", ""); !errors.Is(err, actor.ErrInvalidInvitation) {
		t.Errorf("no inviter: %v", err)
	}
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, "", "", w.leader.ID, ""); !errors.Is(err, actor.ErrInvalidInvitation) {
		t.Errorf("no invitee: %v", err)
	}
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, w.person.PublicCode, w.leader.ID, ""); !errors.Is(err, actor.ErrInvalidInvitation) {
		t.Errorf("two ways: %v", err)
	}
	if _, err := w.um.InviteToGroup(ctx, "nowhere", w.person.ID, "", w.leader.ID, ""); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown group: %v", err)
	}
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, "nobody", "", w.leader.ID, ""); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown invitee: %v", err)
	}
}

func TestAnInvitationExpiryMustBeATimeInTheFuture(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.leader.ID, "next Tuesday"); !errors.Is(err, actor.ErrInvalidInvitation) {
		t.Errorf("not a time: %v", err)
	}
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.leader.ID, past); !errors.Is(err, actor.ErrInvalidInvitation) {
		t.Errorf("in the past: %v", err)
	}
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	inv, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.leader.ID, future)
	if err != nil || inv.State != models.InvitationPending || inv.ExpiresAt == "" {
		t.Fatalf("InviteToGroup = %+v %v", inv, err)
	}
}

func TestAnInviteeIsFoundByTheirPublicCodeAndAcceptingRegistersThem(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	inv, err := w.um.InviteToGroup(ctx, w.group.ID, "", " "+w.person.PublicCode+" ", w.leader.ID, "")
	if err != nil {
		t.Fatalf("InviteToGroup by code: %v", err)
	}
	if inv.ActorID != w.person.ID {
		t.Fatalf("invited %q, want %q", inv.ActorID, w.person.ID)
	}
	if _, err := w.um.AcceptInvitation(ctx, inv.ID, w.leader.ID); !errors.Is(err, actor.ErrInvitationNotFound) {
		t.Errorf("someone else answering: %v, want it to look absent", err)
	}
	if _, err := w.um.AcceptInvitation(ctx, inv.ID, ""); !errors.Is(err, actor.ErrInvalidInvitation) {
		t.Errorf("answer with no actor: %v", err)
	}
	accepted, err := w.um.AcceptInvitation(ctx, inv.ID, w.person.ID)
	if err != nil || accepted.State != models.InvitationAccepted || accepted.AnsweredAt == "" {
		t.Fatalf("AcceptInvitation = %+v %v", accepted, err)
	}
	held, err := w.um.ListGroupsForActor(ctx, w.person.ID)
	if err != nil || len(held) != 1 || held[0].GroupID != w.group.ID {
		t.Errorf("after accepting: %+v %v, want registered at the group", held, err)
	}
	if _, err := w.um.AcceptInvitation(ctx, inv.ID, w.person.ID); !errors.Is(err, actor.ErrInvitationAnswered) {
		t.Errorf("answering twice: %v", err)
	}
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.leader.ID, ""); !errors.Is(err, actor.ErrAlreadyInGroup) {
		t.Errorf("inviting a member: %v", err)
	}
}

func TestDecliningAnInvitationRegistersNobodyAndAllowsAFreshInvitation(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	inv, _ := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.leader.ID, "")
	declined, err := w.um.DeclineInvitation(ctx, inv.ID, w.person.ID)
	if err != nil || declined.State != models.InvitationDeclined {
		t.Fatalf("DeclineInvitation = %+v %v", declined, err)
	}
	if held, _ := w.um.ListGroupsForActor(ctx, w.person.ID); len(held) != 0 {
		t.Errorf("a declined invitation registered the actor: %+v", held)
	}
	again, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.leader.ID, "")
	if err != nil {
		t.Fatalf("inviting again: %v", err)
	}
	if again.ID != inv.ID || again.State != models.InvitationPending || again.AnsweredAt != "" {
		t.Errorf("re-invited = %+v, want the same invitation pending again", again)
	}
}

func TestALapsedInvitationCannotBeAnswered(t *testing.T) {
	short := newManagerWith(t, actor.WithInvitationLifetime(time.Millisecond))
	ctx := context.Background()
	g, _ := short.CreateGroup(ctx, actor.Group{Name: "Rövid Kör"})
	leader, _ := short.CreateActor(ctx, actor.Actor{DisplayName: "Vezető Vera"})
	invitee, _ := short.CreateActor(ctx, actor.Actor{DisplayName: "Meghívott Mór"})
	kind, _ := short.CreateRoleKind(ctx, actor.RoleKind{Name: "Vezető"})
	short.SetRoleKindLeadsGroup(ctx, kind.ID, true)
	short.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: leader.ID, GroupID: g.ID, KindID: kind.ID})
	inv, err := short.InviteToGroup(ctx, g.ID, invitee.ID, "", leader.ID, "")
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := short.AcceptInvitation(ctx, inv.ID, invitee.ID); !errors.Is(err, actor.ErrInvitationLapsed) {
		t.Errorf("accepting after the lifetime: %v", err)
	}
	open, err := short.ListInvitationsForGroup(ctx, g.ID, true)
	if err != nil || len(open) != 0 {
		t.Errorf("open invitations = %+v %v, want none once lapsed", open, err)
	}
	all, err := short.ListInvitationsForGroup(ctx, g.ID, false)
	if err != nil || len(all) != 1 {
		t.Errorf("all invitations = %+v %v, want the lapsed one still listed", all, err)
	}
}

func TestInvitationsListByGroupAndByActorOldestFirst(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	other, _ := w.um.CreateGroup(ctx, actor.Group{Name: "Zöld Kör"})
	kind, _ := w.um.CreateRoleKind(ctx, actor.RoleKind{Name: "Zöld vezető"})
	w.um.SetRoleKindLeadsGroup(ctx, kind.ID, true)
	w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.leader.ID, GroupID: other.ID, KindID: kind.ID})
	first, _ := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.leader.ID, "")
	second, _ := w.um.InviteToGroup(ctx, other.ID, w.person.ID, "", w.leader.ID, "")
	mine, err := w.um.ListInvitationsForActor(ctx, w.person.ID)
	if err != nil || len(mine) != 2 || mine[0].ID != first.ID || mine[1].ID != second.ID {
		t.Errorf("for actor = %+v %v, want oldest first", mine, err)
	}
	forGroup, err := w.um.ListInvitationsForGroup(ctx, w.group.ID, true)
	if err != nil || len(forGroup) != 1 || forGroup[0].ID != first.ID {
		t.Errorf("for group = %+v %v", forGroup, err)
	}
	got, err := w.um.GetInvitation(ctx, first.ID)
	if err != nil || got.GroupID != w.group.ID {
		t.Errorf("GetInvitation = %+v %v", got, err)
	}
	if _, err := w.um.GetInvitation(ctx, "nope"); !errors.Is(err, actor.ErrInvitationNotFound) {
		t.Errorf("unknown invitation: %v", err)
	}
}

func TestAnOutsiderCanNeitherBeInvitedNorJoin(t *testing.T) {
	w := newJoinWorld(t, models.JoinOpen)
	ctx := context.Background()
	kind, err := w.um.CreateRoleKind(ctx, actor.RoleKind{Name: "Külsős", Outsider: true})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if _, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.person.ID, GroupID: w.group.ID, KindID: kind.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if _, err := w.um.InviteToGroup(ctx, w.group.ID, w.person.ID, "", w.leader.ID, ""); !errors.Is(err, actor.ErrOutsiderRole) {
		t.Errorf("inviting an outsider: %v", err)
	}
	if _, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, ""); !errors.Is(err, actor.ErrOutsiderRole) {
		t.Errorf("an outsider joining: %v", err)
	}
}

func TestFindingAGroupByCodeSaysWhereTheCallerStands(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	if _, err := w.um.FindGroupByCode(ctx, w.group.PublicCode, ""); !errors.Is(err, actor.ErrInvalidJoinRequest) {
		t.Errorf("no actor: %v", err)
	}
	if _, err := w.um.FindGroupByCode(ctx, "", w.person.ID); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("no code: %v", err)
	}
	if _, err := w.um.FindGroupByCode(ctx, "ZZZZ-ZZZZ", w.person.ID); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown code: %v", err)
	}
	outside, err := w.um.FindGroupByCode(ctx, w.group.PublicCode, w.person.ID)
	if err != nil || outside.State != models.CallerIsOutside || outside.Group.ID != w.group.ID {
		t.Fatalf("outside = %+v %v", outside, err)
	}
	requested, err := w.um.JoinGroupByCode(ctx, " "+w.group.PublicCode+" ", w.person.ID, "Kód alapján")
	if err != nil || requested.Status != models.JoinRequested {
		t.Fatalf("JoinGroupByCode = %+v %v", requested, err)
	}
	pending, _ := w.um.FindGroupByCode(ctx, w.group.PublicCode, w.person.ID)
	if pending.State != models.CallerRequested || pending.Request == nil {
		t.Errorf("pending = %+v", pending)
	}
	w.um.DeclineJoinRequest(ctx, requested.Request.ID, w.leader.ID, false, "")
	declined, _ := w.um.FindGroupByCode(ctx, w.group.PublicCode, w.person.ID)
	if declined.State != models.CallerDeclined {
		t.Errorf("declined = %+v", declined)
	}
	if _, err := w.um.JoinGroupByCode(ctx, "ZZZZ-ZZZZ", w.person.ID, ""); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("join by an unknown code: %v", err)
	}
}

func TestAHiddenGroupCanStillBeJoinedByItsCode(t *testing.T) {
	w := newJoinWorld(t, models.JoinOpen)
	ctx := context.Background()
	w.um.SetGroupHidden(ctx, w.group.ID, true)
	out, err := w.um.JoinGroupByCode(ctx, w.group.PublicCode, w.person.ID, "")
	if err != nil || out.Status != models.JoinedNow {
		t.Fatalf("JoinGroupByCode on a hidden group = %+v %v", out, err)
	}
	member, _ := w.um.FindGroupByCode(ctx, w.group.PublicCode, w.person.ID)
	if member.State != models.CallerIsMember {
		t.Errorf("state = %q, want member", member.State)
	}
}
