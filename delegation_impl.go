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

func (m *userManager) CreateDelegation(ctx context.Context, grantedBy string, grant models.DelegationGrant) (models.Delegation, error) {
	delegate := strings.TrimSpace(grant.DelegateID)
	subject := strings.TrimSpace(grant.SubjectID)
	if strings.TrimSpace(grantedBy) == "" || delegate == "" || subject == "" || delegate == subject {
		return models.Delegation{}, ErrInvalidDelegation
	}
	endsAt := ""
	if strings.TrimSpace(grant.EndsAt) != "" {
		at, err := time.Parse(time.RFC3339Nano, grant.EndsAt)
		if err != nil {
			return models.Delegation{}, ErrInvalidDelegation
		}
		endsAt = at.UTC().Format(models.TimeLayout)
	}
	for _, id := range []string{delegate, subject} {
		if _, err := m.GetActor(ctx, id); err != nil {
			return models.Delegation{}, err
		}
	}
	d := models.Delegation{
		ID:         newSpecID(),
		DelegateID: delegate,
		SubjectID:  subject,
		GrantedBy:  grantedBy,
		GrantedAt:  NowRFC3339(),
		EndsAt:     endsAt,
	}
	if err := m.store.Insert(ctx, roleDelegation, d); err != nil {
		return models.Delegation{}, fmt.Errorf("CreateDelegation: %w", err)
	}
	return d, nil
}

func (m *userManager) getDelegation(ctx context.Context, id string) (models.Delegation, error) {
	var out models.Delegation
	q := m.store.Query(ctx, roleDelegation).Where("id = ?", id)
	if err := m.store.Take(q, roleDelegation, &out, ErrDelegationNotFound); err != nil {
		if errors.Is(err, ErrDelegationNotFound) {
			return models.Delegation{}, ErrDelegationNotFound
		}
		return models.Delegation{}, fmt.Errorf("getDelegation: %w", err)
	}
	return out, nil
}

func (m *userManager) RevokeDelegation(ctx context.Context, delegationID, revokedBy string) (models.Delegation, error) {
	if strings.TrimSpace(revokedBy) == "" {
		return models.Delegation{}, ErrInvalidDelegation
	}
	d, err := m.getDelegation(ctx, delegationID)
	if err != nil {
		return models.Delegation{}, err
	}
	if d.RevokedAt != "" {
		return models.Delegation{}, ErrDelegationRevoked
	}
	now := NowRFC3339()
	res := m.store.Query(ctx, roleDelegation).
		Where("id = ? AND (revoked_at IS NULL OR revoked_at = '')", d.ID).
		Updates(map[string]any{"revoked_by": revokedBy, "revoked_at": now})
	if res.Error != nil {
		return models.Delegation{}, fmt.Errorf("RevokeDelegation: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return models.Delegation{}, ErrDelegationRevoked
	}
	d.RevokedBy, d.RevokedAt = revokedBy, now
	return d, nil
}

func (m *userManager) ListDelegations(ctx context.Context, actorID string) ([]models.Delegation, error) {
	q := m.store.Query(ctx, roleDelegation).Where("delegate_id = ? OR subject_id = ?", actorID, actorID)
	out, err := specstore.List[models.Delegation](m.store, q, roleDelegation)
	if err != nil {
		return nil, fmt.Errorf("ListDelegations: %w", err)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GrantedAt < out[j].GrantedAt })
	return out, nil
}

func (m *userManager) ActsFor(ctx context.Context, delegateID, subjectID string) (bool, error) {
	if delegateID == "" || subjectID == "" || delegateID == subjectID {
		return false, nil
	}
	q := m.store.Query(ctx, roleDelegation).Where("delegate_id = ? AND subject_id = ?", delegateID, subjectID)
	held, err := specstore.List[models.Delegation](m.store, q, roleDelegation)
	if err != nil {
		return false, fmt.Errorf("ActsFor: %w", err)
	}
	now := NowRFC3339()
	for _, d := range held {
		if d.LiveAt(now) {
			return true, nil
		}
	}
	return false, nil
}
