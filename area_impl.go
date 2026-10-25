package mwanachamaactor

import (
	"context"
	"fmt"
)

func (m *userManager) checkAreaNode(ctx context.Context, levelID string, areaNodeID string) error {
	if areaNodeID == "" {
		return nil
	}
	level, err := m.GetLevel(ctx, levelID)
	if err != nil {
		return err
	}
	if !level.IsArea {
		return ErrNotAnArea
	}
	return nil
}

func (m *userManager) SetGroupAreaNode(ctx context.Context, groupID string, areaNodeID string) (Group, error) {
	current, err := m.GetGroup(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	if err := m.checkAreaNode(ctx, current.LevelID, areaNodeID); err != nil {
		return Group{}, err
	}
	current.AreaNodeID = areaNodeID
	current.LastUpdated = NowRFC3339()
	if err := m.store.Query(ctx, roleGroup).Where("id = ?", groupID).
		Updates(map[string]any{"area_node_id": areaNodeID, "last_updated": current.LastUpdated}).Error; err != nil {
		return Group{}, fmt.Errorf("SetGroupAreaNode: %w", err)
	}
	return current, nil
}
