package mwanachamaactor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func validPopulationDate(asOf string) bool {
	_, err := time.Parse(time.DateOnly, asOf)
	return err == nil
}

func (m *userManager) RecordGroupPopulation(ctx context.Context, groupID string, population int, asOf, source, recordedBy string) (models.GroupPopulation, error) {
	asOf, source = strings.TrimSpace(asOf), strings.TrimSpace(source)
	if population < 0 || !validPopulationDate(asOf) || source == "" {
		return models.GroupPopulation{}, ErrInvalidPopulation
	}
	if _, err := m.GetGroup(ctx, groupID); err != nil {
		return models.GroupPopulation{}, err
	}
	p := models.GroupPopulation{
		ID:         newSpecID(),
		GroupID:    groupID,
		Population: population,
		AsOf:       asOf,
		Source:     source,
		RecordedBy: recordedBy,
		CreatedAt:  NowRFC3339(),
	}
	if err := m.store.Insert(ctx, roleGroupPopulation, p); err != nil {
		return models.GroupPopulation{}, fmt.Errorf("RecordGroupPopulation: %w", err)
	}
	return p, nil
}

func (m *userManager) ListGroupPopulations(ctx context.Context, groupID string) ([]models.GroupPopulation, error) {
	if _, err := m.GetGroup(ctx, groupID); err != nil {
		return nil, err
	}
	q := m.store.Query(ctx, roleGroupPopulation).Where("group_id = ?", groupID)
	out, err := specstore.List[models.GroupPopulation](m.store, q, roleGroupPopulation)
	if err != nil {
		return nil, fmt.Errorf("ListGroupPopulations: %w", err)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].AsOf != out[j].AsOf {
			return out[i].AsOf < out[j].AsOf
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	return out, nil
}

func (m *userManager) LatestGroupPopulation(ctx context.Context, groupID string) (models.GroupPopulation, error) {
	all, err := m.ListGroupPopulations(ctx, groupID)
	if err != nil {
		return models.GroupPopulation{}, err
	}
	if len(all) == 0 {
		return models.GroupPopulation{}, ErrPopulationNotFound
	}
	return all[len(all)-1], nil
}
