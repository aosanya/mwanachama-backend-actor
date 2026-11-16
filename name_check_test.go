package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

var namesThatDrawNothing = map[string]string{
	"a zero-width space":     "Ada​Lovelace",
	"a control character":    "Ada\x07Lovelace",
	"a line break":           "Ada\nLovelace",
	"only a combining mark":  "́",
	"a right-to-left marker": "Ada\u200FLovelace",
}

func TestNamesThatDrawNothingAreRefusedWhereTheyAreTyped(t *testing.T) {
	for label, bad := range namesThatDrawNothing {
		t.Run(label, func(t *testing.T) {
			mgr := newTestManager(t)
			ctx := context.Background()
			h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Council"})
			if err != nil {
				t.Fatalf("CreateHierarchy: %v", err)
			}
			level, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Ward", Depth: 1})
			if err != nil {
				t.Fatalf("CreateLevel: %v", err)
			}
			groupType, err := mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: h.ID, Name: "Branch"})
			if err != nil {
				t.Fatalf("CreateGroupType: %v", err)
			}
			actor, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
			if err != nil {
				t.Fatalf("CreateActor: %v", err)
			}
			group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
			if err != nil {
				t.Fatalf("CreateGroup: %v", err)
			}

			refused := map[string]struct {
				err  error
				want error
			}{}
			add := func(name string, err, want error) { refused[name] = struct{ err, want error }{err, want} }

			_, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: bad})
			add("create actor display name", err, mwanachamaactor.ErrInvalidActor)
			_, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka", FirstName: bad})
			add("create actor first name", err, mwanachamaactor.ErrInvalidActor)
			_, err = mgr.SetActorDisplayName(ctx, actor.ID, bad)
			add("set display name", err, mwanachamaactor.ErrInvalidActor)
			_, err = mgr.SetActorName(ctx, actor.ID, "Amina", bad)
			add("set last name", err, mwanachamaactor.ErrInvalidActor)
			_, err = mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: bad})
			add("create hierarchy", err, mwanachamaactor.ErrInvalidHierarchy)
			_, err = mgr.RenameHierarchy(ctx, h.ID, bad)
			add("rename hierarchy", err, mwanachamaactor.ErrInvalidHierarchy)
			_, err = mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: bad, Depth: 2})
			add("create level", err, mwanachamaactor.ErrInvalidLevel)
			_, err = mgr.RenameLevel(ctx, level.ID, bad)
			add("rename level", err, mwanachamaactor.ErrInvalidLevel)
			_, err = mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: h.ID, Name: bad})
			add("create group type", err, mwanachamaactor.ErrInvalidGroupType)
			_, err = mgr.EditGroupType(ctx, groupType.ID, "Branch", bad, "Branches", "")
			add("edit group type singular", err, mwanachamaactor.ErrInvalidGroupType)
			_, err = mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: bad})
			add("create role kind", err, mwanachamaactor.ErrInvalidRoleKind)
			_, err = mgr.SetGroupNameTranslation(ctx, group.ID, "sw", bad)
			add("translate group name", err, mwanachamaactor.ErrInvalidTranslation)

			for name, got := range refused {
				if !errors.Is(got.err, got.want) {
					t.Errorf("%s with %q = %v, want %v", name, bad, got.err, got.want)
				}
			}
		})
	}
}
