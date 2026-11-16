package mwanachamaactor

import (
	"context"
	"errors"
	"testing"
)

func TestGroupNameKeyFoldsWidthAndTrailingDots(t *testing.T) {
	for _, name := range []string{
		"Ｋertváros körzet", "Kertváros ｋöｒｚｅｔ", "Kertváros körzet.", "Kertváros körzet...",
		"Kertváros körzet．", "Kertváros körzet. ", "Kertváros　körzet", "Kertváros körzet ",
	} {
		if got := groupNameKey(name); got != "kertváros körzet" {
			t.Errorf("groupNameKey(%q) = %q", name, got)
		}
	}
}

func TestGroupNameKeyKeepsNamesThatDifferInsideOrAtTheStart(t *testing.T) {
	keys := map[string]string{}
	for _, name := range []string{"Kertváros körzet", "Kertváros körzet 2", "Kertváros. körzet", ".Kertváros körzet", "Kertváros-körzet"} {
		key := groupNameKey(name)
		if other, taken := keys[key]; taken {
			t.Errorf("%q and %q share the key %q", name, other, key)
		}
		keys[key] = name
	}
}

func TestGroupNameKeyOfOnlyDotsStaysNonEmptyAndApartFromOtherNames(t *testing.T) {
	for _, name := range []string{".", "...", ". ."} {
		if groupNameKey(name) == "" {
			t.Errorf("groupNameKey(%q) is empty, so the unique index would skip it", name)
		}
	}
}

func TestCreatingAFullWidthOrTrailingDotTwinOfAGroupNameIsRefused(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	if _, err := um.CreateGroup(ctx, Group{Name: "Kertváros körzet"}); err != nil {
		t.Fatalf("first: %v", err)
	}
	other, err := um.CreateGroup(ctx, Group{Name: "Malomsor körzet"})
	if err != nil {
		t.Fatalf("other: %v", err)
	}
	for _, twin := range []string{"Ｋertváros körzet", "Kertváros körzet.", "Kertváros körzet..."} {
		if _, err := um.CreateGroup(ctx, Group{Name: twin}); !errors.Is(err, ErrDuplicateGroupName) {
			t.Errorf("create %q: err = %v", twin, err)
		}
		if _, err := um.EditGroup(ctx, other.ID, GroupEdit{Name: twin}); !errors.Is(err, ErrDuplicateGroupName) {
			t.Errorf("rename to %q: err = %v", twin, err)
		}
	}
}

func TestProvisionRekeysGroupsKeyedBeforeWidthAndTrailingDotsFoldedAndWithholdsTheIndexWhileTwinsShareAKey(t *testing.T) {
	db, _, _ := openCounterDB(t)
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	table := tableFor(s, roleGroup)
	if err := db.Exec("drop index " + groupNameIndexName(table)).Error; err != nil {
		t.Fatalf("drop index: %v", err)
	}
	rows := []map[string]any{
		{"id": "a", "code": "G-a", "name": "Kertváros körzet", "name_key": "kertváros körzet"},
		{"id": "b", "code": "G-b", "name": "Kertváros körzet.", "name_key": "kertváros körzet."},
		{"id": "c", "code": "G-c", "name": "Malomsor körzet", "name_key": "malomsor körzet"},
	}
	for _, row := range rows {
		row["deleted"] = false
		if err := db.Table(table).Create(row).Error; err != nil {
			t.Fatalf("insert %v: %v", row["id"], err)
		}
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	var keys []string
	if err := db.Table(table).Order("id").Pluck("name_key", &keys).Error; err != nil || len(keys) != 3 || keys[0] != "kertváros körzet" || keys[1] != keys[0] || keys[2] != "malomsor körzet" {
		t.Fatalf("keys = %v, %v", keys, err)
	}
	if held, _ := groupNameIndexExists(db, table); held {
		t.Fatal("the unique index was created while two groups share a key")
	}
}
