package mwanachamaactor

import (
	"context"
	"errors"
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

func (m *userManager) WithdrawGroupPopulation(ctx context.Context, groupID, populationID, recordedBy string) (models.GroupPopulation, error) {
	all, err := m.ListGroupPopulations(ctx, groupID)
	if err != nil {
		return models.GroupPopulation{}, err
	}
	var target *models.GroupPopulation
	for i := range all {
		if all[i].ID == populationID && all[i].Withdraws == "" {
			target = &all[i]
		}
	}
	if target == nil || withdrawnIDs(all)[populationID] {
		return models.GroupPopulation{}, ErrPopulationNotFound
	}
	w := models.GroupPopulation{
		ID:         newSpecID(),
		GroupID:    groupID,
		Population: target.Population,
		AsOf:       target.AsOf,
		Source:     "withdrawn",
		RecordedBy: recordedBy,
		Withdraws:  populationID,
		CreatedAt:  NowRFC3339(),
	}
	if err := m.store.Insert(ctx, roleGroupPopulation, w); err != nil {
		return models.GroupPopulation{}, fmt.Errorf("WithdrawGroupPopulation: %w", err)
	}
	return w, nil
}

func withdrawnIDs(all []models.GroupPopulation) map[string]bool {
	out := map[string]bool{}
	for _, p := range all {
		if p.Withdraws != "" {
			out[p.Withdraws] = true
		}
	}
	return out
}

func standingFigures(all []models.GroupPopulation) []models.GroupPopulation {
	gone := withdrawnIDs(all)
	out := []models.GroupPopulation{}
	for _, p := range all {
		if p.Withdraws == "" && !gone[p.ID] {
			out = append(out, p)
		}
	}
	return out
}

func (m *userManager) LatestGroupPopulation(ctx context.Context, groupID string) (models.GroupPopulation, error) {
	if _, err := m.GetGroup(ctx, groupID); err != nil {
		return models.GroupPopulation{}, err
	}
	groups, err := m.ListGroups(ctx, "")
	if err != nil {
		return models.GroupPopulation{}, fmt.Errorf("LatestGroupPopulation: %w", err)
	}
	children := map[string][]string{}
	for _, g := range groups {
		if g.ParentID != "" {
			children[g.ParentID] = append(children[g.ParentID], g.ID)
		}
	}
	return m.latestOrInnerSum(ctx, groupID, children, map[string]bool{})
}

func (m *userManager) latestOrInnerSum(ctx context.Context, groupID string, children map[string][]string, seen map[string]bool) (models.GroupPopulation, error) {
	seen[groupID] = true
	all, err := m.ListGroupPopulations(ctx, groupID)
	if err != nil {
		return models.GroupPopulation{}, err
	}
	if own := standingFigures(all); len(own) > 0 {
		return own[len(own)-1], nil
	}
	sum := models.GroupPopulation{GroupID: groupID, Source: "inner groups"}
	found := false
	for _, child := range children[groupID] {
		if seen[child] {
			continue
		}
		part, err := m.latestOrInnerSum(ctx, child, children, seen)
		if errors.Is(err, ErrPopulationNotFound) {
			continue
		}
		if err != nil {
			return models.GroupPopulation{}, err
		}
		found = true
		sum.Population += part.Population
		if sum.AsOf == "" || part.AsOf < sum.AsOf {
			sum.AsOf = part.AsOf
		}
	}
	if !found {
		return models.GroupPopulation{}, ErrPopulationNotFound
	}
	return sum, nil
}
