package mwanachamaactor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

type nameWrite struct {
	label   string
	refusal error
	write   func(mgr mwanachamaactor.UserManager, name string) error
}

func nameWrites(t *testing.T, mgr mwanachamaactor.UserManager) []nameWrite {
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
	return []nameWrite{
		{"create group", mwanachamaactor.ErrInvalidGroup, func(mgr mwanachamaactor.UserManager, name string) error {
			_, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: name})
			return err
		}},
		{"edit group", mwanachamaactor.ErrInvalidGroup, func(mgr mwanachamaactor.UserManager, name string) error {
			_, err := mgr.EditGroup(ctx, group.ID, mwanachamaactor.GroupEdit{Name: name})
			return err
		}},
		{"create actor", mwanachamaactor.ErrInvalidActor, func(mgr mwanachamaactor.UserManager, name string) error {
			_, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: name})
			return err
		}},
		{"set display name", mwanachamaactor.ErrInvalidActor, func(mgr mwanachamaactor.UserManager, name string) error {
			_, err := mgr.SetActorDisplayName(ctx, actor.ID, name)
			return err
		}},
		{"create hierarchy", mwanachamaactor.ErrInvalidHierarchy, func(mgr mwanachamaactor.UserManager, name string) error {
			_, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: name})
			return err
		}},
		{"rename hierarchy", mwanachamaactor.ErrInvalidHierarchy, func(mgr mwanachamaactor.UserManager, name string) error {
			_, err := mgr.RenameHierarchy(ctx, hierarchy.ID, name)
			return err
		}},
	}
}

func TestNamesOverTwoHundredCharactersAreRefused(t *testing.T) {
	tooLong := map[string]string{
		"201 letters":      strings.Repeat("a", 201),
		"201 accented":     strings.Repeat("é", 201),
		"1000 letters":     strings.Repeat("a", 1000),
		"100000 letters":   strings.Repeat("a", 100000),
		"201 with a space": strings.Repeat("ab ", 67),
	}
	for label, name := range tooLong {
		t.Run(label, func(t *testing.T) {
			mgr := newTestManager(t)
			for _, w := range nameWrites(t, mgr) {
				err := w.write(mgr, name)
				if !errors.Is(err, w.refusal) {
					t.Errorf("%s with %d characters = %.100v, want %v", w.label, len([]rune(name)), err, w.refusal)
				}
			}
		})
	}
}

func TestNamesOfTwoHundredCharactersAreKept(t *testing.T) {
	for label, fill := range map[string]string{"200 letters": "a", "200 accented": "é"} {
		t.Run(label, func(t *testing.T) {
			mgr := newTestManager(t)
			for i, w := range nameWrites(t, mgr) {
				distinct := string(rune('b'+i)) + strings.Repeat(fill, 199)
				if err := w.write(mgr, distinct); err != nil {
					t.Errorf("%s with %d characters = %.100v, want it kept", w.label, len([]rune(distinct)), err)
				}
			}
		})
	}
}

func TestBlankNamesAreRefused(t *testing.T) {
	for label, name := range map[string]string{
		"empty":                "",
		"one space":            " ",
		"five spaces":          "     ",
		"a no-break space":     "\u00a0",
		"an ideographic space": "\u3000",
	} {
		t.Run(label, func(t *testing.T) {
			mgr := newTestManager(t)
			for _, w := range nameWrites(t, mgr) {
				if name == "" && errors.Is(w.refusal, mwanachamaactor.ErrInvalidActor) {
					continue
				}
				if err := w.write(mgr, name); !errors.Is(err, w.refusal) {
					t.Errorf("%s with %q = %v, want %v", w.label, name, err, w.refusal)
				}
			}
		})
	}
}
