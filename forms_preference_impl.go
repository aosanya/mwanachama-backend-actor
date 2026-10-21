package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (m *userManager) GetMyFormsWeekSplit(ctx context.Context, actorID string) (FormsPreference, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return FormsPreference{FormsWeekSplit: true}, nil
	}
	held, err := m.GetActor(ctx, actorID)
	if errors.Is(err, ErrActorNotFound) {
		return FormsPreference{ActorID: actorID, FormsWeekSplit: true}, nil
	}
	if err != nil {
		return FormsPreference{}, fmt.Errorf("GetMyFormsWeekSplit: %w", err)
	}
	return FormsPreference{ActorID: held.ID, FormsWeekSplit: held.FormsWeekSplit}, nil
}

func (m *userManager) SetMyFormsWeekSplit(ctx context.Context, actorID string, formsWeekSplit bool) (FormsPreference, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return FormsPreference{}, ErrActorNotFound
	}
	if _, err := m.GetActor(ctx, actorID); err != nil {
		return FormsPreference{}, err
	}
	err := m.store.Query(ctx, roleActor).Where("id = ?", actorID).
		Updates(map[string]any{"forms_week_split": formsWeekSplit, "last_updated": NowRFC3339()}).Error
	if err != nil {
		return FormsPreference{}, fmt.Errorf("SetMyFormsWeekSplit: %w", err)
	}
	return FormsPreference{ActorID: actorID, FormsWeekSplit: formsWeekSplit}, nil
}
