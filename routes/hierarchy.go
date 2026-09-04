package routes

import "context"

// HierarchyChecker answers the two reference questions CreateGroup and
// EditGroup need about the hierarchy/chapter_level tables this package does
// not own (DSN-1699 gap 1's default keeps them physically in the mounting
// process — see models.Group's doc). The mounting process supplies an
// implementation at wiring time; this package takes no import on whatever
// owns those tables in return.
type HierarchyChecker interface {
	// HierarchyExists reports whether id names a hierarchy.
	HierarchyExists(ctx context.Context, id string) (bool, error)
	// LevelInHierarchy reports whether levelID names a level belonging to
	// hierarchyID.
	LevelInHierarchy(ctx context.Context, levelID, hierarchyID string) (bool, error)
}
