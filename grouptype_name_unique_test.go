package mwanachamaactor

import (
	"context"
	"errors"
	"testing"
)

func TestCreateGroupTypeRefusesASecondTypeWithTheSameNameInTheSameHierarchy(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	h, err := um.CreateHierarchy(ctx, Hierarchy{Name: "Municipality"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	first, err := um.CreateGroupType(ctx, GroupType{Name: "Ward", HierarchyID: h.ID})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := um.CreateGroupType(ctx, GroupType{Name: " WARD ", HierarchyID: h.ID}); !errors.Is(err, ErrDuplicateGroupTypeName) {
		t.Fatalf("second: err = %v", err)
	}
	types, err := um.ListGroupTypes(ctx, h.ID)
	if err != nil {
		t.Fatalf("ListGroupTypes: %v", err)
	}
	if len(types) != 1 || types[0].ID != first.ID {
		t.Fatalf("types = %d, want only the first", len(types))
	}
}

func TestCreateGroupTypeAllowsTheSameNameInAnotherHierarchy(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	a, err := um.CreateHierarchy(ctx, Hierarchy{Name: "Municipality"})
	if err != nil {
		t.Fatalf("CreateHierarchy a: %v", err)
	}
	b, err := um.CreateHierarchy(ctx, Hierarchy{Name: "Association"})
	if err != nil {
		t.Fatalf("CreateHierarchy b: %v", err)
	}
	if _, err := um.CreateGroupType(ctx, GroupType{Name: "Board", HierarchyID: a.ID}); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := um.CreateGroupType(ctx, GroupType{Name: "Board", HierarchyID: b.ID}); err != nil {
		t.Fatalf("same name in another hierarchy: %v", err)
	}
}
