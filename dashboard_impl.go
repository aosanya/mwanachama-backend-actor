package mwanachamaactor

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

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
