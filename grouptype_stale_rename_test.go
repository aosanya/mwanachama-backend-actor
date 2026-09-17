package mwanachamaactor_test

// Pins board row ACT2 (documentation/3. implementation/todo.md):
// DeleteGroupType's worn-by-Groups guard checks
// `Group.NodeType = <the GroupType's CURRENT Name>` (grouptype_impl.go's
// DeleteGroupType), but EditGroupType renaming a GroupType never touches any
// Group's own NodeType column (documented as intentional — NodeType is free
// text with no live foreign key). Once a GroupType is renamed, the guard is
// checking a name no Group can possibly still carry, so it always reports
// zero and DeleteGroupType always succeeds — even while a Group still names
// the type's OLD, now-orphaned name. The delete goes through with no error,
// leaving a live Group whose NodeType matches no existing GroupType, and
// nothing else in this package (or the Postman/route surface) can detect it
// afterward: GetGroupType(id) correctly 404s, but there is no reverse lookup
// from an orphaned NodeType string back to "this used to be GroupType X".
//
// Once fixed (DeleteGroupType should refuse while any Group's NodeType
// matches ANY name the GroupType has ever held — in practice, tracking
// worn-ness by GroupType ID rather than by the current Name string, the same
// way DeleteLevel/DeleteGroupType's own doc comment says the check should
// mirror DeleteLevel's — or EditGroupType should cascade the rename onto
// every Group.NodeType currently matching the old name), this test's second
// DeleteGroupType call should return ErrGroupTypeWornByGroups instead of nil,
// and should go red at that assertion.
import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestDeleteGroupType_PinsStaleWornCheckAfterRename(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "H"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	level, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "National", IsDefaultAnchor: true})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	dept, err := mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: h.ID, Name: "Department"})
	if err != nil {
		t.Fatalf("CreateGroupType: %v", err)
	}
	grp, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{
		HierarchyID: h.ID, LevelID: level.ID, Name: "Comms", NodeType: "Department",
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	// Confirm the guard works BEFORE any rename — this is the behavior
	// TestGroupType_DeleteWornRefused already covers, kept here only as a
	// control so the contrast with the post-rename case is explicit.
	if err := mgr.DeleteGroupType(ctx, dept.ID); !errors.Is(err, mwanachamaactor.ErrGroupTypeWornByGroups) {
		t.Fatalf("expected ErrGroupTypeWornByGroups before any rename, got %v", err)
	}

	// Rename the GroupType. Group.NodeType is deliberately left untouched.
	if _, err := mgr.EditGroupType(ctx, dept.ID, "Committee", "", "", ""); err != nil {
		t.Fatalf("EditGroupType: %v", err)
	}

	// BUG: the worn check now compares against "Committee", which no Group
	// carries, so it reports zero matches and the delete silently succeeds
	// — even though `grp` still carries NodeType="Department", now orphaned.
	err = mgr.DeleteGroupType(ctx, dept.ID)
	if err != nil {
		t.Fatalf("pinning current (broken) behavior: expected DeleteGroupType to succeed after rename despite the Group still referencing the old name, got error %v — if this now fails with ErrGroupTypeWornByGroups, ACT2 is fixed; update this test to assert that instead", err)
	}

	if _, err := mgr.GetGroupType(ctx, dept.ID); !errors.Is(err, mwanachamaactor.ErrGroupTypeNotFound) {
		t.Fatalf("expected the GroupType to really be gone, got %v", err)
	}

	stillGrp, err := mgr.GetGroup(ctx, grp.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if stillGrp.NodeType != "Department" {
		t.Fatalf("expected the Group's NodeType to remain the orphaned %q, got %q", "Department", stillGrp.NodeType)
	}
}
