package routes

import "context"

// HierarchyChecker answers the two reference questions CreateGroup and
// EditGroup need about the hierarchy/level tables — DSN-1699 gap 1, resolved:
// those tables live in this same repo now (see [mwanachamaactor.UserManager],
// which satisfies this interface directly), but the small interface stays so
// this package still takes no dependency on the full UserManager surface, or
// on the models package, for the one thing it actually needs from it.
type HierarchyChecker interface {
	// HierarchyExists reports whether id names a hierarchy.
	HierarchyExists(ctx context.Context, id string) (bool, error)
	// LevelInHierarchy reports whether levelID names a level belonging to
	// hierarchyID.
	LevelInHierarchy(ctx context.Context, levelID, hierarchyID string) (bool, error)
}
