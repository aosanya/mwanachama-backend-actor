package mwanachamaactor_test

import (
	"context"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestACT4_EveryCreateMintsItsOwnServerOwnedFields(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	const forgedID = "attacker-chosen-id"
	const forgedAt = "1999-01-01T00:00:00Z"

	actor, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{ID: forgedID, CreatedAt: forgedAt, LastUpdated: forgedAt, Deleted: true})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	hierarchy, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{ID: forgedID, Name: "Governance", Code: "H-99"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	level, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{ID: forgedID, HierarchyID: hierarchy.ID, Name: "Branch", Code: "L-99"})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	groupType, err := mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{ID: forgedID, HierarchyID: hierarchy.ID, Name: "Chapter", Code: "GT-99"})
	if err != nil {
		t.Fatalf("CreateGroupType: %v", err)
	}
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{ID: forgedID, HierarchyID: hierarchy.ID, LevelID: level.ID, Name: "Ghost", CreatedAt: forgedAt, LastUpdated: forgedAt, Deleted: true, Code: "G-99"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	roleKind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{ID: forgedID, Name: "Chair", Code: "RK-99", RetiredAt: forgedAt, RetiredBy: "someone"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}

	ids := map[string]string{
		"actor": actor.ID, "hierarchy": hierarchy.ID, "level": level.ID,
		"group_type": groupType.ID, "group": group.ID, "role_kind": roleKind.ID,
	}
	for name, id := range ids {
		if id == "" || id == forgedID {
			t.Errorf("ACT4: Create %s answered the id %q, want a server-minted one", name, id)
		}
	}
	codes := map[string]string{
		"hierarchy": hierarchy.Code, "level": level.Code, "group_type": groupType.Code,
		"group": group.Code, "role_kind": roleKind.Code,
	}
	for name, code := range codes {
		if code == "" || code[len(code)-2:] == "99" {
			t.Errorf("ACT4: Create %s kept a caller-set code %q", name, code)
		}
	}
	if actor.CreatedAt == forgedAt || actor.LastUpdated == forgedAt || actor.Deleted {
		t.Errorf("ACT4: CreateActor kept a caller-set audit field: %+v", actor)
	}
	if group.CreatedAt == forgedAt || group.LastUpdated == forgedAt || group.Deleted {
		t.Errorf("ACT4: CreateGroup kept a caller-set audit field: %+v", group)
	}
	if roleKind.RetiredAt != "" || roleKind.RetiredBy != "" {
		t.Errorf("ACT4: CreateRoleKind kept a caller-set retirement: %+v", roleKind)
	}

	if _, err := mgr.GetActor(ctx, forgedID); err == nil {
		t.Error("ACT4: the forged actor id addresses a row")
	}
	if _, err := mgr.GetActor(ctx, actor.ID); err != nil {
		t.Errorf("ACT4: the minted actor cannot be read back: %v", err)
	}
	if _, err := mgr.GetGroup(ctx, forgedID); err == nil {
		t.Error("ACT4: the forged group id addresses a row")
	}
}
