package mwanachamauser

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/entitygraph"
)

// memberTypeID is the TypeDefinition.Name used for Member entities.
const memberTypeID = "Member"

// groupTypeID is the TypeDefinition.Name used for Group entities.
const groupTypeID = "Group"

// UserManager is the primary interface for Member and Group lifecycle
// management, and for the Registration edge between them. Every method is
// explicitly scoped to an agencyID argument, mirroring
// mwanachama-backend-taskmanager.TaskManager's shape.
//
// Implementations must be safe for concurrent use.
type UserManager interface {
	// CreateMember creates a new Member entity. Assigns a server-generated
	// ID when m.ID is empty (the storage backend mints a UUID — see
	// mwanachama-backend-shared/postgres/entities.go's CreateEntity, which has
	// no caller-supplied-ID path).
	CreateMember(ctx context.Context, agencyID string, m Member) (Member, error)

	// GetMember retrieves a single Member by entity ID. Returns
	// [ErrMemberNotFound] if no matching member exists.
	GetMember(ctx context.Context, agencyID, id string) (Member, error)

	// GetMembers returns the Members for the given ids, sorted by id. Ids
	// with no member are skipped rather than erroring.
	GetMembers(ctx context.Context, agencyID string, ids []string) ([]Member, error)

	// SetMemberDisplayName records the name a member gave for themselves.
	// Returns [ErrMemberNotFound] if the member does not exist.
	SetMemberDisplayName(ctx context.Context, agencyID, id, displayName string) (Member, error)

	// ListMembers returns every non-deleted Member in the agency, id order.
	ListMembers(ctx context.Context, agencyID string) ([]Member, error)

	// CreateGroup creates a new Group entity. Returns [ErrInvalidGroup] when
	// g.Name is empty.
	CreateGroup(ctx context.Context, agencyID string, g Group) (Group, error)

	// GetGroup retrieves a single Group by entity ID. Returns
	// [ErrGroupNotFound] if no matching group exists.
	GetGroup(ctx context.Context, agencyID, id string) (Group, error)

	// EditGroup writes the three columns Group.GroupEdit carries — name,
	// node_type, anchor_level_override — clearing whichever are empty.
	// Returns [ErrGroupNotFound] if the group does not exist.
	EditGroup(ctx context.Context, agencyID, id string, e GroupEdit) (Group, error)

	// MoveGroup re-parents a group. Returns [ErrRootCannotMove],
	// [ErrParentIsSelf], [ErrParentInSubtree], or [ErrParentNotFound] for the
	// four moves that cannot mean anything.
	MoveGroup(ctx context.Context, agencyID, id, newParentID string) (Group, error)

	// ListGroupChildren returns the direct children of a group (empty
	// parentID returns the roots).
	ListGroupChildren(ctx context.Context, agencyID, parentID string) ([]Group, error)

	// ListGroups returns every group in the agency, optionally filtered to
	// one hierarchy (empty hierarchyID returns every group).
	ListGroups(ctx context.Context, agencyID, hierarchyID string) ([]Group, error)

	// ListDiscoverableGroups returns groups flagged discoverable, optionally
	// filtered by a case-insensitive name substring (empty query returns all).
	ListDiscoverableGroups(ctx context.Context, agencyID, query string) ([]Group, error)

	// Register enrols a member at a group by writing (or updating) the
	// registered_at edge. Idempotent on (memberID, groupID): a second call
	// updates IsHome in place and preserves the original JoinedAt — mirrors
	// DEV-1319. Unlike the gateway's pre-DSN-1698 behaviour, registering a
	// new home group does NOT clear any other registration's home flag —
	// decision 8 drops that exclusivity outright.
	Register(ctx context.Context, agencyID string, r Registration) (Registration, error)

	// Deregister removes a member's registered_at edge to a group. Returns
	// the removed Registration and found=true if one existed; found=false
	// and a zero Registration if the member was not registered at the group
	// (idempotent no-op — mirrors the gateway store's Deregister semantics).
	// Unlike the gateway's Repository.Deregister, this does NOT write any
	// act-log row — the act log lives in the gateway's custody domain, which
	// this package deliberately does not depend on; composing the two is the
	// gateway adapter's job.
	Deregister(ctx context.Context, agencyID, memberID, groupID string) (Registration, bool, error)

	// ListGroupsForMember returns every group a member is registered at.
	ListGroupsForMember(ctx context.Context, agencyID, memberID string) ([]Registration, error)

	// ListMembersForGroup returns every member registered at a group.
	ListMembersForGroup(ctx context.Context, agencyID, groupID string) ([]Registration, error)

	// HomeCounts returns how many members call each group home, keyed by
	// group id. Groups with nobody are absent rather than zero.
	HomeCounts(ctx context.Context, agencyID string) (map[string]int, error)
}

// UserSchemaManager is a type alias for [entitygraph.SchemaManager]. Used to
// seed [DefaultUserSchema] on startup.
type UserSchemaManager = entitygraph.SchemaManager

// userManager is the concrete implementation of [UserManager].
type userManager struct {
	dm entitygraph.DataManager
}

// NewUserManager constructs a [UserManager] backed by the given
// [entitygraph.DataManager]. Returns an error if dm is nil.
func NewUserManager(dm entitygraph.DataManager) (UserManager, error) {
	if dm == nil {
		return nil, fmt.Errorf("NewUserManager: data manager must not be nil")
	}
	return &userManager{dm: dm}, nil
}
