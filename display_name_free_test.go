package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestADisplayNameThatOnlyLooksLikeAnotherMembersIsRefused(t *testing.T) {
	lookalikes := map[string]string{
		"another case":            "AMINA",
		"full-width letters":      "Ａｍｉｎａ",
		"extra spaces":            "  Amina   ",
		"a trailing dot":          "Amina.",
		"a Cyrillic letter":       "Аmina",
		"a Greek letter":          "Αmina",
		"a Cyrillic lower letter": "Aminа",
	}
	for label, typed := range lookalikes {
		t.Run(label, func(t *testing.T) {
			mgr := newTestManager(t)
			ctx := context.Background()
			if _, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"}); err != nil {
				t.Fatalf("first: %v", err)
			}
			other, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Beatrix"})
			if err != nil {
				t.Fatalf("second: %v", err)
			}
			if _, err := mgr.SetActorDisplayName(ctx, other.ID, typed); !errors.Is(err, mwanachamaactor.ErrDuplicateDisplayName) {
				t.Fatalf("SetActorDisplayName(%q): err = %v", typed, err)
			}
			kept, err := mgr.GetActor(ctx, other.ID)
			if err != nil || kept.DisplayName != "Beatrix" {
				t.Fatalf("display name = %q, %v; want it unchanged", kept.DisplayName, err)
			}
		})
	}
}

func TestADisplayNameThatLooksLikeALiveRoleKindOrOneOfItsPartsIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Chair/Secretary"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	actor, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	for _, typed := range []string{"chair/secretary", "SECRETARY", "Chаir"} {
		if _, err := mgr.SetActorDisplayName(ctx, actor.ID, typed); !errors.Is(err, mwanachamaactor.ErrDuplicateDisplayName) {
			t.Fatalf("SetActorDisplayName(%q): err = %v", typed, err)
		}
	}
	if _, err := mgr.RetireRoleKind(ctx, kind.ID, "actor-1"); err != nil {
		t.Fatalf("RetireRoleKind: %v", err)
	}
	if _, err := mgr.SetActorDisplayName(ctx, actor.ID, "Secretary"); err != nil {
		t.Fatalf("a retired kind's name: %v", err)
	}
}

func TestAccentedHungarianLettersKeepDisplayNamesApartAndAMemberMayKeepOwnName(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	first, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Kovacs Aron"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Beatrix"})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if _, err := mgr.SetActorDisplayName(ctx, second.ID, "Kovács Áron"); err != nil {
		t.Fatalf("accented name: %v", err)
	}
	if _, err := mgr.SetActorDisplayName(ctx, first.ID, "KOVACS ARON"); err != nil {
		t.Fatalf("own name in another case: %v", err)
	}
}
