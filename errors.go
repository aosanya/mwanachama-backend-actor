package mwanachamaactor

import "errors"

var ErrActorNotFound = errors.New("mwanachamaactor: actor not found")

var ErrGroupNotFound = errors.New("mwanachamaactor: group not found")

var ErrInvalidActor = errors.New("mwanachamaactor: invalid actor")

var ErrInvalidGroup = errors.New("mwanachamaactor: invalid group")

var ErrDuplicateAttribute = errors.New("mwanachamaactor: attribute value already in use")

var ErrDuplicateID = errors.New("mwanachamaactor: an actor with this id already exists")

var ErrRootCannotMove = errors.New("mwanachamaactor: the root group has no parent, so it cannot be moved")

var ErrParentIsSelf = errors.New("mwanachamaactor: a group cannot be moved under itself")

var ErrParentInSubtree = errors.New("mwanachamaactor: a group cannot be moved beneath its own subtree")

var ErrParentNotFound = errors.New("mwanachamaactor: parent group not found")

var ErrRoleKindNotFound = errors.New("mwanachamaactor: role kind not found")

var ErrAssignmentNotFound = errors.New("mwanachamaactor: role assignment not found")

var ErrKindHasLiveAssignments = errors.New("mwanachamaactor: role kind still has live assignments")

var ErrKindRetired = errors.New("mwanachamaactor: role kind is retired — un-retire it before granting a seat")

var ErrHierarchyNotFound = errors.New("mwanachamaactor: hierarchy not found")

var ErrLevelNotFound = errors.New("mwanachamaactor: level not found")

var ErrLevelWornByGroups = errors.New("mwanachamaactor: this level still carries groups, so it cannot be removed")

var ErrLevelHasRoleKinds = errors.New("mwanachamaactor: this level still carries role kinds, so it cannot be removed")

var ErrGroupTypeNotFound = errors.New("mwanachamaactor: group type not found")

var ErrInvalidGroupType = errors.New("mwanachamaactor: invalid group type")

var ErrGroupTypeWornByGroups = errors.New("mwanachamaactor: this group type still carries groups, so it cannot be removed")

var ErrDuplicateDefaultAnchor = errors.New("mwanachamaactor: hierarchy already has a default anchor level")

var ErrInvalidJoinMode = errors.New("mwanachamaactor: join mode must be open or confirmed")

var ErrInvalidJoinRequest = errors.New("mwanachamaactor: invalid join request")

var ErrJoinMessageTooLong = errors.New("mwanachamaactor: a join request's message is at most 500 characters")

var ErrJoinRequestNotFound = errors.New("mwanachamaactor: join request not found")

var ErrJoinRequestDecided = errors.New("mwanachamaactor: join request is not pending, so it cannot be decided")

var ErrDeclineReasonTooLong = errors.New("mwanachamaactor: a decline's reason is at most 500 characters")

var ErrQuietDeclineReason = errors.New("mwanachamaactor: a quiet decline tells the actor nothing, so it carries no reason")

var ErrInvalidInvitation = errors.New("mwanachamaactor: invalid invitation")

var ErrInvitationNotFound = errors.New("mwanachamaactor: invitation not found")

var ErrInvitationAnswered = errors.New("mwanachamaactor: invitation is not pending, so it cannot be answered")

var ErrInvitationLapsed = errors.New("mwanachamaactor: invitation has lapsed")

var ErrAlreadyInGroup = errors.New("mwanachamaactor: the actor is already registered at this group")

var ErrNotGroupLeader = errors.New("mwanachamaactor: only the group's leaders may do this")

var ErrInvalidCodeFormat = errors.New("mwanachamaactor: invalid code format")
