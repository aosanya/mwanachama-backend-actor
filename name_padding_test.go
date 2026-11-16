package mwanachamaactor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

type paddedWrite struct {
	label string
	write func(mgr mwanachamaactor.UserManager, name string) (stored string, err error)
}

func paddedWrites(t *testing.T, mgr mwanachamaactor.UserManager) []paddedWrite {
	t.Helper()
	ctx := context.Background()
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	actor, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	hierarchy, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Council"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	return []paddedWrite{
		{"create group", func(mgr mwanachamaactor.UserManager, name string) (string, error) {
			made, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: name})
			if err != nil {
				return "", err
			}
			read, err := mgr.GetGroup(ctx, made.ID)
			return read.Name, err
		}},
		{"edit group", func(mgr mwanachamaactor.UserManager, name string) (string, error) {
			_, err := mgr.EditGroup(ctx, group.ID, mwanachamaactor.GroupEdit{Name: name})
			if err != nil {
				return "", err
			}
			read, err := mgr.GetGroup(ctx, group.ID)
			return read.Name, err
		}},
		{"create actor", func(mgr mwanachamaactor.UserManager, name string) (string, error) {
			made, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: name})
			if err != nil {
				return "", err
			}
			read, err := mgr.GetActor(ctx, made.ID)
			return read.DisplayName, err
		}},
		{"rename actor", func(mgr mwanachamaactor.UserManager, name string) (string, error) {
			if _, err := mgr.SetActorDisplayName(ctx, actor.ID, name); err != nil {
				return "", err
			}
			read, err := mgr.GetActor(ctx, actor.ID)
			return read.DisplayName, err
		}},
		{"create hierarchy", func(mgr mwanachamaactor.UserManager, name string) (string, error) {
			made, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: name})
			if err != nil {
				return "", err
			}
			read, err := mgr.GetHierarchy(ctx, made.ID)
			return read.Name, err
		}},
		{"rename hierarchy", func(mgr mwanachamaactor.UserManager, name string) (string, error) {
			if _, err := mgr.RenameHierarchy(ctx, hierarchy.ID, name); err != nil {
				return "", err
			}
			read, err := mgr.GetHierarchy(ctx, hierarchy.ID)
			return read.Name, err
		}},
	}
}

func TestPaddedNamesAreTrimmedBeforeTheyAreCountedAndStored(t *testing.T) {
	cases := map[string]struct{ typed, stored string }{
		"short name with padding":            {"  Padded  ", "Padded"},
		"200 letters with a space each side": {" " + strings.Repeat("a", 200) + " ", strings.Repeat("a", 200)},
		"200 accented letters padded":        {"\t" + strings.Repeat("é", 200) + " ", strings.Repeat("é", 200)},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			for index := range paddedWrites(t, newTestManager(t)) {
				mgr := newTestManager(t)
				w := paddedWrites(t, mgr)[index]
				stored, err := w.write(mgr, c.typed)
				if err != nil {
					t.Errorf("%s: %v", w.label, err)
					continue
				}
				if stored != c.stored {
					t.Errorf("%s stored %.40q, want %.40q", w.label, stored, c.stored)
				}
			}
		})
	}
}

func TestAPaddedNameOf201CharactersIsStillRefused(t *testing.T) {
	mgr := newTestManager(t)
	typed := " " + strings.Repeat("a", 201) + " "
	for _, w := range paddedWrites(t, mgr) {
		_, err := w.write(mgr, typed)
		if err == nil || !strings.Contains(err.Error(), "at most 200 characters") {
			t.Errorf("%s = %v, want the 200 character refusal", w.label, err)
		}
	}
}

func TestAPaddedDuplicateIsStillRefused(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	if _, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 9"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "  Ward 9  "}); !errors.Is(err, mwanachamaactor.ErrDuplicateGroupName) {
		t.Errorf("group = %v, want ErrDuplicateGroupName", err)
	}
	if _, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Council"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: " Council "}); !errors.Is(err, mwanachamaactor.ErrDuplicateHierarchyName) {
		t.Errorf("hierarchy = %v, want ErrDuplicateHierarchyName", err)
	}
}
