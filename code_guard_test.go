package mwanachamaactor_test

import (
	"context"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestACT78_ACounterBehindTheDataSkipsCodesAlreadyIssued(t *testing.T) {
	mgr, db := newSharedCacheManager(t, "act78")
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{}); err != nil {
			t.Fatalf("CreateActor %d: %v", i, err)
		}
	}
	var table string
	if err := db.Raw("SELECT m.name FROM sqlite_master m WHERE m.type = 'table' AND m.name NOT LIKE '%_test' AND EXISTS (SELECT 1 FROM pragma_table_info(m.name) WHERE name = 'next_number')").Scan(&table).Error; err != nil || table == "" {
		t.Fatalf("find the counter table: %q, %v", table, err)
	}
	if err := db.Exec("UPDATE " + table + " SET next_number = 2 WHERE entity_type = 'actor'").Error; err != nil {
		t.Fatalf("rewind the counter: %v", err)
	}
	got, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{})
	if err != nil {
		t.Fatalf("CreateActor after the counter fell behind: %v", err)
	}
	if got.Code != "AC-4" {
		t.Fatalf("code = %q, want AC-4", got.Code)
	}
}
