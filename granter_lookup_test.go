package mwanachamaactor_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func (l *statementLog) forgetStatements() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sqls = nil
}

func (l *statementLog) statementsTaken() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.sqls...)
}

type countedManager struct {
	mgr   mwanachamaactor.UserManager
	db    *gorm.DB
	log   *statementLog
	table string
}

func newCountedManager(t *testing.T) countedManager {
	t.Helper()
	log := &statementLog{}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: log})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	o, _ := s.ByRole("actor")
	return countedManager{mgr: mgr, db: db, log: log, table: s.TableFor(o)}
}

func (c countedManager) softDeleteActor(t *testing.T, id string) {
	t.Helper()
	if err := c.db.Exec("update "+c.table+" set deleted = ? where id = ?", true, id).Error; err != nil {
		t.Fatalf("soft delete actor %s: %v", id, err)
	}
}

func (c countedManager) personSeatedBy(t *testing.T, name string, granters int) (mwanachamaactor.Actor, []mwanachamaactor.Actor) {
	t.Helper()
	ctx := context.Background()
	person := mustActor(t, c.mgr, name)
	kind, err := c.mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: name + " Coordinator"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	var named []mwanachamaactor.Actor
	for i := 1; i <= granters; i++ {
		granter := mustActor(t, c.mgr, fmt.Sprintf("%s Granter %d", name, i))
		ward := mustGroup(t, c.mgr, fmt.Sprintf("%s Ward %d", name, i))
		_, err := c.mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{
			ActorID: person.ID, GroupID: ward.ID, KindID: kind.ID, GrantedBy: granter.ID,
		})
		if err != nil {
			t.Fatalf("GrantRole from granter %d: %v", i, err)
		}
		named = append(named, granter)
	}
	return person, named
}

func (c countedManager) statementsOfStanding(t *testing.T, actorID string) (int, []string) {
	t.Helper()
	c.log.forgetStatements()
	if _, err := c.mgr.GetActorStanding(context.Background(), actorID); err != nil {
		t.Fatalf("GetActorStanding: %v", err)
	}
	statements := c.log.statementsTaken()
	return len(statements), statements
}

func TestStandingCostsTheSameNumberOfStatementsForSixGrantersAsForOne(t *testing.T) {
	c := newCountedManager(t)
	one, _ := c.personSeatedBy(t, "Solo", 1)
	six, granters := c.personSeatedBy(t, "Busy", 6)

	forOne, _ := c.statementsOfStanding(t, one.ID)
	forSix, statements := c.statementsOfStanding(t, six.ID)

	t.Logf("statements for one seat from one granter: %d; for six seats from six granters: %d", forOne, forSix)
	if forSix != forOne {
		t.Errorf("a standing with six granters took %d statements, one granter took %d; the granters must be read in one statement:\n%s",
			forSix, forOne, strings.Join(statements, "\n"))
	}

	c.log.forgetStatements()
	standing, err := c.mgr.GetActorStanding(context.Background(), six.ID)
	if err != nil {
		t.Fatalf("GetActorStanding: %v", err)
	}
	if len(standing.Seats) != len(granters) {
		t.Fatalf("seats = %d, want %d", len(standing.Seats), len(granters))
	}
	byGranter := map[string]string{}
	for _, seat := range standing.Seats {
		byGranter[seat.GrantedBy] = seat.GrantedByName
	}
	for _, g := range granters {
		if byGranter[g.ID] != g.DisplayName {
			t.Errorf("seat granted by %s names the granter %q, want %q", g.ID, byGranter[g.ID], g.DisplayName)
		}
	}
}

func TestGetActorsReadsDistinctIdsInOneStatement(t *testing.T) {
	c := newCountedManager(t)
	var ids []string
	for i := 0; i < 6; i++ {
		ids = append(ids, mustActor(t, c.mgr, fmt.Sprintf("Member %d", i)).ID)
	}
	ids = append(ids, ids[0], ids[3])

	c.log.forgetStatements()
	out, err := c.mgr.GetActors(context.Background(), ids)
	if err != nil {
		t.Fatalf("GetActors: %v", err)
	}
	statements := c.log.statementsTaken()

	if len(out) != 6 {
		t.Errorf("actors = %d, want 6", len(out))
	}
	if len(statements) != 1 {
		t.Errorf("GetActors took %d statements for 6 distinct ids, want 1:\n%s", len(statements), strings.Join(statements, "\n"))
	}
}

func TestGetActorsAnswersAsGetActorDoesForUnknownDeletedAndRepeatedIds(t *testing.T) {
	c := newCountedManager(t)
	ctx := context.Background()
	var live []mwanachamaactor.Actor
	for i := 0; i < 8; i++ {
		a := mustActor(t, c.mgr, fmt.Sprintf("Member %d", i))
		got, err := c.mgr.GetActor(ctx, a.ID)
		if err != nil {
			t.Fatalf("GetActor %s: %v", a.ID, err)
		}
		live = append(live, got)
	}
	gone := mustActor(t, c.mgr, "Gone")
	c.softDeleteActor(t, gone.ID)
	sort.Slice(live, func(i, j int) bool { return live[i].ID < live[j].ID })

	asked := []string{"missing", gone.ID, "", live[0].ID}
	for i := len(live) - 1; i >= 0; i-- {
		asked = append(asked, live[i].ID, live[i].ID)
	}
	out, err := c.mgr.GetActors(ctx, asked)
	if err != nil {
		t.Fatalf("GetActors: %v", err)
	}

	if !reflect.DeepEqual(out, live) {
		t.Errorf("GetActors = %+v, want the eight live actors once each in id order, as GetActor reads them: %+v", out, live)
	}
}

func TestGetActorsOfNothingKnownIsAnEmptyListNotNil(t *testing.T) {
	c := newCountedManager(t)
	for name, ids := range map[string][]string{
		"nil":          nil,
		"empty":        {},
		"only unknown": {"missing", ""},
	} {
		out, err := c.mgr.GetActors(context.Background(), ids)
		if err != nil {
			t.Fatalf("%s: GetActors: %v", name, err)
		}
		if out == nil || len(out) != 0 {
			t.Errorf("%s: GetActors = %#v, want an empty, non-nil list", name, out)
		}
	}
}

func TestGetActorsWithNoIdsTakesNoStatement(t *testing.T) {
	c := newCountedManager(t)
	for name, ids := range map[string][]string{"nil": nil, "empty": {}} {
		c.log.forgetStatements()
		if _, err := c.mgr.GetActors(context.Background(), ids); err != nil {
			t.Fatalf("%s: GetActors: %v", name, err)
		}
		if statements := c.log.statementsTaken(); len(statements) != 0 {
			t.Errorf("%s: GetActors took statements %v, want none", name, statements)
		}
	}
}

func TestGetActorsReadsMoreIdsThanOneStatementCanBind(t *testing.T) {
	const total = 33000
	c := newCountedManager(t)
	seed := "with recursive n(i) as (select 1 union all select i + 1 from n where i < ?) " +
		"insert into " + c.table + " (id, code, display_name, deleted) select printf('a-%05d', i), printf('c-%05d', i), 'n', 0 from n"
	if err := c.db.Exec(seed, total).Error; err != nil {
		t.Fatalf("seed %d actors: %v", total, err)
	}
	ids := make([]string, 0, total+1)
	for i := total; i >= 1; i-- {
		ids = append(ids, fmt.Sprintf("a-%05d", i))
	}
	ids = append(ids, "a-99999", ids[0], ids[total-1])

	out, err := c.mgr.GetActors(context.Background(), ids)
	if err != nil {
		t.Fatalf("GetActors: %v", err)
	}

	if len(out) != total {
		t.Fatalf("actors = %d, want %d", len(out), total)
	}
	for i, a := range out {
		if want := fmt.Sprintf("a-%05d", i+1); a.ID != want {
			t.Fatalf("actor %d is %s, want %s", i, a.ID, want)
		}
	}
}
