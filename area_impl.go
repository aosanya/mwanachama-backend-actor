package mwanachamaactor

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func outlineShaped(outline geo.Geometry) bool {
	switch outline.Kind {
	case "Polygon", "MultiPolygon":
		return outline.Validate() == nil
	}
	return false
}

func (m *userManager) checkOutline(ctx context.Context, levelID string, outline geo.Geometry) error {
	if outline.Kind == "" {
		return nil
	}
	if !outlineShaped(outline) {
		return ErrInvalidOutline
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

func (m *userManager) SetGroupOutline(ctx context.Context, groupID string, outline geo.Geometry) (Group, error) {
	current, err := m.GetGroup(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	if err := m.checkOutline(ctx, current.LevelID, outline); err != nil {
		return Group{}, err
	}
	current.Outline = outline
	current.LastUpdated = NowRFC3339()
	row, err := encode(m.object(roleGroup), current)
	if err != nil {
		return Group{}, fmt.Errorf("SetGroupOutline: %w", err)
	}
	if err := m.store.Query(ctx, roleGroup).Where("id = ?", groupID).
		Updates(map[string]any{"outline": row["outline"], "last_updated": current.LastUpdated}).Error; err != nil {
		return Group{}, fmt.Errorf("SetGroupOutline: %w", err)
	}
	return current, nil
}

func (m *userManager) ResolvePoint(ctx context.Context, hierarchyID string, lon, lat float64) (Group, bool, error) {
	levels, err := m.ListLevels(ctx, hierarchyID)
	if err != nil {
		return Group{}, false, err
	}
	areas := map[string]bool{}
	for _, l := range levels {
		if l.IsArea {
			areas[l.ID] = true
		}
	}
	if len(areas) == 0 {
		return Group{}, false, nil
	}
	groups, err := m.ListGroups(ctx, hierarchyID)
	if err != nil {
		return Group{}, false, err
	}
	pin := geo.Position{lon, lat}
	var found Group
	var foundDepth = -1
	depth := map[string]int{}
	for _, l := range levels {
		depth[l.ID] = l.Depth
	}
	for _, g := range groups {
		if !areas[g.LevelID] || g.Outline.Kind == "" || !geo.Contains(g.Outline, pin) {
			continue
		}
		if depth[g.LevelID] > foundDepth {
			found, foundDepth = g, depth[g.LevelID]
		}
	}
	return found, foundDepth >= 0, nil
}
