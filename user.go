package mwanachamaactor

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

// Actor, Group, GroupEdit, ActorGroupAssignment, RoleKind, and
// ActorRoleAssignment are aliases of their models. counterparts, so a caller
// needs only this package's import, never models's directly.
type (
	Actor                = models.Actor
	Group                = models.Group
	GroupEdit            = models.GroupEdit
	ActorGroupAssignment = models.ActorGroupAssignment
	RoleKind             = models.RoleKind
	ActorRoleAssignment  = models.ActorRoleAssignment
	Hierarchy            = models.Hierarchy
	Level                = models.Level
)

// TimeLayout is the timestamp layout every model in this package is written
// and read with. See [models.TimeLayout].
const TimeLayout = models.TimeLayout

// EndedByRevocation, EndedByResignation, EndedByEviction, and
// EndedByDeparture are the four acts that can end an ActorRoleAssignment.
// See [models.EndReason].
const (
	EndedByRevocation  = models.EndedByRevocation
	EndedByResignation = models.EndedByResignation
	EndedByEviction    = models.EndedByEviction
	EndedByDeparture   = models.EndedByDeparture
)

// NowRFC3339 returns the current UTC time formatted per [TimeLayout]. See
// [models.NowRFC3339].
func NowRFC3339() string { return models.NowRFC3339() }

// UserManager is the primary interface for Actor and Group lifecycle
// management, and for the ActorGroupAssignment edge between them. This is a
// single-tenant package — one deployment serves one agency, so no method
// takes a tenant-scoping argument, mirroring
// mwanachama-backend-taskmanager.TaskManager's shape.
//
// Implementations must be safe for concurrent use.
type UserManager interface {
	// CreateActor creates a new Actor entity. Assigns a server-generated
	// ID when a.ID is empty (the storage backend mints a UUID — see
	// mwanachama-backend-shared/postgres/entities.go's CreateEntity, which has
	// no caller-supplied-ID path).
	CreateActor(ctx context.Context, a Actor) (Actor, error)

	// GetActor retrieves a single Actor by entity ID. Returns
	// [ErrActorNotFound] if no matching actor exists.
	GetActor(ctx context.Context, id string) (Actor, error)

	// GetActors returns the Actors for the given ids, sorted by id. Ids
	// with no actor are skipped rather than erroring.
	GetActors(ctx context.Context, ids []string) ([]Actor, error)

	// SetActorDisplayName records the name an actor gave for themselves.
	// Returns [ErrActorNotFound] if the actor does not exist.
	SetActorDisplayName(ctx context.Context, id, displayName string) (Actor, error)

	// ListActors returns every non-deleted Actor, id order.
	ListActors(ctx context.Context) ([]Actor, error)

	// CreateGroup creates a new Group entity. Returns [ErrInvalidGroup] when
	// g.Name is empty.
	CreateGroup(ctx context.Context, g Group) (Group, error)

	// GetGroup retrieves a single Group by entity ID. Returns
	// [ErrGroupNotFound] if no matching group exists.
	GetGroup(ctx context.Context, id string) (Group, error)

	// EditGroup writes the three columns Group.GroupEdit carries — name,
	// node_type, anchor_level_override — clearing whichever are empty.
	// Returns [ErrGroupNotFound] if the group does not exist.
	EditGroup(ctx context.Context, id string, e GroupEdit) (Group, error)

	// MoveGroup re-parents a group. Returns [ErrRootCannotMove],
	// [ErrParentIsSelf], [ErrParentInSubtree], or [ErrParentNotFound] for the
	// four moves that cannot mean anything.
	MoveGroup(ctx context.Context, id, newParentID string) (Group, error)

	// ListGroupChildren returns the direct children of a group (empty
	// parentID returns the roots).
	ListGroupChildren(ctx context.Context, parentID string) ([]Group, error)

	// ListGroups returns every group, optionally filtered to one hierarchy
	// (empty hierarchyID returns every group).
	ListGroups(ctx context.Context, hierarchyID string) ([]Group, error)

	// ListDiscoverableGroups returns groups flagged discoverable, optionally
	// filtered by a case-insensitive name substring (empty query returns all).
	ListDiscoverableGroups(ctx context.Context, query string) ([]Group, error)

	// AssignGroup enrols an actor at a group by writing (or updating) the
	// registered_at edge. Idempotent on (actorID, groupID): a second call
	// updates Attributes in place and preserves the original CreatedAt —
	// mirrors DEV-1319. Unlike the gateway's pre-DSN-1698 behaviour,
	// registering a new home group does NOT clear any other registration's
	// home flag, wherever a caller keeps one — decision 8 drops that
	// exclusivity outright.
	AssignGroup(ctx context.Context, r ActorGroupAssignment) (ActorGroupAssignment, error)

	// Deregister removes an actor's registered_at edge to a group. Returns
	// the removed ActorGroupAssignment and found=true if one existed;
	// found=false and a zero ActorGroupAssignment if the actor was not
	// registered at the group (idempotent no-op — mirrors the gateway
	// store's Deregister semantics). Unlike the gateway's
	// Repository.Deregister, this does NOT write any act-log row — the act
	// log lives in the gateway's custody domain, which this package
	// deliberately does not depend on; composing the two is the gateway
	// adapter's job.
	Deregister(ctx context.Context, actorID, groupID string) (ActorGroupAssignment, bool, error)

	// ListGroupsForActor returns every group an actor is registered at.
	ListGroupsForActor(ctx context.Context, actorID string) ([]ActorGroupAssignment, error)

	// ListActorsForGroup returns every actor registered at a group.
	ListActorsForGroup(ctx context.Context, groupID string) ([]ActorGroupAssignment, error)

	// HomeCounts returns how many actors call each group home, keyed by
	// group id, reading the caller-declared Attributes["is_home"] JSON path
	// (there is no dedicated column — see models.ActorGroupAssignment's
	// doc). Groups with nobody are absent rather than zero.
	HomeCounts(ctx context.Context) (map[string]int, error)

	// CreateRoleKind creates a new position with a capability set. Assigns a
	// server-generated ID when k.ID is empty.
	CreateRoleKind(ctx context.Context, k RoleKind) (RoleKind, error)
	// ListRoleKinds returns every role kind, id order.
	ListRoleKinds(ctx context.Context) ([]RoleKind, error)
	// GetRoleKind retrieves a single RoleKind by id. Returns
	// [ErrRoleKindNotFound] if no matching kind exists.
	GetRoleKind(ctx context.Context, id string) (RoleKind, error)
	// RetireRoleKind ends a role kind, stamping the current time into
	// RetiredAt and actorID into RetiredBy. Returns
	// [ErrKindHasLiveAssignments] when any active assignment still names the
	// kind. Retiring an already-retired kind is idempotent: the original
	// stamp is preserved. actorID is the mounting process's caller, resolved
	// from its own session — this package never reads one from a request
	// body.
	//
	// **Does not write any audit-log row.** That composition belongs to
	// whatever domain the mounting process keeps for it (the gateway's
	// custody domain, today) — see Deregister's doc for why this package
	// never depends on one.
	RetireRoleKind(ctx context.Context, kindID, actorID string) (RoleKind, error)
	// UnretireRoleKind reverses a retirement, clearing both ending fields
	// together. Un-retiring a live kind is idempotent. Takes no actor: unlike
	// RetireRoleKind, nothing here stores who reversed it — a caller that
	// wants that composes it the same way it composes RetireRoleKind's log
	// row.
	UnretireRoleKind(ctx context.Context, kindID string) (RoleKind, error)

	// GrantRole assigns a role kind to an actor at a group. Returns
	// [ErrKindRetired] when the kind is retired, [ErrRoleKindNotFound] when
	// it does not exist.
	//
	// **Does not write any audit-log row** — same reason as RetireRoleKind.
	GrantRole(ctx context.Context, a ActorRoleAssignment) (ActorRoleAssignment, error)
	// GetRoleAssignment returns one assignment by id, active or not.
	GetRoleAssignment(ctx context.Context, id string) (ActorRoleAssignment, error)
	// RevokeRole deactivates an assignment (an operator removing the
	// holder). Re-revoking an already-ended seat is a no-op.
	RevokeRole(ctx context.Context, assignmentID string) error
	// StepDownRole deactivates an assignment at the holder's own request.
	// Same idempotence as RevokeRole.
	StepDownRole(ctx context.Context, assignmentID string) error
	// EndRoleOnEviction deactivates an assignment because an operator ended
	// the holder's membership at the seat's group.
	EndRoleOnEviction(ctx context.Context, assignmentID string) error
	// EndRoleOnDeparture deactivates an assignment because the holder ended
	// their own membership at the seat's group.
	EndRoleOnDeparture(ctx context.Context, assignmentID string) error
	// ListRoleAssignmentsForGroup returns assignments at a group (active
	// only when activeOnly is true), (granted_at, id) order.
	ListRoleAssignmentsForGroup(ctx context.Context, groupID string, activeOnly bool) ([]ActorRoleAssignment, error)
	// ListRoleAssignmentsForActor returns every assignment an actor holds,
	// across every group (active only when activeOnly is true), (granted_at,
	// id) order.
	ListRoleAssignmentsForActor(ctx context.Context, actorID string, activeOnly bool) ([]ActorRoleAssignment, error)

	// GroupDashboard returns the roles-with-members view at a group plus
	// rollup counts — DEV-1660, a read model over RoleKind/ActorRoleAssignment
	// and Actor/Group data that only became intra-repo once role landed
	// here. Returns [ErrGroupNotFound] if the group does not exist.
	GroupDashboard(ctx context.Context, groupID string) (models.GroupDashboard, error)

	// CreateHierarchy creates a new named ladder of levels. Assigns a
	// server-generated ID when h.ID is empty.
	CreateHierarchy(ctx context.Context, h Hierarchy) (Hierarchy, error)
	// ListHierarchies returns every hierarchy, id order.
	ListHierarchies(ctx context.Context) ([]Hierarchy, error)
	// GetHierarchy retrieves a single Hierarchy by id. Returns
	// [ErrHierarchyNotFound] if no matching hierarchy exists.
	GetHierarchy(ctx context.Context, id string) (Hierarchy, error)
	// RenameHierarchy writes a hierarchy's name and nothing else. Returns
	// [ErrHierarchyNotFound] if the hierarchy does not exist.
	RenameHierarchy(ctx context.Context, id, name string) (Hierarchy, error)

	// CreateLevel creates a new rung on a hierarchy. Assigns a
	// server-generated ID when l.ID is empty. Returns [ErrHierarchyNotFound]
	// when l.HierarchyID names no hierarchy, and
	// [ErrDuplicateDefaultAnchor] when l.IsDefaultAnchor is set and the
	// hierarchy already has one.
	CreateLevel(ctx context.Context, l Level) (Level, error)
	// GetLevel retrieves a single Level by id. Returns [ErrLevelNotFound]
	// if no matching level exists.
	GetLevel(ctx context.Context, id string) (Level, error)
	// RenameLevel writes a level's name and nothing else. Returns
	// [ErrLevelNotFound] if the level does not exist.
	RenameLevel(ctx context.Context, id, name string) (Level, error)
	// DeleteLevel removes a rung. Returns [ErrLevelWornByGroups] when a
	// Group still wears it (LevelID or AnchorLevelOverrideID), or
	// [ErrLevelHasRoleKinds] when a RoleKind is still scoped to it.
	DeleteLevel(ctx context.Context, id string) error
	// SetDefaultAnchor moves a hierarchy's anchor rung — the level at which
	// a self-service member registers — clearing the old flag and setting
	// the new one in one transaction. Returns [ErrLevelNotFound] if levelID
	// does not exist or does not belong to hierarchyID. A no-op when
	// levelID already carries the flag.
	SetDefaultAnchor(ctx context.Context, hierarchyID, levelID string) error
	// ListLevels returns the levels of a hierarchy, or — when hierarchyID
	// is empty — every level across every hierarchy, depth-then-id order.
	ListLevels(ctx context.Context, hierarchyID string) ([]Level, error)

	// HierarchyExists reports whether id names a hierarchy.
	HierarchyExists(ctx context.Context, id string) (bool, error)
	// LevelInHierarchy reports whether levelID names a level belonging to
	// hierarchyID.
	LevelInHierarchy(ctx context.Context, levelID, hierarchyID string) (bool, error)
}

// userManager is the concrete implementation of [UserManager].
type userManager struct {
	db     *gorm.DB
	tables TableNames
}

// NewUserManager constructs a [UserManager] backed by db, reading and
// writing the three tables named by t (see [DefaultTableNames]). Callers
// must run [Migrate] against the same db and t before use. Returns an error
// if db is nil.
func NewUserManager(db *gorm.DB, t TableNames) (UserManager, error) {
	if db == nil {
		return nil, fmt.Errorf("NewUserManager: db must not be nil")
	}
	return &userManager{db: db, tables: t}, nil
}
