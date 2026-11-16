package mwanachamaactor

import (
	"context"
	"fmt"
	"strings"

	"github.com/aosanya/mwanachama-backend-shared/plaintext"
)

const roleNamePartSeparator = "/"

func (m *userManager) checkDisplayNameFree(ctx context.Context, name, exceptID string) error {
	key := plaintext.FoldLookalike(name)
	if key == "" {
		return nil
	}
	actors, err := m.ListActors(ctx)
	if err != nil {
		return fmt.Errorf("checkDisplayNameFree: %w", err)
	}
	for _, other := range actors {
		if other.ID != exceptID && plaintext.FoldLookalike(other.DisplayName) == key {
			return fmt.Errorf("%w: %q", ErrDuplicateDisplayName, name)
		}
	}
	kinds, err := m.ListRoleKinds(ctx)
	if err != nil {
		return fmt.Errorf("checkDisplayNameFree: %w", err)
	}
	for _, kind := range kinds {
		if kind.RetiredAt != "" {
			continue
		}
		for _, part := range append([]string{kind.Name}, strings.Split(kind.Name, roleNamePartSeparator)...) {
			if plaintext.FoldLookalike(part) == key {
				return fmt.Errorf("%w: %q", ErrDuplicateDisplayName, name)
			}
		}
	}
	return nil
}
