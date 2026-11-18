package mwanachamaactor

import (
	"context"
	"fmt"
	"strings"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) ListGroupMembers(ctx context.Context, groupID string, offset, limit int, search string) (models.GroupMemberPage, error) {
	limit = rosterPageLimit(limit)
	offset = max(offset, 0)
	tx := m.db.WithContext(ctx)
	inGroup := m.from(tx, roleActorGroupAssignment).Select("actor_id").
		Where("group_id = ?", groupID).
		Where("actor_id NOT IN (?)", m.outsiderActors(tx))
	named := m.from(tx, roleActor).Where("id IN (?)", inGroup)
	if search = strings.TrimSpace(search); search != "" {
		named = named.Where("id IN (?)", m.actorsNamedLike(tx, search))
	}
	var total int64
	if err := named.Count(&total).Error; err != nil {
		return models.GroupMemberPage{}, fmt.Errorf("ListGroupMembers: %w", err)
	}
	var rows []struct {
		ID          string
		DisplayName string
		FirstName   string
		LastName    string
	}
	err := named.Select("id", "display_name", "first_name", "last_name").
		Order("LOWER(display_name), LOWER(first_name), LOWER(last_name), id").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return models.GroupMemberPage{}, fmt.Errorf("ListGroupMembers: %w", err)
	}
	items := make([]models.GroupMember, len(rows))
	for i, r := range rows {
		items[i] = models.GroupMember{ActorID: r.ID, DisplayName: memberName(r.DisplayName, r.FirstName, r.LastName)}
	}
	return models.GroupMemberPage{Items: items, Total: int(total)}, nil
}

func memberName(display, first, last string) string {
	if display = strings.TrimSpace(display); display != "" {
		return display
	}
	return strings.TrimSpace(first + " " + last)
}
