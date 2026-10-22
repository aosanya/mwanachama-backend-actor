package mwanachamaactor

import (
	"reflect"
	"sort"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func TestBlueprintParses(t *testing.T) {
	b, err := Blueprint()
	if err != nil {
		t.Fatalf("Blueprint: %v", err)
	}
	if b.Module != "actor" {
		t.Errorf("module = %q, want %q", b.Module, "actor")
	}
	want := []string{
		"actor", "actor_group_assignment", "actor_role_assignment", "code_format", "code_sequence",
		"group", "group_invitation", "group_join_request", "group_name_translation", "group_removal", "group_type", "hierarchy", "join_decision", "level", "role_kind",
	}
	if got := b.Roles(); !reflect.DeepEqual(got, want) {
		t.Errorf("roles = %v, want %v", got, want)
	}
}

func TestTheShippedDomainSpecLoads(t *testing.T) {
	s, err := SpecFor("wakala")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if s.Instance != "wakala" {
		t.Errorf("instance = %q, want %q", s.Instance, "wakala")
	}
	if s.MountName() != spec.DefaultMount {
		t.Errorf("mount = %q, want the default %q", s.MountName(), spec.DefaultMount)
	}
	o, ok := s.ByRole(roleActor)
	if !ok {
		t.Fatal("the spec fills no object for the role actor")
	}
	if got, want := s.RawNameFor(o), "actor_main_actors"; got != want {
		t.Errorf("logical name = %q, want %q", got, want)
	}
	if got, want := s.TableFor(o), "wakala_"+s.MountKey()+"_"+spec.HashName("actor_actors"); got != want {
		t.Errorf("table = %q, want %q", got, want)
	}
}

func TestASecondMountTakesItsOwnTables(t *testing.T) {
	first, err := SpecFor("wakala")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	second, err := SpecForMount("wakala", "shadow")
	if err != nil {
		t.Fatalf("SpecForMount: %v", err)
	}
	for _, role := range []string{roleActor, roleGroup, roleCodeSequence} {
		a, ok := first.ByRole(role)
		if !ok {
			t.Fatalf("the default mount fills no object for the role %q", role)
		}
		b, ok := second.ByRole(role)
		if !ok {
			t.Fatalf("the second mount fills no object for the role %q", role)
		}
		if first.TableFor(a) == second.TableFor(b) {
			t.Errorf("both mounts put %q in %q, so the second would adopt the first's rows",
				role, first.TableFor(a))
		}
	}
	if got, want := second.RawNameFor(mustRole(t, second, roleActor)), "actor_shadow_actors"; got != want {
		t.Errorf("logical name = %q, want %q", got, want)
	}
}

func mustRole(t *testing.T, s *spec.Spec, role string) spec.Object {
	t.Helper()
	o, ok := s.ByRole(role)
	if !ok {
		t.Fatalf("the spec fills no object for the role %q", role)
	}
	return o
}

func TestEveryPhysicalNameFitsTheIdentifierBudget(t *testing.T) {
	s, err := SpecForMount("wakala", "shadow")
	if err != nil {
		t.Fatalf("SpecForMount: %v", err)
	}
	for _, o := range s.Objects {
		if got := len(s.TableFor(o)); got > spec.MaxIdentifier {
			t.Errorf("table %q needs %d bytes, and the database truncates at %d",
				s.TableFor(o), got, spec.MaxIdentifier)
		}
		for _, idx := range o.Indexes {
			if got := len(s.IndexFor(o, idx)); got > spec.MaxIdentifier {
				t.Errorf("index %q needs %d bytes, and the database truncates at %d",
					s.IndexFor(o, idx), got, spec.MaxIdentifier)
			}
		}
	}
}

func TestEveryCarrierAgreesWithItsObject(t *testing.T) {
	s, err := SpecFor("wakala")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	for role, carrier := range carriers() {
		o, ok := s.ByRole(role)
		if !ok {
			t.Errorf("the spec fills no object for the role %q", role)
			continue
		}
		declared := map[string]bool{}
		for _, f := range o.Fields {
			declared[f.Name] = true
		}
		carried := specstore.ColumnsOf(reflect.TypeOf(carrier))
		for name := range declared {
			if !carried[name] {
				t.Errorf("%s declares %q, which %T does not carry", role, name, carrier)
			}
		}
		for name := range carried {
			if !declared[name] {
				t.Errorf("%T carries %q, which %s does not declare", carrier, name, role)
			}
		}
	}
}

func TestEveryColumnTheRowStructsHeldIsDeclared(t *testing.T) {
	held := map[string][]string{
		roleActor:                {"id", "code", "display_name", "is_agentic", "attributes", "created_at", "last_updated", "deleted"},
		roleGroup:                {"id", "code", "hierarchy_id", "level_id", "parent_id", "name", "discoverable", "anchor_level_override_id", "node_type", "attributes", "created_at", "last_updated", "deleted"},
		roleActorGroupAssignment: {"actor_id", "group_id", "created_at", "last_updated", "deleted", "attributes"},
		roleRoleKind:             {"id", "code", "name", "capabilities", "description", "is_delegate", "leads_group", "level_id", "retired_at", "retired_by"},
		roleActorRoleAssignment:  {"id", "actor_id", "group_id", "kind_id", "active", "granted_at", "granted_by", "ended_at", "ended_reason"},
		roleHierarchy:            {"id", "code", "name"},
		roleLevel:                {"id", "code", "hierarchy_id", "name", "depth", "is_default_anchor"},
		roleGroupType:            {"id", "code", "hierarchy_id", "name", "singular", "plural", "level_id"},
		roleCodeSequence:         {"entity_type", "next_number"},
	}

	s, err := SpecFor("wakala")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}

	total := 0
	for role, columns := range held {
		o, ok := s.ByRole(role)
		if !ok {
			t.Errorf("the spec fills no object for the role %q", role)
			continue
		}
		declared := map[string]bool{}
		for _, f := range o.Fields {
			declared[f.Name] = true
		}
		for _, c := range columns {
			if !declared[c] {
				t.Errorf("%s stored %q and the blueprint declares no field for it", role, c)
			}
		}
		total += len(o.Fields) - len(columnsDeclaredSince[role])
	}
	if total != 63 {
		t.Errorf("the blueprint declares %d fields, and the row structs carried 63", total)
	}
}

func TestEveryDeclaredCodeIsUniqueAndImmutable(t *testing.T) {
	b, err := Blueprint()
	if err != nil {
		t.Fatalf("Blueprint: %v", err)
	}
	var coded []string
	for _, c := range codedRoles {
		coded = append(coded, c.role)
	}
	sort.Strings(coded)

	for _, role := range coded {
		o, ok := b.Object(role)
		if !ok {
			t.Errorf("the blueprint declares no role %q", role)
			continue
		}
		var found bool
		for _, f := range o.Fields {
			if f.Name != "code" {
				continue
			}
			found = true
			if !f.Unique || !f.Immutable || !f.Required {
				t.Errorf("%s.code is unique=%v immutable=%v required=%v, and a minted code is all three",
					role, f.Unique, f.Immutable, f.Required)
			}
		}
		if !found {
			t.Errorf("%s mints a code and declares no field for it", role)
		}
	}
}
