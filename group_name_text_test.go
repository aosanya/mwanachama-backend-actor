package mwanachamaactor

import (
	"context"
	"errors"
	"testing"
)

func newTextCheckedManager(t *testing.T) UserManager {
	t.Helper()
	db, _, _ := openCounterDB(t)
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	um, err := NewUserManager(db, s)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return um
}

func TestAGroupNameWithAControlCharacterIsRefusedOnCreateAndRename(t *testing.T) {
	um := newTextCheckedManager(t)
	ctx := context.Background()
	kept, err := um.CreateGroup(ctx, Group{Name: "Kertváros körzet"})
	if err != nil {
		t.Fatalf("plain name: %v", err)
	}
	for _, name := range []string{"Kert\u0000város", "Kert\u0007város", "Kert\nváros", "Kert\u202Eváros"} {
		if _, err := um.CreateGroup(ctx, Group{Name: name}); !errors.Is(err, ErrInvalidGroup) {
			t.Errorf("create %q: err = %v", name, err)
		}
		if _, err := um.EditGroup(ctx, kept.ID, GroupEdit{Name: name}); !errors.Is(err, ErrInvalidGroup) {
			t.Errorf("rename to %q: err = %v", name, err)
		}
	}
	got, err := um.GetGroup(ctx, kept.ID)
	if err != nil || got.Name != "Kertváros körzet" {
		t.Fatalf("after refused renames: %+v, %v", got, err)
	}
}
