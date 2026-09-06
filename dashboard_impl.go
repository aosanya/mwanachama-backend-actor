// dashboard_impl.go — GroupDashboard, the roles-with-members view at a group
// plus rollup counts. Ported from mwanachama-backend-api-gateway's
// internal/domain/dashboard package and
// internal/store/memory/dashboard_service.go's composition — DEV-1660 folds
// it into this repo as a read-model method now that role and the
// actor/group data it joins against are intra-repo, exactly as
// architecture-domain-decomposition.md predicted ("the join becomes
// intra-repo once they do").
//
// Unlike the gateway's old memory.DashboardService, this is not a separate
// composer type wired up beside UserManager — it is a method on UserManager
// itself, since every source it reads (ListGroupChildren, ListActorsForGroup,
// ListRoleKinds, ListRoleAssignmentsForGroup) already is one.
package mwanachamaactor

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

// GroupDashboard returns the composed view for a group: every role kind and
// who holds it there, alongside the group's member and child counts.
// Returns [ErrGroupNotFound] if the group does not exist.
func (m *userManager) GroupDashboard(ctx context.Context, groupID string) (models.GroupDashboard, error) {
	if _, err := m.GetGroup(ctx, groupID); err != nil {
		return models.GroupDashboard{}, err
	}
	kinds, err := m.ListRoleKinds(ctx)
	if err != nil {
		return models.GroupDashboard{}, fmt.Errorf("GroupDashboard: %w", err)
	}
	assignments, err := m.ListRoleAssignmentsForGroup(ctx, groupID, true)
	if err != nil {
		return models.GroupDashboard{}, fmt.Errorf("GroupDashboard: %w", err)
	}
	byKind := map[string][]string{}
	for _, a := range assignments {
		byKind[a.KindID] = append(byKind[a.KindID], a.ActorID)
	}
	roles := make([]models.RoleWithMembers, 0, len(kinds))
	for _, k := range kinds {
		// byKind is a map, so a kind nobody holds at this group reads back as
		// a nil slice — DEV-1077's rule, ported: nil marshals to `null`, not
		// `[]`, and a kind that exists but is held by nobody here is the
		// ordinary case, not an edge one.
		holders := byKind[k.ID]
		if holders == nil {
			holders = []string{}
		}
		roles = append(roles, models.RoleWithMembers{
			KindID:   k.ID,
			KindName: k.Name,
			Members:  holders,
		})
	}
	members, err := m.ListActorsForGroup(ctx, groupID)
	if err != nil {
		return models.GroupDashboard{}, fmt.Errorf("GroupDashboard: %w", err)
	}
	children, err := m.ListGroupChildren(ctx, groupID)
	if err != nil {
		return models.GroupDashboard{}, fmt.Errorf("GroupDashboard: %w", err)
	}
	return models.GroupDashboard{
		GroupID:     groupID,
		Roles:       roles,
		MemberCount: len(members),
		ChildCount:  len(children),
		ActiveRoles: len(assignments),
	}, nil
}
