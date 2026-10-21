package mwanachamaactor

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

type (
	Actor                = models.Actor
	Group                = models.Group
	MyGroup              = models.MyGroup
	FormsPreference      = models.FormsPreference
	GroupEdit            = models.GroupEdit
	ActorGroupAssignment = models.ActorGroupAssignment
	RoleKind             = models.RoleKind
	ActorRoleAssignment  = models.ActorRoleAssignment
	Hierarchy            = models.Hierarchy
	Level                = models.Level
	GroupType            = models.GroupType
	GroupJoinRequest     = models.GroupJoinRequest
	JoinDecision         = models.JoinDecision
	JoinOutcome          = models.JoinOutcome
	GroupByCode          = models.GroupByCode
	GroupInvitation      = models.GroupInvitation
	InvitationBatchRow   = models.InvitationBatchRow
	InvitationBatchCheck = models.InvitationBatchCheck
	CodeFormat           = models.CodeFormat
)

const TimeLayout = models.TimeLayout

const (
	EndedByRevocation  = models.EndedByRevocation
	EndedByResignation = models.EndedByResignation
	EndedByEviction    = models.EndedByEviction
	EndedByDeparture   = models.EndedByDeparture
)

func NowRFC3339() string { return models.NowRFC3339() }

type UserManager interface {
	CreateActor(ctx context.Context, a Actor) (Actor, error)

	EnsureActor(ctx context.Context, id string) (Actor, error)

	GetActor(ctx context.Context, id string) (Actor, error)

	GetActors(ctx context.Context, ids []string) ([]Actor, error)

	GetActorByPhone(ctx context.Context, phone string) (Actor, error)

	SetActorDisplayName(ctx context.Context, id, displayName string) (Actor, error)

	SetActorName(ctx context.Context, id, firstName, lastName string) (Actor, error)

	ListActors(ctx context.Context) ([]Actor, error)

	CreateGroup(ctx context.Context, g Group) (Group, error)

	GetGroup(ctx context.Context, id string) (Group, error)

	EditGroup(ctx context.Context, id string, e GroupEdit) (Group, error)

	MoveGroup(ctx context.Context, id, newParentID string) (Group, error)

	ListGroupChildren(ctx context.Context, parentID string) ([]Group, error)

	ListGroups(ctx context.Context, hierarchyID string) ([]Group, error)

	ListDiscoverableGroups(ctx context.Context, query string) ([]Group, error)

	AssignGroup(ctx context.Context, r ActorGroupAssignment) (ActorGroupAssignment, error)

	Deregister(ctx context.Context, actorID, groupID string) (ActorGroupAssignment, bool, error)
	RemoveFromGroup(ctx context.Context, actorID, groupID, removedBy string) (models.GroupRemoval, error)
	RemoveFromAllGroups(ctx context.Context, actorID, removedBy string) ([]models.GroupRemoval, error)
	ListRemovals(ctx context.Context, actorID string) ([]models.GroupRemoval, error)

	ListGroupsForActor(ctx context.Context, actorID string) ([]ActorGroupAssignment, error)

	ListMyGroups(ctx context.Context, actorID string) ([]MyGroup, error)

	SetMyGroupPriority(ctx context.Context, groupID, actorID, priority string) (ActorGroupAssignment, error)

	GetMyFormsWeekSplit(ctx context.Context, actorID string) (FormsPreference, error)

	SetMyFormsWeekSplit(ctx context.Context, actorID string, formsWeekSplit bool) (FormsPreference, error)

	ListActorsForGroup(ctx context.Context, groupID string) ([]ActorGroupAssignment, error)

	HomeCounts(ctx context.Context) (map[string]int, error)

	SetGroupJoinMode(ctx context.Context, groupID, mode string) (Group, error)
	SetGroupHidden(ctx context.Context, groupID string, hidden bool) (Group, error)
	ListJoinableGroups(ctx context.Context, actorID string) ([]Group, error)
	JoinGroup(ctx context.Context, groupID, actorID, message string) (JoinOutcome, error)
	FindGroupByCode(ctx context.Context, code, actorID string) (GroupByCode, error)
	CodeFormat(ctx context.Context) (CodeFormat, error)
	SetCodeFormat(ctx context.Context, groupCount int) (CodeFormat, error)
	JoinGroupByCode(ctx context.Context, code, actorID, message string) (JoinOutcome, error)
	GetJoinRequest(ctx context.Context, id string) (GroupJoinRequest, error)
	ListJoinRequestsForGroup(ctx context.Context, groupID string, pendingOnly bool) ([]GroupJoinRequest, error)
	ListJoinRequestsForActor(ctx context.Context, actorID string) ([]GroupJoinRequest, error)
	ApproveJoinRequest(ctx context.Context, id, decidedBy string) (GroupJoinRequest, error)
	DeclineJoinRequest(ctx context.Context, id, decidedBy string, silent bool, reason string) (GroupJoinRequest, error)
	GetMyJoinDecision(ctx context.Context, decisionID, actorID string) (JoinDecision, error)
	ListJoinDecisions(ctx context.Context, requestID string) ([]JoinDecision, error)
	IsGroupLeader(ctx context.Context, groupID, actorID string) (bool, error)
	ListLedGroups(ctx context.Context, actorID string) ([]Group, error)

	InviteToGroup(ctx context.Context, groupID, actorID, actorCode, invitedBy, expiresAt string) (GroupInvitation, error)
	GetInvitation(ctx context.Context, id string) (GroupInvitation, error)
	ListInvitationsForGroup(ctx context.Context, groupID string, openOnly bool) ([]GroupInvitation, error)
	ListInvitationsForActor(ctx context.Context, actorID string) ([]GroupInvitation, error)
	AcceptInvitation(ctx context.Context, id, actorID string) (GroupInvitation, error)
	DeclineInvitation(ctx context.Context, id, actorID string) (GroupInvitation, error)
	CheckInvitationBatch(ctx context.Context, rows []InvitationBatchRow) ([]InvitationBatchCheck, error)

	CreateRoleKind(ctx context.Context, k RoleKind) (RoleKind, error)
	ListRoleKinds(ctx context.Context) ([]RoleKind, error)
	GetRoleKind(ctx context.Context, id string) (RoleKind, error)
	RetireRoleKind(ctx context.Context, kindID, actorID string) (RoleKind, error)
	UnretireRoleKind(ctx context.Context, kindID string) (RoleKind, error)
	SetRoleKindLeadsGroup(ctx context.Context, kindID string, leads bool) (RoleKind, error)

	GrantRole(ctx context.Context, a ActorRoleAssignment) (ActorRoleAssignment, error)
	GetRoleAssignment(ctx context.Context, id string) (ActorRoleAssignment, error)
	RevokeRole(ctx context.Context, assignmentID string) error
	StepDownRole(ctx context.Context, assignmentID string) error
	EndRoleOnEviction(ctx context.Context, assignmentID string) error
	EndRoleOnDeparture(ctx context.Context, assignmentID string) error
	ListRoleAssignmentsForGroup(ctx context.Context, groupID string, activeOnly bool) ([]ActorRoleAssignment, error)
	ListRoleAssignmentsForActor(ctx context.Context, actorID string, activeOnly bool) ([]ActorRoleAssignment, error)
	ListRoleHolders(ctx context.Context, kindID string, activeOnly bool) ([]models.RoleHolder, error)

	GroupDashboard(ctx context.Context, groupID string) (models.GroupDashboard, error)

	CreateHierarchy(ctx context.Context, h Hierarchy) (Hierarchy, error)
	ListHierarchies(ctx context.Context) ([]Hierarchy, error)
	GetHierarchy(ctx context.Context, id string) (Hierarchy, error)
	RenameHierarchy(ctx context.Context, id, name string) (Hierarchy, error)

	CreateLevel(ctx context.Context, l Level) (Level, error)
	GetLevel(ctx context.Context, id string) (Level, error)
	RenameLevel(ctx context.Context, id, name string) (Level, error)
	DeleteLevel(ctx context.Context, id string) error
	SetDefaultAnchor(ctx context.Context, hierarchyID, levelID string) error
	ListLevels(ctx context.Context, hierarchyID string) ([]Level, error)

	CreateGroupType(ctx context.Context, g GroupType) (GroupType, error)
	GetGroupType(ctx context.Context, id string) (GroupType, error)
	EditGroupType(ctx context.Context, id, name, singular, plural, levelID string) (GroupType, error)
	DeleteGroupType(ctx context.Context, id string) error
	ListGroupTypes(ctx context.Context, hierarchyID string) ([]GroupType, error)

	HierarchyExists(ctx context.Context, id string) (bool, error)
	LevelInHierarchy(ctx context.Context, levelID, hierarchyID string) (bool, error)
}

type userManager struct {
	db                 *gorm.DB
	store              *store
	invitationLifetime time.Duration
}

const DefaultInvitationLifetime = 14 * 24 * time.Hour

type Option func(*userManager)

func WithInvitationLifetime(d time.Duration) Option {
	return func(m *userManager) { m.invitationLifetime = d }
}

func NewUserManager(db *gorm.DB, s *spec.Spec, opts ...Option) (UserManager, error) {
	if db == nil {
		return nil, fmt.Errorf("NewUserManager: db must not be nil")
	}
	st, err := newStore(db, s)
	if err != nil {
		return nil, err
	}
	m := &userManager{db: db, store: st, invitationLifetime: DefaultInvitationLifetime}
	for _, opt := range opts {
		opt(m)
	}
	if m.invitationLifetime <= 0 {
		return nil, fmt.Errorf("NewUserManager: an invitation lifetime must be positive, got %s", m.invitationLifetime)
	}
	return m, nil
}

func (m *userManager) table(role string) string { return m.store.Table(role) }

func (m *userManager) object(role string) spec.Object { return m.store.Object(role) }
