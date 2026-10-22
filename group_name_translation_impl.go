package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

var languageTag = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})*$`)

func normalLanguage(language string) (string, error) {
	language = strings.ToLower(strings.TrimSpace(language))
	if !languageTag.MatchString(language) {
		return "", ErrInvalidLanguage
	}
	return language, nil
}

func (m *userManager) findGroupNameTranslation(ctx context.Context, groupID, language string) (models.GroupNameTranslation, error) {
	var out models.GroupNameTranslation
	q := m.store.Query(ctx, roleGroupNameTranslation).Where("group_id = ? AND language = ?", groupID, language)
	if err := m.store.Take(q, roleGroupNameTranslation, &out, ErrTranslationNotFound); err != nil {
		return models.GroupNameTranslation{}, err
	}
	return out, nil
}

func (m *userManager) SetGroupNameTranslation(ctx context.Context, groupID, language, name string) (models.GroupNameTranslation, error) {
	language, err := normalLanguage(language)
	if err != nil {
		return models.GroupNameTranslation{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return models.GroupNameTranslation{}, ErrInvalidTranslation
	}
	if _, err := m.GetGroup(ctx, groupID); err != nil {
		return models.GroupNameTranslation{}, err
	}
	now := NowRFC3339()
	held, err := m.findGroupNameTranslation(ctx, groupID, language)
	if errors.Is(err, ErrTranslationNotFound) {
		t := models.GroupNameTranslation{
			ID:          newSpecID(),
			GroupID:     groupID,
			Language:    language,
			Name:        name,
			CreatedAt:   now,
			LastUpdated: now,
		}
		if err := m.store.Insert(ctx, roleGroupNameTranslation, t); err != nil {
			return models.GroupNameTranslation{}, fmt.Errorf("SetGroupNameTranslation: %w", err)
		}
		return t, nil
	}
	if err != nil {
		return models.GroupNameTranslation{}, fmt.Errorf("SetGroupNameTranslation: %w", err)
	}
	if held.Name == name {
		return held, nil
	}
	held.Name, held.Reviewed, held.ReviewedBy, held.ReviewedAt, held.LastUpdated = name, false, "", "", now
	err = m.store.Query(ctx, roleGroupNameTranslation).Where("id = ?", held.ID).
		Updates(map[string]any{"name": name, "reviewed": false, "reviewed_by": "", "reviewed_at": "", "last_updated": now}).Error
	if err != nil {
		return models.GroupNameTranslation{}, fmt.Errorf("SetGroupNameTranslation: %w", err)
	}
	return held, nil
}

func (m *userManager) ReviewGroupNameTranslation(ctx context.Context, groupID, language, reviewedBy string) (models.GroupNameTranslation, error) {
	language, err := normalLanguage(language)
	if err != nil {
		return models.GroupNameTranslation{}, err
	}
	if strings.TrimSpace(reviewedBy) == "" {
		return models.GroupNameTranslation{}, ErrReviewerRequired
	}
	held, err := m.findGroupNameTranslation(ctx, groupID, language)
	if err != nil {
		return models.GroupNameTranslation{}, err
	}
	now := NowRFC3339()
	held.Reviewed, held.ReviewedBy, held.ReviewedAt, held.LastUpdated = true, reviewedBy, now, now
	err = m.store.Query(ctx, roleGroupNameTranslation).Where("id = ?", held.ID).
		Updates(map[string]any{"reviewed": true, "reviewed_by": reviewedBy, "reviewed_at": now, "last_updated": now}).Error
	if err != nil {
		return models.GroupNameTranslation{}, fmt.Errorf("ReviewGroupNameTranslation: %w", err)
	}
	return held, nil
}

func (m *userManager) ListGroupNameTranslations(ctx context.Context, groupID string) ([]models.GroupNameTranslation, error) {
	if _, err := m.GetGroup(ctx, groupID); err != nil {
		return nil, err
	}
	q := m.store.Query(ctx, roleGroupNameTranslation).Where("group_id = ?", groupID)
	out, err := specstore.List[models.GroupNameTranslation](m.store, q, roleGroupNameTranslation)
	if err != nil {
		return nil, fmt.Errorf("ListGroupNameTranslations: %w", err)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Language < out[j].Language })
	return out, nil
}

func (m *userManager) GetGroupInLanguage(ctx context.Context, groupID, language string) (Group, error) {
	g, err := m.GetGroup(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	names, err := m.GroupNamesInLanguage(ctx, language, []string{g.ID})
	if err != nil {
		return Group{}, err
	}
	if name, ok := names[g.ID]; ok {
		g.Name = name
	}
	return g, nil
}

func (m *userManager) GroupNamesInLanguage(ctx context.Context, language string, groupIDs []string) (map[string]string, error) {
	out := map[string]string{}
	if strings.TrimSpace(language) == "" || len(groupIDs) == 0 {
		return out, nil
	}
	language, err := normalLanguage(language)
	if err != nil {
		return nil, err
	}
	q := m.store.Query(ctx, roleGroupNameTranslation).
		Where("language = ? AND reviewed AND group_id IN ?", language, groupIDs)
	held, err := specstore.List[models.GroupNameTranslation](m.store, q, roleGroupNameTranslation)
	if err != nil {
		return nil, fmt.Errorf("GroupNamesInLanguage: %w", err)
	}
	for _, t := range held {
		out[t.GroupID] = t.Name
	}
	return out, nil
}
