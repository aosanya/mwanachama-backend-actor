package mwanachamaactor

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func newNamedManager(t *testing.T) (UserManager, string) {
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
	return um, tableFor(s, roleGroup)
}

func TestGroupNameKeyIgnoresCaseEdgeSpacesAndSpaceRuns(t *testing.T) {
	for _, name := range []string{"Kertváros körzet", "kertváros körzet", " Kertváros körzet ", "Kertváros  \t körzet"} {
		if got := groupNameKey(name); got != "kertváros körzet" {
			t.Errorf("groupNameKey(%q) = %q", name, got)
		}
	}
}

func TestRenamingAGroupToAnotherGroupsNameIsRefusedAndToItsOwnIsAdmitted(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	first, err := um.CreateGroup(ctx, Group{Name: "Kertváros körzet"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := um.CreateGroup(ctx, Group{Name: "Malomsor körzet"})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if _, err := um.EditGroup(ctx, second.ID, GroupEdit{Name: "KERTVÁROS  körzet"}); !errors.Is(err, ErrDuplicateGroupName) {
		t.Fatalf("rename onto a taken name: err = %v", err)
	}
	if _, err := um.EditGroup(ctx, first.ID, GroupEdit{Name: "kertváros körzet"}); err != nil {
		t.Fatalf("rename to its own name in another case: %v", err)
	}
	if _, err := um.EditGroup(ctx, second.ID, GroupEdit{Name: "Malomsor alsó"}); err != nil {
		t.Fatalf("rename to a free name: %v", err)
	}
	if _, err := um.CreateGroup(ctx, Group{Name: "Malomsor körzet"}); err != nil {
		t.Fatalf("the name freed by the rename: %v", err)
	}
}

func TestThirtyGroupsCreatedAtOnceWithOneNameGiveOneGroup(t *testing.T) {
	um, _ := newNamedManager(t)
	const senders = 30
	var wg sync.WaitGroup
	errs := make([]error, senders)
	for i := 0; i < senders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = um.CreateGroup(context.Background(), Group{Name: "Kertváros körzet"})
		}(i)
	}
	wg.Wait()
	created, refused := 0, 0
	for i, err := range errs {
		switch {
		case err == nil:
			created++
		case errors.Is(err, ErrDuplicateGroupName):
			refused++
		default:
			t.Fatalf("sender %d: %v", i, err)
		}
	}
	if created != 1 || refused != senders-1 {
		t.Fatalf("created %d, refused %d, want 1 and %d", created, refused, senders-1)
	}
}

func TestProvisionKeysOldGroupsAndWithholdsTheIndexWhileTwoShareAName(t *testing.T) {
	db, _, _ := openCounterDB(t)
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	table := tableFor(s, roleGroup)
	if err := db.Exec("drop index " + groupNameIndexName(table)).Error; err != nil {
		t.Fatalf("drop index: %v", err)
	}
	for _, id := range []string{"a", "b"} {
		row := map[string]any{"id": id, "code": "G-" + id, "name": "Kertváros  körzet", "deleted": false, "name_key": ""}
		if err := db.Table(table).Create(row).Error; err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision with a shared name: %v", err)
	}
	var keys []string
	if err := db.Table(table).Order("id").Pluck("name_key", &keys).Error; err != nil || len(keys) != 2 || keys[0] != "kertváros körzet" || keys[1] != keys[0] {
		t.Fatalf("keys = %v, %v", keys, err)
	}
	if held, _ := groupNameIndexExists(db, table); held {
		t.Fatal("the unique index was created while two groups share a name")
	}
	if err := db.Table(table).Where("id = ?", "b").Updates(map[string]any{"name": "Malomsor körzet", "name_key": "malomsor körzet"}).Error; err != nil {
		t.Fatalf("rename b: %v", err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision after the rename: %v", err)
	}
	if held, _ := groupNameIndexExists(db, table); !held {
		t.Fatal("the unique index was not created once the names differ")
	}
}
