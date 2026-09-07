package mwanachamaactor

import "errors"

// ErrActorNotFound is returned when an Actor id has no record.
var ErrActorNotFound = errors.New("mwanachamaactor: actor not found")

// ErrGroupNotFound is returned when a Group id has no record.
var ErrGroupNotFound = errors.New("mwanachamaactor: group not found")

// ErrInvalidActor is returned when an Actor write is missing a required field.
var ErrInvalidActor = errors.New("mwanachamaactor: invalid actor")

// ErrInvalidGroup is returned when a Group write is missing a required field.
var ErrInvalidGroup = errors.New("mwanachamaactor: invalid group")

// ErrDuplicateAttribute is returned when a write's Attributes value for a
// [models.Property] marked Unique already appears on another row of the
// same model — see attributes.go's checkUniqueAttributes.
var ErrDuplicateAttribute = errors.New("mwanachamaactor: attribute value already in use")

// ErrRootCannotMove — the root group has no parent, so there is nothing to
// move it under. Mirrors the gateway's chapter.ErrRootCannotMove.
var ErrRootCannotMove = errors.New("mwanachamaactor: the root group has no parent, so it cannot be moved")

// ErrParentIsSelf — a group cannot be its own parent.
var ErrParentIsSelf = errors.New("mwanachamaactor: a group cannot be moved under itself")

// ErrParentInSubtree — the new parent is a descendant of the group being
// moved, which would cut the whole subtree off from the root.
var ErrParentInSubtree = errors.New("mwanachamaactor: a group cannot be moved beneath its own subtree")

// ErrParentNotFound — MoveGroup's newParentID names no Group.
var ErrParentNotFound = errors.New("mwanachamaactor: parent group not found")

// ErrRoleKindNotFound is returned when a role kind id has no record.
var ErrRoleKindNotFound = errors.New("mwanachamaactor: role kind not found")

// ErrAssignmentNotFound is returned when a role assignment id has no record.
var ErrAssignmentNotFound = errors.New("mwanachamaactor: role assignment not found")

// ErrKindHasLiveAssignments refuses a retirement while somebody still holds
// the kind — G229, ported unchanged from the gateway's role.ErrKindHasLiveAssignments.
var ErrKindHasLiveAssignments = errors.New("mwanachamaactor: role kind still has live assignments")

// ErrKindRetired refuses a grant that names a role kind which has been
// retired — DEV-1201, ported unchanged from the gateway's role.ErrKindRetired.
var ErrKindRetired = errors.New("mwanachamaactor: role kind is retired — un-retire it before granting a seat")

// ErrHierarchyNotFound is returned when a Hierarchy id has no record.
var ErrHierarchyNotFound = errors.New("mwanachamaactor: hierarchy not found")

// ErrLevelNotFound is returned when a Level id has no record.
var ErrLevelNotFound = errors.New("mwanachamaactor: level not found")

// ErrLevelWornByGroups refuses a level delete while a Group still wears the
// rung (LevelID or AnchorLevelOverrideID). Mirrors the gateway's
// chapter.ErrLevelWornByChapters.
var ErrLevelWornByGroups = errors.New("mwanachamaactor: this level still carries groups, so it cannot be removed")

// ErrLevelHasRoleKinds refuses a level delete while a RoleKind is still
// scoped to it. Mirrors the gateway's chapter.ErrLevelHasRoleKinds.
var ErrLevelHasRoleKinds = errors.New("mwanachamaactor: this level still carries role kinds, so it cannot be removed")

// ErrDuplicateDefaultAnchor refuses a second default-anchor level on the
// same hierarchy — at most one may carry the flag. The DB-level half of this
// guarantee is gormstore.Migrate's partial unique index.
var ErrDuplicateDefaultAnchor = errors.New("mwanachamaactor: hierarchy already has a default anchor level")
