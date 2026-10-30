package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

type runRig struct {
	live mwanachamaactor.UserManager
	db   *gorm.DB
	spec *spec.Spec
}

func newRunRig(t *testing.T) runRig {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatal(err)
	}
	if err := mwanachamaactor.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	live, err := mwanachamaactor.NewUserManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	return runRig{live: live, db: db, spec: s}
}

func (r runRig) run(t *testing.T, id string) mwanachamaactor.UserManager {
	t.Helper()
	m, err := r.live.ForRun(id)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func (r runRig) rowsIn(t *testing.T, table func(spec.Object) string) int64 {
	t.Helper()
	var total int64
	for _, o := range r.spec.Objects {
		var n int64
		if err := r.db.Table(table(o)).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		total += n
	}
	return total
}

func (r runRig) liveRows(t *testing.T) int64 { return r.rowsIn(t, r.spec.TableFor) }

func (r runRig) testCopyRows(t *testing.T) int64 { return r.rowsIn(t, r.spec.TestTableFor) }

type workout struct {
	hierarchy mwanachamaactor.Hierarchy
	level     mwanachamaactor.Level
	groupType mwanachamaactor.GroupType
	kind      mwanachamaactor.RoleKind
	group     mwanachamaactor.Group
	actor     mwanachamaactor.Actor
}

func work(t *testing.T, m mwanachamaactor.UserManager, label string) workout {
	t.Helper()
	ctx := context.Background()
	var w workout
	var err error
	if w.hierarchy, err = m.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: label + " council"}); err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	if w.level, err = m.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: w.hierarchy.ID, Name: label + " ward"}); err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	if w.groupType, err = m.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: w.hierarchy.ID, LevelID: w.level.ID, Name: label + " type"}); err != nil {
		t.Fatalf("CreateGroupType: %v", err)
	}
	if w.kind, err = m.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: label + " chair", LeadsGroup: true}); err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if w.group, err = m.CreateGroup(ctx, mwanachamaactor.Group{Name: label + " group", HierarchyID: w.hierarchy.ID, LevelID: w.level.ID}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if w.actor, err = m.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: label + " member"}); err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if _, err = m.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: w.actor.ID, GroupID: w.group.ID, KindID: w.kind.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	joiner, err := m.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: label + " joiner"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if _, err = m.JoinGroup(ctx, w.group.ID, joiner.ID, "hello"); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err = m.InviteToGroup(ctx, w.group.ID, w.actor.ID, "", w.actor.ID, ""); err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	if _, err = m.SetCodeFormat(ctx, 4); err != nil {
		t.Fatalf("SetCodeFormat: %v", err)
	}
	return w
}

func TestARunThroughTheDirectoryLeavesNoRowInAnyLiveTable(t *testing.T) {
	rig := newRunRig(t)
	work(t, rig.run(t, "run-1"), "one")

	if got := rig.liveRows(t); got != 0 {
		t.Fatalf("live rows = %d after a run, want 0", got)
	}
	rows, err := specstore.RowsOfRun(context.Background(), rig.db, rig.spec, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	stored := map[string]int64{}
	for _, r := range rows {
		stored[r.Object] = r.Rows
	}
	for _, object := range []string{"actor", "group", "hierarchy", "level", "group_type", "role_kind", "actor_role_assignment", "group_join_request", "group_invitation", "code_sequence", "code_format"} {
		if stored[object] == 0 {
			t.Fatalf("run-1 stored nothing in %s: %+v", object, rows)
		}
	}
}

func TestALiveWriteLeavesTheTestCopyEmpty(t *testing.T) {
	rig := newRunRig(t)
	work(t, rig.live, "live")

	if got := rig.testCopyRows(t); got != 0 {
		t.Fatalf("test copy holds %d rows after a live write, want 0", got)
	}
}

func TestARunDoesNotSeeLiveOrOtherRunRows(t *testing.T) {
	rig := newRunRig(t)
	ctx := context.Background()
	first, second := rig.run(t, "run-1"), rig.run(t, "run-2")
	liveWork := work(t, rig.live, "live")
	firstWork := work(t, first, "one")
	work(t, second, "two")

	if _, err := first.GetGroup(ctx, liveWork.group.ID); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Fatalf("run-1 reading a live group = %v, want ErrGroupNotFound", err)
	}
	if _, err := rig.live.GetGroup(ctx, firstWork.group.ID); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Fatalf("live reading run-1's group = %v, want ErrGroupNotFound", err)
	}
	if _, err := second.GetGroup(ctx, firstWork.group.ID); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Fatalf("run-2 reading run-1's group = %v, want ErrGroupNotFound", err)
	}
	if _, err := second.GetHierarchy(ctx, firstWork.hierarchy.ID); !errors.Is(err, mwanachamaactor.ErrHierarchyNotFound) {
		t.Fatalf("run-2 reading run-1's hierarchy = %v, want ErrHierarchyNotFound", err)
	}
	groups, err := first.ListGroups(ctx, "")
	if err != nil || len(groups) != 1 || groups[0].ID != firstWork.group.ID {
		t.Fatalf("run-1 groups = %+v, %v; want only its own", groups, err)
	}
	led, err := first.IsGroupLeader(ctx, liveWork.group.ID, liveWork.actor.ID)
	if err != nil || led {
		t.Fatalf("run-1 saw a live leader: %v, %v", led, err)
	}
	if led, err := first.IsGroupLeader(ctx, firstWork.group.ID, firstWork.actor.ID); err != nil || !led {
		t.Fatalf("run-1 lost its own leader: %v, %v", led, err)
	}
}

func TestEachRunMintsItsOwnCodesFromOne(t *testing.T) {
	rig := newRunRig(t)
	ctx := context.Background()
	live := work(t, rig.live, "live")
	first := work(t, rig.run(t, "run-1"), "one")
	second := work(t, rig.run(t, "run-2"), "two")

	if live.group.Code != first.group.Code || first.group.Code != second.group.Code {
		t.Fatalf("group codes = %q, %q, %q; each sequence should start alike", live.group.Code, first.group.Code, second.group.Code)
	}
	again, err := rig.run(t, "run-1").CreateGroup(ctx, mwanachamaactor.Group{Name: "one more"})
	if err != nil {
		t.Fatal(err)
	}
	if again.Code == first.group.Code {
		t.Fatalf("run-1 minted %q twice", again.Code)
	}
	liveAgain, err := rig.live.CreateGroup(ctx, mwanachamaactor.Group{Name: "live more"})
	if err != nil {
		t.Fatal(err)
	}
	if liveAgain.Code != again.Code {
		t.Fatalf("live second group code = %q, run-1 second = %q; sequences moved together", liveAgain.Code, again.Code)
	}
}

func TestAChangeInARunLeavesAnotherRunsRowsAlone(t *testing.T) {
	rig := newRunRig(t)
	ctx := context.Background()
	first, second := rig.run(t, "run-1"), rig.run(t, "run-2")
	firstWork := work(t, first, "one")
	secondWork := work(t, second, "two")

	if err := first.DeleteLevel(ctx, secondWork.level.ID); !errors.Is(err, mwanachamaactor.ErrLevelNotFound) {
		t.Fatalf("run-1 deleting run-2's level = %v, want ErrLevelNotFound", err)
	}
	if _, err := first.RetireRoleKind(ctx, secondWork.kind.ID, "x"); !errors.Is(err, mwanachamaactor.ErrRoleKindNotFound) {
		t.Fatalf("run-1 retiring run-2's kind = %v, want ErrRoleKindNotFound", err)
	}
	if _, err := first.RenameHierarchy(ctx, secondWork.hierarchy.ID, "hijacked"); !errors.Is(err, mwanachamaactor.ErrHierarchyNotFound) {
		t.Fatalf("run-1 renaming run-2's hierarchy = %v, want ErrHierarchyNotFound", err)
	}
	if err := first.DeleteLevel(ctx, firstWork.level.ID); !errors.Is(err, mwanachamaactor.ErrLevelWornByGroups) {
		t.Fatalf("run-1 deleting its own level = %v, want ErrLevelWornByGroups", err)
	}

	if got, err := second.GetLevel(ctx, secondWork.level.ID); err != nil || got.Name != "two ward" {
		t.Fatalf("run-2 level = %+v, %v", got, err)
	}
	if got, err := second.GetHierarchy(ctx, secondWork.hierarchy.ID); err != nil || got.Name != "two council" {
		t.Fatalf("run-2 hierarchy = %+v, %v", got, err)
	}
	if got, err := second.GetRoleKind(ctx, secondWork.kind.ID); err != nil || got.RetiredAt != "" {
		t.Fatalf("run-2 role kind = %+v, %v", got, err)
	}
}

func TestRemovingARunRemovesItsRowsAndKeepsTheOthers(t *testing.T) {
	rig := newRunRig(t)
	ctx := context.Background()
	work(t, rig.run(t, "run-1"), "one")
	secondWork := work(t, rig.run(t, "run-2"), "two")

	removed, err := specstore.RemoveRun(ctx, rig.db, rig.spec, "run-1")
	if err != nil || removed == 0 {
		t.Fatalf("removed %d rows, %v", removed, err)
	}
	left, err := specstore.RowsOfRun(ctx, rig.db, rig.spec, "run-1")
	if err != nil || len(left) != 0 {
		t.Fatalf("run-1 still holds %+v, %v", left, err)
	}
	if got, err := rig.run(t, "run-2").GetGroup(ctx, secondWork.group.ID); err != nil || got.Name != "two group" {
		t.Fatalf("run-2 group after run-1 was removed = %+v, %v", got, err)
	}
	if got := rig.liveRows(t); got != 0 {
		t.Fatalf("live rows = %d, want 0", got)
	}
}

func TestARunWithoutAnIdIsRefused(t *testing.T) {
	rig := newRunRig(t)
	if _, err := rig.live.ForRun(""); !errors.Is(err, specstore.ErrRunIDEmpty) {
		t.Fatalf("ForRun(\"\") = %v, want ErrRunIDEmpty", err)
	}
}

func TestARunKeepsTheLiveManagersOwnSettings(t *testing.T) {
	rig := newRunRig(t)
	ctx := context.Background()
	work(t, rig.live, "live")
	run := rig.run(t, "run-1")
	if _, err := run.CreateGroup(ctx, mwanachamaactor.Group{Name: "in run", JoinMode: models.JoinConfirmed}); err != nil {
		t.Fatal(err)
	}
	groups, err := rig.live.ListGroups(ctx, "")
	if err != nil || len(groups) != 1 {
		t.Fatalf("live groups = %+v, %v; the live manager changed scope", groups, err)
	}
}

func TestAnotherRunsGroupDoesNotKeepALevelFromBeingDeleted(t *testing.T) {
	rig := newRunRig(t)
	ctx := context.Background()
	first, second := rig.run(t, "run-1"), rig.run(t, "run-2")
	h, err := first.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "council"})
	if err != nil {
		t.Fatal(err)
	}
	l, err := first.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "ward"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.CreateGroup(ctx, mwanachamaactor.Group{Name: "wears it", LevelID: l.ID}); err != nil {
		t.Fatal(err)
	}

	if err := first.DeleteLevel(ctx, l.ID); err != nil {
		t.Fatalf("run-1 deleting its idle level = %v, want nil", err)
	}
}
