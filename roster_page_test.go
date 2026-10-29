package mwanachamaactor_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func rosterOf(t *testing.T, mgr mwanachamaactor.UserManager, size int) (mwanachamaactor.Group, []string) {
	t.Helper()
	ctx := context.Background()
	g := mustGroup(t, mgr, "Ward A")
	ids := make([]string, 0, size)
	for i := 0; i < size; i++ {
		a := mustActor(t, mgr, fmt.Sprintf("Member %03d", i))
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g.ID}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
		ids = append(ids, a.ID)
	}
	sort.Strings(ids)
	return g, ids
}

func actorIDs(rows []mwanachamaactor.ActorGroupAssignment) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.ActorID
	}
	return out
}

func TestTheRosterComesFiftyAtATimeByTheActorKey(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, ids := rosterOf(t, mgr, 120)

	first, err := mgr.ListGroupRoster(ctx, g.ID, "", 0, "")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	if got := actorIDs(first); strings.Join(got, ",") != strings.Join(ids[:50], ",") {
		t.Fatalf("first page = %d rows, want the first 50 by key", len(got))
	}
	second, err := mgr.ListGroupRoster(ctx, g.ID, first[49].ActorID, 0, "")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	if got := actorIDs(second); strings.Join(got, ",") != strings.Join(ids[50:100], ",") {
		t.Fatalf("second page = %d rows, want keys 50 to 99", len(got))
	}
	third, err := mgr.ListGroupRoster(ctx, g.ID, second[49].ActorID, 0, "")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	if got := actorIDs(third); strings.Join(got, ",") != strings.Join(ids[100:], ",") {
		t.Fatalf("third page = %d rows, want the last 20", len(got))
	}
	none, err := mgr.ListGroupRoster(ctx, g.ID, third[19].ActorID, 0, "")
	if err != nil || len(none) != 0 {
		t.Fatalf("page past the end = %d rows, %v, want none", len(none), err)
	}
}

func TestARosterPageIsCappedWhateverLimitIsAsked(t *testing.T) {
	mgr := newTestManager(t)
	g, _ := rosterOf(t, mgr, mwanachamaactor.MaxRosterPageSize+5)
	page, err := mgr.ListGroupRoster(context.Background(), g.ID, "", 100000, "")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	if len(page) != mwanachamaactor.MaxRosterPageSize {
		t.Fatalf("page = %d rows, want the cap of %d", len(page), mwanachamaactor.MaxRosterPageSize)
	}
	small, err := mgr.ListGroupRoster(context.Background(), g.ID, "", 7, "")
	if err != nil || len(small) != 7 {
		t.Fatalf("limit 7 = %d rows, %v, want 7", len(small), err)
	}
}

func TestARosterPageStaysFullWhenOutsidersSitInsideIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, ids := rosterOf(t, mgr, 12)
	for _, id := range []string{ids[1], ids[2], ids[5]} {
		seatOutsider(t, mgr, g.ID, id)
	}
	want := []string{ids[0], ids[3], ids[4], ids[6], ids[7]}
	page, err := mgr.ListGroupRoster(ctx, g.ID, "", 5, "")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	if strings.Join(actorIDs(page), ",") != strings.Join(want, ",") {
		t.Fatalf("page = %v, want %v", actorIDs(page), want)
	}
	next, err := mgr.ListGroupRoster(ctx, g.ID, page[4].ActorID, 5, "")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	wantNext := []string{ids[8], ids[9], ids[10], ids[11]}
	if strings.Join(actorIDs(next), ",") != strings.Join(wantNext, ",") {
		t.Fatalf("next page = %v, want %v", actorIDs(next), wantNext)
	}
}

func TestARosterPageOfAnUnknownGroupIsEmpty(t *testing.T) {
	mgr := newTestManager(t)
	page, err := mgr.ListGroupRoster(context.Background(), "no-such-group", "", 0, "")
	if err != nil || len(page) != 0 {
		t.Fatalf("page = %v, %v, want none", page, err)
	}
}

type statementLog struct {
	mu   sync.Mutex
	sqls []string
}

func (l *statementLog) LogMode(logger.LogLevel) logger.Interface      { return l }
func (l *statementLog) Info(context.Context, string, ...interface{})  {}
func (l *statementLog) Warn(context.Context, string, ...interface{})  {}
func (l *statementLog) Error(context.Context, string, ...interface{}) {}
func (l *statementLog) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	l.mu.Lock()
	l.sqls = append(l.sqls, sql)
	l.mu.Unlock()
}

func (l *statementLog) seatReads() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, s := range l.sqls {
		if strings.Contains(s, "outsider = true") {
			out = append(out, s)
		}
	}
	return out
}

func loggedManager(t *testing.T) (mwanachamaactor.UserManager, *statementLog) {
	t.Helper()
	log := &statementLog{}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: log})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return newTestManagerOn(t, db), log
}

func TestTheOutsiderCheckOfARosterPageOnlyLooksAtThePagesActors(t *testing.T) {
	mgr, log := loggedManager(t)
	ctx := context.Background()
	g, ids := rosterOf(t, mgr, 8)
	seatOutsider(t, mgr, g.ID, ids[3])
	log.mu.Lock()
	log.sqls = nil
	log.mu.Unlock()

	if _, err := mgr.ListGroupRoster(ctx, g.ID, "", 5, ""); err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	reads := log.seatReads()
	if len(reads) == 0 {
		t.Fatal("the page never looked at the seats")
	}
	for _, s := range reads {
		if !strings.Contains(s, "actor_id IN") {
			t.Fatalf("a seat read ran over every actor: %s", s)
		}
	}
}

func TestTheWholeRosterOutsiderCheckOnlyLooksAtTheGroupsActors(t *testing.T) {
	mgr, log := loggedManager(t)
	ctx := context.Background()
	g, ids := rosterOf(t, mgr, 6)
	seatOutsider(t, mgr, g.ID, ids[2])
	log.mu.Lock()
	log.sqls = nil
	log.mu.Unlock()

	members, err := mgr.ListActorsForGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("ListActorsForGroup: %v", err)
	}
	if len(members) != 5 {
		t.Fatalf("members = %d, want 5", len(members))
	}
	reads := log.seatReads()
	if len(reads) == 0 {
		t.Fatal("the roster never looked at the seats")
	}
	for _, s := range reads {
		if !strings.Contains(s, "actor_id IN") {
			t.Fatalf("a seat read ran over every actor: %s", s)
		}
	}
}
