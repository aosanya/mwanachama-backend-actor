package mwanachamaactor

import "errors"

// ErrMemberNotFound is returned when a Member id has no record.
var ErrMemberNotFound = errors.New("mwanachamaactor: member not found")

// ErrGroupNotFound is returned when a Group id has no record.
var ErrGroupNotFound = errors.New("mwanachamaactor: group not found")

// ErrInvalidMember is returned when a Member write is missing a required field.
var ErrInvalidMember = errors.New("mwanachamaactor: invalid member")

// ErrInvalidGroup is returned when a Group write is missing a required field.
var ErrInvalidGroup = errors.New("mwanachamaactor: invalid group")

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
