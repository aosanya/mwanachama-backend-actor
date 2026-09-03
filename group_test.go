package mwanachamauser_test

import (
	"context"
	"errors"
	"testing"

	mwanachamauser "github.com/aosanya/mwanachama-backend-user"
)

func TestCreateGroup_RequiresName(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateGroup(context.Background(), testAgency, mwanachamauser.Group{}); !errors.Is(err, mwanachamauser.ErrInvalidGroup) {
		t.Fatalf("err = %v, want ErrInvalidGroup", err)
	}
}

func TestCreateGroup_RoundTrips(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	g, err := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{
		Name:         "National Council",
		HierarchyID:  "hier-1",
		LevelID:      "lvl-1",
		Discoverable: true,
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.ID == "" {
		t.Error("expected minted ID")
	}
	got, err := mgr.GetGroup(ctx, testAgency, g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.Name != "National Council" || got.HierarchyID != "hier-1" || !got.Discoverable {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestGetGroup_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetGroup(context.Background(), testAgency, "nope"); !errors.Is(err, mwanachamauser.ErrGroupNotFound) {
		t.Fatalf("err = %v, want ErrGroupNotFound", err)
	}
}

func TestEditGroup_EmptyClears(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	g, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Ward A"})
	edited, err := mgr.EditGroup(ctx, testAgency, g.ID, mwanachamauser.GroupEdit{
		Name:     "Ward A Renamed",
		NodeType: "committee",
	})
	if err != nil {
		t.Fatalf("EditGroup: %v", err)
	}
	if edited.Name != "Ward A Renamed" || edited.NodeType != "committee" {
		t.Errorf("first edit mismatch: %+v", edited)
	}

	// Second edit with empty fields clears them — mirrors the gateway's
	// EditChapter contract exactly (every field written on every call).
	cleared, err := mgr.EditGroup(ctx, testAgency, g.ID, mwanachamauser.GroupEdit{Name: "Ward A"})
	if err != nil {
		t.Fatalf("EditGroup (clear): %v", err)
	}
	if cleared.NodeType != "" {
		t.Errorf("NodeType = %q, want cleared", cleared.NodeType)
	}
}

func TestMoveGroup_RootCannotMove(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Root"})
	child, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Child", ParentID: root.ID})

	if _, err := mgr.MoveGroup(ctx, testAgency, root.ID, child.ID); !errors.Is(err, mwanachamauser.ErrRootCannotMove) {
		t.Fatalf("err = %v, want ErrRootCannotMove", err)
	}
}

func TestMoveGroup_ParentIsSelf(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Root"})
	child, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Child", ParentID: root.ID})

	if _, err := mgr.MoveGroup(ctx, testAgency, child.ID, child.ID); !errors.Is(err, mwanachamauser.ErrParentIsSelf) {
		t.Fatalf("err = %v, want ErrParentIsSelf", err)
	}
}

func TestMoveGroup_ParentInSubtree(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Root"})
	mid, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Mid", ParentID: root.ID})
	leaf, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Leaf", ParentID: mid.ID})

	// Moving mid beneath its own descendant leaf must be refused.
	if _, err := mgr.MoveGroup(ctx, testAgency, mid.ID, leaf.ID); !errors.Is(err, mwanachamauser.ErrParentInSubtree) {
		t.Fatalf("err = %v, want ErrParentInSubtree", err)
	}
}

func TestMoveGroup_Succeeds(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Root"})
	a, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "A", ParentID: root.ID})
	b, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "B", ParentID: root.ID})

	moved, err := mgr.MoveGroup(ctx, testAgency, a.ID, b.ID)
	if err != nil {
		t.Fatalf("MoveGroup: %v", err)
	}
	if moved.ParentID != b.ID {
		t.Errorf("ParentID = %q, want %q", moved.ParentID, b.ID)
	}
}

func TestListGroupChildren_EmptyParentReturnsRoots(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root1, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Root1"})
	root2, _ := mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Root2"})
	_, _ = mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Child", ParentID: root1.ID})

	roots, err := mgr.ListGroupChildren(ctx, testAgency, "")
	if err != nil {
		t.Fatalf("ListGroupChildren: %v", err)
	}
	if len(roots) != 2 {
		t.Fatalf("len(roots) = %d, want 2", len(roots))
	}
	ids := map[string]bool{root1.ID: true, root2.ID: true}
	for _, r := range roots {
		if !ids[r.ID] {
			t.Errorf("unexpected root %v", r)
		}
	}
}

func TestListDiscoverableGroups_FiltersByFlagAndQuery(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, _ = mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Kibera Ward", Discoverable: true})
	_, _ = mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Hidden Ward", Discoverable: false})
	_, _ = mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "Nairobi Central", Discoverable: true})

	all, err := mgr.ListDiscoverableGroups(ctx, testAgency, "")
	if err != nil {
		t.Fatalf("ListDiscoverableGroups: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("len(all) = %d, want 2", len(all))
	}

	filtered, err := mgr.ListDiscoverableGroups(ctx, testAgency, "kibera")
	if err != nil {
		t.Fatalf("ListDiscoverableGroups(kibera): %v", err)
	}
	if len(filtered) != 1 || filtered[0].Name != "Kibera Ward" {
		t.Errorf("filtered = %v, want just Kibera Ward", filtered)
	}
}

func TestListGroups_FilteredByHierarchy(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, _ = mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "A", HierarchyID: "h1"})
	_, _ = mgr.CreateGroup(ctx, testAgency, mwanachamauser.Group{Name: "B", HierarchyID: "h2"})

	out, err := mgr.ListGroups(ctx, testAgency, "h1")
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(out) != 1 || out[0].Name != "A" {
		t.Errorf("out = %v, want just A", out)
	}

	all, err := mgr.ListGroups(ctx, testAgency, "")
	if err != nil {
		t.Fatalf("ListGroups(all): %v", err)
	}
	if len(all) != 2 {
		t.Errorf("len(all) = %d, want 2", len(all))
	}
}
