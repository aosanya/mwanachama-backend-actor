package mwanachamaactor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

type joinWorld struct {
	um     actor.UserManager
	group  actor.Group
	leader actor.Actor
	person actor.Actor
}

func newJoinWorld(t *testing.T, mode string) joinWorld {
	t.Helper()
	um := newManager(t)
	ctx := context.Background()
	g, err := um.CreateGroup(ctx, actor.Group{Name: "Kertbarátok", JoinMode: mode})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	leader, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Nagy Péter"})
	if err != nil {
		t.Fatalf("CreateActor leader: %v", err)
	}
	person, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Kiss Zsuzsa"})
	if err != nil {
		t.Fatalf("CreateActor person: %v", err)
	}
	kind, err := um.CreateRoleKind(ctx, actor.RoleKind{Name: "Csoportvezető"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if _, err := um.SetRoleKindLeadsGroup(ctx, kind.ID, true); err != nil {
		t.Fatalf("SetRoleKindLeadsGroup: %v", err)
	}
	if _, err := um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: leader.ID, GroupID: g.ID, KindID: kind.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	return joinWorld{um: um, group: g, leader: leader, person: person}
}

func TestAConfirmedGroupTurnsAJoinIntoAPendingRequest(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	out, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "  Szeretnék segíteni  ")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if out.Status != models.JoinRequested || out.Request == nil || !out.Request.Pending() {
		t.Fatalf("outcome = %+v, want a pending request", out)
	}
	if out.Request.Message != "Szeretnék segíteni" {
		t.Errorf("message = %q, want it trimmed", out.Request.Message)
	}
	again, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "Még egyszer")
	if err != nil {
		t.Fatalf("JoinGroup again: %v", err)
	}
	if again.Request.ID != out.Request.ID || again.Request.Message != "Még egyszer" {
		t.Errorf("asking again = %+v, want the same request carrying the new message", again.Request)
	}
	held, err := w.um.ListGroupsForActor(ctx, w.person.ID)
	if err != nil || len(held) != 0 {
		t.Errorf("a pending request must not register the actor: %v %v", held, err)
	}
}

func TestAnOpenGroupRegistersTheActorAtOnce(t *testing.T) {
	w := newJoinWorld(t, models.JoinOpen)
	ctx := context.Background()
	out, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if out.Status != models.JoinedNow || out.Assignment == nil || out.Assignment.GroupID != w.group.ID {
		t.Fatalf("outcome = %+v, want joined now", out)
	}
	again, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "")
	if err != nil || again.Status != models.AlreadyAMember {
		t.Errorf("second join = %+v %v, want already a member", again, err)
	}
}

func TestJoiningIsRefusedForBadInput(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	if _, err := w.um.JoinGroup(ctx, "", w.person.ID, ""); !errors.Is(err, actor.ErrInvalidJoinRequest) {
		t.Errorf("no group: %v", err)
	}
	if _, err := w.um.JoinGroup(ctx, w.group.ID, "", ""); !errors.Is(err, actor.ErrInvalidJoinRequest) {
		t.Errorf("no actor: %v", err)
	}
	if _, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, strings.Repeat("ű", 501)); !errors.Is(err, actor.ErrJoinMessageTooLong) {
		t.Errorf("long message: %v", err)
	}
	if _, err := w.um.JoinGroup(ctx, "nowhere", w.person.ID, ""); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown group: %v", err)
	}
	if _, err := w.um.JoinGroup(ctx, w.group.ID, "nobody", ""); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown actor: %v", err)
	}
}

func TestAHiddenGroupCannotBeJoinedByItsId(t *testing.T) {
	w := newJoinWorld(t, models.JoinOpen)
	ctx := context.Background()
	if _, err := w.um.SetGroupHidden(ctx, w.group.ID, true); err != nil {
		t.Fatalf("SetGroupHidden: %v", err)
	}
	if _, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, ""); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("join hidden by id: %v, want group not found", err)
	}
	joinable, err := w.um.ListJoinableGroups(ctx, w.person.ID)
	if err != nil {
		t.Fatalf("ListJoinableGroups: %v", err)
	}
	for _, g := range joinable {
		if g.ID == w.group.ID {
			t.Errorf("a hidden group is listed as joinable")
		}
	}
}

func TestJoinableGroupsLeaveOutWhereTheActorAlreadyBelongs(t *testing.T) {
	w := newJoinWorld(t, models.JoinOpen)
	ctx := context.Background()
	other, err := w.um.CreateGroup(ctx, actor.Group{Name: "Zöld Kör", JoinMode: models.JoinOpen})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, ""); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	joinable, err := w.um.ListJoinableGroups(ctx, w.person.ID)
	if err != nil {
		t.Fatalf("ListJoinableGroups: %v", err)
	}
	if len(joinable) != 1 || joinable[0].ID != other.ID {
		t.Errorf("joinable = %+v, want only %q", joinable, other.Name)
	}
}

func TestSettingAJoinModeChecksTheModeAndTheGroup(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	if _, err := w.um.SetGroupJoinMode(ctx, w.group.ID, "whenever"); !errors.Is(err, actor.ErrInvalidJoinMode) {
		t.Errorf("bad mode: %v", err)
	}
	if _, err := w.um.SetGroupJoinMode(ctx, "nowhere", models.JoinOpen); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown group: %v", err)
	}
	g, err := w.um.SetGroupJoinMode(ctx, w.group.ID, models.JoinOpen)
	if err != nil || !g.Open() {
		t.Fatalf("SetGroupJoinMode = %+v %v", g, err)
	}
	read, _ := w.um.GetGroup(ctx, w.group.ID)
	if read.JoinMode != models.JoinOpen {
		t.Errorf("stored mode = %q", read.JoinMode)
	}
}

func TestOnlyAGroupLeaderDecidesAJoinRequest(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	out, _ := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "")
	id := out.Request.ID
	if _, err := w.um.ApproveJoinRequest(ctx, id, w.person.ID); !errors.Is(err, actor.ErrNotGroupLeader) {
		t.Errorf("a non-leader approving: %v", err)
	}
	if _, err := w.um.ApproveJoinRequest(ctx, id, ""); !errors.Is(err, actor.ErrInvalidJoinRequest) {
		t.Errorf("no decider: %v", err)
	}
	if _, err := w.um.ApproveJoinRequest(ctx, "nope", w.leader.ID); !errors.Is(err, actor.ErrJoinRequestNotFound) {
		t.Errorf("unknown request: %v", err)
	}
	approved, err := w.um.ApproveJoinRequest(ctx, id, w.leader.ID)
	if err != nil {
		t.Fatalf("ApproveJoinRequest: %v", err)
	}
	if approved.State != models.JoinRequestApproved || approved.DecidedBy != w.leader.ID || approved.DecidedAt == "" {
		t.Errorf("approved = %+v", approved)
	}
	if _, err := w.um.DeclineJoinRequest(ctx, id, w.leader.ID, false, ""); !errors.Is(err, actor.ErrJoinRequestDecided) {
		t.Errorf("deciding twice: %v", err)
	}
}

func TestDecliningCarriesAReasonUnlessItIsQuiet(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	out, _ := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "")
	id := out.Request.ID
	if _, err := w.um.DeclineJoinRequest(ctx, id, w.leader.ID, true, "no"); !errors.Is(err, actor.ErrQuietDeclineReason) {
		t.Errorf("quiet with a reason: %v", err)
	}
	if _, err := w.um.DeclineJoinRequest(ctx, id, w.leader.ID, false, strings.Repeat("é", 501)); !errors.Is(err, actor.ErrDeclineReasonTooLong) {
		t.Errorf("long reason: %v", err)
	}
	declined, err := w.um.DeclineJoinRequest(ctx, id, w.leader.ID, false, "  Tele a csoport  ")
	if err != nil {
		t.Fatalf("DeclineJoinRequest: %v", err)
	}
	if declined.State != models.JoinRequestDeclined || declined.DeclineReason != "Tele a csoport" || declined.Silent {
		t.Errorf("declined = %+v", declined)
	}
}

func TestAskingAgainAfterADeclineIsAnAppeal(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	out, _ := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "")
	if _, err := w.um.DeclineJoinRequest(ctx, out.Request.ID, w.leader.ID, true, ""); err != nil {
		t.Fatalf("DeclineJoinRequest: %v", err)
	}
	again, err := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "Kérem, nézzék át")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	r := again.Request
	if r.ID != out.Request.ID || !r.Appeal || !r.Pending() || r.DecidedBy != "" || r.Silent {
		t.Errorf("appeal = %+v, want the same request pending again as an appeal", r)
	}
}

func TestTheHostCanDecideWithoutBeingALeader(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	out, _ := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "")
	approved, err := w.um.OperatorApproveJoinRequest(ctx, out.Request.ID, "host-operator")
	if err != nil || approved.State != models.JoinRequestApproved || approved.DecidedBy != "host-operator" {
		t.Fatalf("OperatorApproveJoinRequest = %+v %v", approved, err)
	}
	second, _ := w.um.CreateActor(ctx, actor.Actor{DisplayName: "Tóth Gábor"})
	out2, _ := w.um.JoinGroup(ctx, w.group.ID, second.ID, "")
	declined, err := w.um.OperatorDeclineJoinRequest(ctx, out2.Request.ID, "host-operator", true, "")
	if err != nil || declined.State != models.JoinRequestDeclined || !declined.Silent {
		t.Fatalf("OperatorDeclineJoinRequest = %+v %v", declined, err)
	}
}

func TestJoinRequestsListByGroupAndByActor(t *testing.T) {
	w := newJoinWorld(t, "")
	ctx := context.Background()
	second, _ := w.um.CreateActor(ctx, actor.Actor{DisplayName: "Tóth Gábor"})
	a, _ := w.um.JoinGroup(ctx, w.group.ID, w.person.ID, "")
	if _, err := w.um.JoinGroup(ctx, w.group.ID, second.ID, ""); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := w.um.DeclineJoinRequest(ctx, a.Request.ID, w.leader.ID, false, "later"); err != nil {
		t.Fatalf("DeclineJoinRequest: %v", err)
	}
	all, err := w.um.ListJoinRequestsForGroup(ctx, w.group.ID, false)
	if err != nil || len(all) != 2 {
		t.Fatalf("all = %d %v, want 2", len(all), err)
	}
	pending, err := w.um.ListJoinRequestsForGroup(ctx, w.group.ID, true)
	if err != nil || len(pending) != 1 || pending[0].ActorID != second.ID {
		t.Errorf("pending = %+v %v, want only the second actor's", pending, err)
	}
	mine, err := w.um.ListJoinRequestsForActor(ctx, w.person.ID)
	if err != nil || len(mine) != 1 || mine[0].State != models.JoinRequestDeclined {
		t.Errorf("mine = %+v %v", mine, err)
	}
	got, err := w.um.GetJoinRequest(ctx, a.Request.ID)
	if err != nil || got.DeclineReason != "later" {
		t.Errorf("GetJoinRequest = %+v %v", got, err)
	}
}
