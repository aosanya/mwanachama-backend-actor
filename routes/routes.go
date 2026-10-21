package routes

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

type Route = httpwire.Route

const maxGroupName = 200

var operations = sync.OnceValues(func() (*dispatch.Spec, error) {
	return dispatch.Parse(mwanachamaactor.Operations())
})

var sentinels = map[string]error{
	"ErrActorNotFound":          mwanachamaactor.ErrActorNotFound,
	"ErrGroupNotFound":          mwanachamaactor.ErrGroupNotFound,
	"ErrRoleKindNotFound":       mwanachamaactor.ErrRoleKindNotFound,
	"ErrAssignmentNotFound":     mwanachamaactor.ErrAssignmentNotFound,
	"ErrHierarchyNotFound":      mwanachamaactor.ErrHierarchyNotFound,
	"ErrLevelNotFound":          mwanachamaactor.ErrLevelNotFound,
	"ErrGroupTypeNotFound":      mwanachamaactor.ErrGroupTypeNotFound,
	"ErrInvalidActor":           mwanachamaactor.ErrInvalidActor,
	"ErrInvalidGroup":           mwanachamaactor.ErrInvalidGroup,
	"ErrInvalidGroupType":       mwanachamaactor.ErrInvalidGroupType,
	"ErrParentNotFound":         mwanachamaactor.ErrParentNotFound,
	"ErrDuplicateAttribute":     mwanachamaactor.ErrDuplicateAttribute,
	"ErrDuplicateID":            mwanachamaactor.ErrDuplicateID,
	"ErrRootCannotMove":         mwanachamaactor.ErrRootCannotMove,
	"ErrParentIsSelf":           mwanachamaactor.ErrParentIsSelf,
	"ErrParentInSubtree":        mwanachamaactor.ErrParentInSubtree,
	"ErrKindHasLiveAssignments": mwanachamaactor.ErrKindHasLiveAssignments,
	"ErrKindRetired":            mwanachamaactor.ErrKindRetired,
	"ErrLevelWornByGroups":      mwanachamaactor.ErrLevelWornByGroups,
	"ErrLevelHasRoleKinds":      mwanachamaactor.ErrLevelHasRoleKinds,
	"ErrGroupTypeWornByGroups":  mwanachamaactor.ErrGroupTypeWornByGroups,
	"ErrDuplicateDefaultAnchor": mwanachamaactor.ErrDuplicateDefaultAnchor,
	"ErrInvalidJoinMode":        mwanachamaactor.ErrInvalidJoinMode,
	"ErrInvalidJoinRequest":     mwanachamaactor.ErrInvalidJoinRequest,
	"ErrJoinMessageTooLong":     mwanachamaactor.ErrJoinMessageTooLong,
	"ErrDeclineReasonTooLong":   mwanachamaactor.ErrDeclineReasonTooLong,
	"ErrQuietDeclineReason":     mwanachamaactor.ErrQuietDeclineReason,
	"ErrJoinRequestNotFound":    mwanachamaactor.ErrJoinRequestNotFound,
	"ErrJoinDecisionNotFound":   mwanachamaactor.ErrJoinDecisionNotFound,
	"ErrJoinRequestDecided":     mwanachamaactor.ErrJoinRequestDecided,
	"ErrInvalidInvitation":      mwanachamaactor.ErrInvalidInvitation,
	"ErrInvitationNotFound":     mwanachamaactor.ErrInvitationNotFound,
	"ErrInvitationAnswered":     mwanachamaactor.ErrInvitationAnswered,
	"ErrInvitationLapsed":       mwanachamaactor.ErrInvitationLapsed,
	"ErrAlreadyInGroup":         mwanachamaactor.ErrAlreadyInGroup,
	"ErrNotGroupLeader":         mwanachamaactor.ErrNotGroupLeader,
	"ErrInvalidCodeFormat":      mwanachamaactor.ErrInvalidCodeFormat,
}

type Mount struct {
	Authorize dispatch.Authorizer
	Caller    dispatch.Caller

	CheckReferences bool
}

type referenceChecked struct {
	mwanachamaactor.UserManager
}

func (m referenceChecked) CreateGroup(ctx context.Context, g mwanachamaactor.Group) (mwanachamaactor.Group, error) {
	if err := m.checkReferences(ctx, g.HierarchyID, g.LevelID, g.AnchorLevelOverrideID); err != nil {
		return mwanachamaactor.Group{}, err
	}
	if g.ParentID != "" {
		if _, err := m.UserManager.GetGroup(ctx, g.ParentID); err != nil {
			return mwanachamaactor.Group{}, parentReferenceErr(err)
		}
	}
	return m.UserManager.CreateGroup(ctx, g)
}

func (m referenceChecked) EditGroup(ctx context.Context, id string, e mwanachamaactor.GroupEdit) (mwanachamaactor.Group, error) {
	e.Name = strings.TrimSpace(e.Name)
	e.NodeType = strings.TrimSpace(e.NodeType)
	if e.Name == "" {
		return mwanachamaactor.Group{}, fmt.Errorf("%w: a name is required", mwanachamaactor.ErrInvalidGroup)
	}
	if len([]rune(e.Name)) > maxGroupName {
		return mwanachamaactor.Group{}, fmt.Errorf("%w: a name may be at most %d characters", mwanachamaactor.ErrInvalidGroup, maxGroupName)
	}
	if e.AnchorLevelOverrideID != "" {
		current, err := m.UserManager.GetGroup(ctx, id)
		if err != nil {
			return mwanachamaactor.Group{}, err
		}
		if err := m.checkReferences(ctx, current.HierarchyID, "", e.AnchorLevelOverrideID); err != nil {
			return mwanachamaactor.Group{}, err
		}
	}
	return m.UserManager.EditGroup(ctx, id, e)
}

func (m referenceChecked) checkReferences(ctx context.Context, hierarchyID, levelID, anchorLevelOverrideID string) error {
	if hierarchyID != "" {
		exists, err := m.UserManager.HierarchyExists(ctx, hierarchyID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: hierarchy %s does not exist", mwanachamaactor.ErrInvalidGroup, hierarchyID)
		}
	}
	for _, id := range []string{levelID, anchorLevelOverrideID} {
		if id == "" {
			continue
		}
		inHierarchy, err := m.UserManager.LevelInHierarchy(ctx, id, hierarchyID)
		if err != nil {
			return err
		}
		if !inHierarchy {
			return fmt.Errorf("%w: level %s is not in that hierarchy", mwanachamaactor.ErrInvalidGroup, id)
		}
	}
	return nil
}

func parentReferenceErr(err error) error {
	if err == mwanachamaactor.ErrGroupNotFound {
		return mwanachamaactor.ErrParentNotFound
	}
	return err
}

func managerFor(um mwanachamaactor.UserManager, m Mount) any {
	if m.CheckReferences {
		return referenceChecked{UserManager: um}
	}
	return um
}

func Build(um mwanachamaactor.UserManager) ([]Route, error) {
	return BuildFor(um, Mount{CheckReferences: true})
}

func BuildFor(um mwanachamaactor.UserManager, m Mount) ([]Route, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{
		Manager:   managerFor(um, m),
		Errors:    sentinels,
		Authorize: m.Authorize,
		Caller:    m.Caller,
	})
}

func Routes(um mwanachamaactor.UserManager) []Route {
	return RoutesFor(um, Mount{CheckReferences: true})
}

func RoutesFor(um mwanachamaactor.UserManager, m Mount) []Route {
	out, err := BuildFor(um, m)
	if err != nil {
		panic(fmt.Sprintf("actor routes: %v", err))
	}
	return out
}

func Shape() []Route {
	s, err := operations()
	if err != nil {
		panic(fmt.Sprintf("actor routes: %v", err))
	}
	names := make([]string, 0, len(s.Operations))
	for name := range s.Operations {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Route, 0, len(names))
	for _, name := range names {
		op := s.Operations[name]
		out = append(out, Route{Method: op.Method, Path: s.Base + op.Path, Action: op.Action})
	}
	return out
}
