package mwanachamaactor

import (
	"context"
	"errors"
	"sync"
	"testing"

	"golang.org/x/text/unicode/norm"
)

const (
	composedWardName   = "Kertváros Újhegy"
	decomposedWardName = "Kertva\u0301ros U\u0301jhegy"
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

func TestGroupNameKeyTreatsComposedAndDecomposedLettersAsOneName(t *testing.T) {
	if composedWardName == decomposedWardName || norm.NFC.String(decomposedWardName) != composedWardName {
		t.Fatalf("the fixtures are not two spellings of one name: %q %q", composedWardName, decomposedWardName)
	}
	if groupNameKey(composedWardName) != groupNameKey(decomposedWardName) {
		t.Fatalf("keys differ: %q and %q", groupNameKey(composedWardName), groupNameKey(decomposedWardName))
	}
	if groupNameKey(composedWardName) != "kertváros újhegy" {
		t.Fatalf("key = %q", groupNameKey(composedWardName))
	}
}

func TestCreatingTheDecomposedTwinOfAGroupNameIsRefusedAndTheNameIsStoredAsTyped(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	if _, err := um.CreateGroup(ctx, Group{Name: composedWardName}); err != nil {
		t.Fatalf("composed: %v", err)
	}
	if _, err := um.CreateGroup(ctx, Group{Name: decomposedWardName}); !errors.Is(err, ErrDuplicateGroupName) {
		t.Fatalf("decomposed twin: err = %v", err)
	}
	other, err := um.CreateGroup(ctx, Group{Name: "Malomsor körzet"})
	if err != nil {
		t.Fatalf("other: %v", err)
	}
	if _, err := um.EditGroup(ctx, other.ID, GroupEdit{Name: decomposedWardName}); !errors.Is(err, ErrDuplicateGroupName) {
		t.Fatalf("rename onto the twin: err = %v", err)
	}
	if _, err := um.EditGroup(ctx, other.ID, GroupEdit{Name: "Malomsor alsó"}); err != nil {
		t.Fatalf("rename to a free name: %v", err)
	}
	renamed, err := um.EditGroup(ctx, other.ID, GroupEdit{Name: "Újhegy alsó"})
	if err != nil || renamed.Name != "Újhegy alsó" {
		t.Fatalf("stored name = %q, %v", renamed.Name, err)
	}
}

func TestADecomposedNameIsStoredAsTypedWhenItIsFree(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	created, err := um.CreateGroup(ctx, Group{Name: decomposedWardName})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Name != decomposedWardName {
		t.Fatalf("stored name = %q, want %q", created.Name, decomposedWardName)
	}
	if _, err := um.CreateGroup(ctx, Group{Name: composedWardName}); !errors.Is(err, ErrDuplicateGroupName) {
		t.Fatalf("composed twin: err = %v", err)
	}
}

func TestProvisionRekeysGroupsKeyedByTheOldRuleAndWithholdsTheIndexWhileTwinsShareAKey(t *testing.T) {
	db, _, _ := openCounterDB(t)
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	table := tableFor(s, roleGroup)
	rows := []map[string]any{
		{"id": "a", "code": "G-a", "name": composedWardName, "name_key": "kertváros újhegy"},
		{"id": "b", "code": "G-b", "name": decomposedWardName, "name_key": "kertva\u0301ros u\u0301jhegy"},
		{"id": "c", "code": "G-c", "name": "Malomsor körzet", "name_key": "malomsor körzet"},
	}
	for _, row := range rows {
		row["deleted"] = false
		if err := db.Table(table).Create(row).Error; err != nil {
			t.Fatalf("insert %v: %v", row["id"], err)
		}
	}
	if held, _ := groupNameIndexExists(db, table); !held {
		t.Fatal("the unique index should exist before the migration")
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision with twins under the old keys: %v", err)
	}
	var keys []string
	if err := db.Table(table).Order("id").Pluck("name_key", &keys).Error; err != nil {
		t.Fatalf("read keys: %v", err)
	}
	if len(keys) != 3 || keys[0] != "kertváros újhegy" || keys[1] != keys[0] || keys[2] != "malomsor körzet" {
		t.Fatalf("keys = %q", keys)
	}
	var names []string
	if err := db.Table(table).Order("id").Pluck("name", &names).Error; err != nil || names[1] != decomposedWardName {
		t.Fatalf("names = %q, %v", names, err)
	}
	if held, _ := groupNameIndexExists(db, table); held {
		t.Fatal("the unique index was kept while two groups share a key")
	}
	if err := db.Table(table).Where("id = ?", "b").Updates(map[string]any{"name": "Újhegy alsó", "name_key": "újhegy alsó"}).Error; err != nil {
		t.Fatalf("rename b: %v", err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision after the rename: %v", err)
	}
	if held, _ := groupNameIndexExists(db, table); !held {
		t.Fatal("the unique index was not created once the keys differ")
	}
}

func TestProvisionRekeysAGroupWhoseOldKeyIsFreeWithoutLosingTheIndex(t *testing.T) {
	db, _, _ := openCounterDB(t)
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	table := tableFor(s, roleGroup)
	row := map[string]any{"id": "b", "code": "G-b", "name": decomposedWardName, "name_key": "kertva\u0301ros u\u0301jhegy", "deleted": false}
	if err := db.Table(table).Create(row).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	var key string
	if err := db.Table(table).Where("id = ?", "b").Pluck("name_key", &key).Error; err != nil || key != "kertváros újhegy" {
		t.Fatalf("key = %q, %v", key, err)
	}
	if held, _ := groupNameIndexExists(db, table); !held {
		t.Fatal("the unique index is missing")
	}
}
