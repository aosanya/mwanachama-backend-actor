package mwanachamaactor

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func (m *userManager) GroupWardIndex(ctx context.Context) (*geo.WardIndex, map[string]Group, error) {
	all, err := m.ListGroups(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("GroupWardIndex: %w", err)
	}
	wards := []geo.Ward{}
	byID := map[string]Group{}
	for _, g := range all {
		if g.Boundary == nil {
			continue
		}
		wards = append(wards, geo.Ward{ID: g.ID, Name: g.Name, Boundary: *g.Boundary})
		byID[g.ID] = g
	}
	index, err := geo.NewWardIndex(wards)
	if err != nil {
		return nil, nil, fmt.Errorf("GroupWardIndex: %w", err)
	}
	return index, byID, nil
}

func (m *userManager) GroupsAtPoint(ctx context.Context, lon, lat float64) ([]Group, error) {
	if lon < -180 || lon > 180 || lat < -90 || lat > 90 {
		return nil, fmt.Errorf("%w: longitude %v, latitude %v is not on the earth", ErrInvalidGroup, lon, lat)
	}
	index, byID, err := m.GroupWardIndex(ctx)
	if err != nil {
		return nil, err
	}
	out := []Group{}
	for _, w := range index.WardsOf(geo.Position{lon, lat}) {
		out = append(out, byID[w.ID])
	}
	return out, nil
}
