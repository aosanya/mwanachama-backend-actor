package mwanachamaactor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func checkBoundary(b *geo.Geometry) error {
	if b == nil {
		return nil
	}
	if b.Kind != geo.KindPolygon && b.Kind != geo.KindMultiPolygon {
		return fmt.Errorf("%w: got %q", ErrBoundaryNotAnArea, b.Kind)
	}
	return b.Validate()
}

func (m *userManager) SetGroupBoundary(ctx context.Context, groupID string, boundary *geo.Geometry) (Group, error) {
	current, err := m.GetGroup(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	if err := checkBoundary(boundary); err != nil {
		return Group{}, err
	}
	var stored any
	if boundary != nil {
		raw, err := json.Marshal(boundary)
		if err != nil {
			return Group{}, fmt.Errorf("SetGroupBoundary: %w", err)
		}
		stored = string(raw)
	}
	current.Boundary = boundary
	current.LastUpdated = NowRFC3339()
	if err := m.store.Query(ctx, roleGroup).Where("id = ?", groupID).
		Updates(map[string]any{"boundary": stored, "last_updated": current.LastUpdated}).Error; err != nil {
		return Group{}, fmt.Errorf("SetGroupBoundary: %w", err)
	}
	return current, nil
}
