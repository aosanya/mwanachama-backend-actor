package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) InviteToGroup(ctx context.Context, groupID, actorID, actorCode, invitedBy, expiresAt string) (GroupInvitation, error) {
	groupID, actorID, invitedBy = strings.TrimSpace(groupID), strings.TrimSpace(actorID), strings.TrimSpace(invitedBy)
	actorCode = normalizePublicCode(actorCode)
	if groupID == "" || invitedBy == "" {
		return GroupInvitation{}, fmt.Errorf("%w: a group, an actor and whoever invites are required", ErrInvalidInvitation)
	}
	if (actorID == "") == (actorCode == "") {
		return GroupInvitation{}, fmt.Errorf("%w: name the invitee by actor_id or by actor_code, exactly one of them", ErrInvalidInvitation)
	}
	issuedAt := time.Now().UTC()
	now := issuedAt.Format(TimeLayout)
	expires, err := invitationExpiry(expiresAt, now)
	if err != nil {
		return GroupInvitation{}, err
	}
	if expires == "" {
		expires = issuedAt.Add(m.invitationLifetime).Format(TimeLayout)
	}
	if _, err := m.GetGroup(ctx, groupID); err != nil {
		return GroupInvitation{}, err
	}
	if err := m.mustLead(m.db.WithContext(ctx), groupID, invitedBy); err != nil {
		if errors.Is(err, ErrNotGroupLeader) {
			return GroupInvitation{}, err
		}
		return GroupInvitation{}, fmt.Errorf("InviteToGroup: %w", err)
	}
	invitee, err := m.inviteeOf(ctx, actorID, actorCode)
	if err != nil {
		return GroupInvitation{}, err
	}
	actorID = invitee.ID
	_, found, err := m.findAssignment(ctx, actorID, groupID)
	if err != nil {
		return GroupInvitation{}, fmt.Errorf("InviteToGroup: %w", err)
	}
	if found {
		return GroupInvitation{}, ErrAlreadyInGroup
	}
	return m.issueInvitation(ctx, GroupInvitation{
		GroupID:     groupID,
		ActorID:     actorID,
		State:       models.InvitationPending,
		InvitedBy:   invitedBy,
		InvitedAt:   now,
		ExpiresAt:   expires,
		LastUpdated: now,
	}, true)
}

func (m *userManager) inviteeOf(ctx context.Context, actorID, actorCode string) (Actor, error) {
	if actorCode != "" {
		return m.getActorByCode(ctx, actorCode)
	}
	return m.GetActor(ctx, actorID)
}

func invitationExpiry(expiresAt, now string) (string, error) {
	expiresAt = strings.TrimSpace(expiresAt)
	if expiresAt == "" {
		return "", nil
	}
	at, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return "", fmt.Errorf("%w: expires_at must be an RFC 3339 time", ErrInvalidInvitation)
	}
	expires := at.UTC().Format(TimeLayout)
	if expires <= now {
		return "", fmt.Errorf("%w: expires_at must be in the future", ErrInvalidInvitation)
	}
	return expires, nil
}

func (m *userManager) issueInvitation(ctx context.Context, inv GroupInvitation, retryOnRace bool) (GroupInvitation, error) {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, found, err := m.findInvitation(ctx, tx, inv.GroupID, inv.ActorID)
		if err != nil {
			return err
		}
		if !found {
			inv.ID = newSpecID()
			row, err := encode(m.object(roleGroupInvitation), inv)
			if err != nil {
				return err
			}
			return tx.Table(m.table(roleGroupInvitation)).Create(row).Error
		}
		inv.ID = existing.ID
		return tx.Table(m.table(roleGroupInvitation)).Where("id = ?", inv.ID).
			Updates(map[string]any{
				"state":        inv.State,
				"invited_by":   inv.InvitedBy,
				"invited_at":   inv.InvitedAt,
				"expires_at":   inv.ExpiresAt,
				"answered_at":  "",
				"last_updated": inv.LastUpdated,
			}).Error
	})
	if err != nil {
		if retryOnRace && errors.Is(classifyDuplicateID(err), ErrDuplicateID) {
			return m.issueInvitation(ctx, inv, false)
		}
		return GroupInvitation{}, fmt.Errorf("InviteToGroup: %w", err)
	}
	return inv, nil
}

func (m *userManager) findInvitation(ctx context.Context, tx *gorm.DB, groupID, actorID string) (GroupInvitation, bool, error) {
	var out GroupInvitation
	q := tx.WithContext(ctx).Table(m.table(roleGroupInvitation)).
		Where("group_id = ? AND actor_id = ?", groupID, actorID)
	err := m.store.Take(q, roleGroupInvitation, &out, ErrInvitationNotFound)
	if errors.Is(err, ErrInvitationNotFound) {
		return GroupInvitation{}, false, nil
	}
	if err != nil {
		return GroupInvitation{}, false, err
	}
	return out, true, nil
}

func (m *userManager) GetInvitation(ctx context.Context, id string) (GroupInvitation, error) {
	var out GroupInvitation
	q := m.store.Query(ctx, roleGroupInvitation).Where("id = ?", id)
	if err := m.store.Take(q, roleGroupInvitation, &out, ErrInvitationNotFound); err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			return GroupInvitation{}, ErrInvitationNotFound
		}
		return GroupInvitation{}, fmt.Errorf("GetInvitation: %w", err)
	}
	return out, nil
}

func (m *userManager) ListInvitationsForGroup(ctx context.Context, groupID string, openOnly bool) ([]GroupInvitation, error) {
	q := m.store.Query(ctx, roleGroupInvitation).Where("group_id = ?", groupID)
	if openOnly {
		q = m.stillOpen(q, NowRFC3339())
	}
	return m.listInvitations(q, "ListInvitationsForGroup")
}

func (m *userManager) ListInvitationsForActor(ctx context.Context, actorID string) ([]GroupInvitation, error) {
	q := m.store.Query(ctx, roleGroupInvitation).Where("actor_id = ?", actorID)
	return m.listInvitations(q, "ListInvitationsForActor")
}

func (m *userManager) stillOpen(q *gorm.DB, now string) *gorm.DB {
	return q.Where("state = ?", models.InvitationPending).
		Where("("+m.store.Unset(roleGroupInvitation, "ExpiresAt")+" OR expires_at > ?)", now)
}

func (m *userManager) listInvitations(q *gorm.DB, op string) ([]GroupInvitation, error) {
	out, err := specstore.List[GroupInvitation](m.store, q, roleGroupInvitation)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].InvitedAt != out[j].InvitedAt {
			return out[i].InvitedAt < out[j].InvitedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *userManager) AcceptInvitation(ctx context.Context, id, actorID string) (GroupInvitation, error) {
	return m.answerInvitation(ctx, id, actorID, models.InvitationAccepted)
}

func (m *userManager) DeclineInvitation(ctx context.Context, id, actorID string) (GroupInvitation, error) {
	return m.answerInvitation(ctx, id, actorID, models.InvitationDeclined)
}

func (m *userManager) answerInvitation(ctx context.Context, id, actorID, state string) (GroupInvitation, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return GroupInvitation{}, fmt.Errorf("%w: an answer needs the invited actor", ErrInvalidInvitation)
	}
	inv, err := m.GetInvitation(ctx, id)
	if err != nil {
		return GroupInvitation{}, err
	}
	if inv.ActorID != actorID {
		return GroupInvitation{}, ErrInvitationNotFound
	}
	now := NowRFC3339()
	if inv.State != models.InvitationPending {
		return GroupInvitation{}, ErrInvitationAnswered
	}
	if inv.Lapsed(now) {
		return GroupInvitation{}, ErrInvitationLapsed
	}
	if state == models.InvitationAccepted {
		if _, err := m.GetGroup(ctx, inv.GroupID); err != nil {
			return GroupInvitation{}, err
		}
	}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		taken := m.stillOpen(tx.Table(m.table(roleGroupInvitation)).Where("id = ?", id), now).
			Updates(map[string]any{
				"state":        state,
				"answered_at":  now,
				"last_updated": now,
			})
		if taken.Error != nil {
			return taken.Error
		}
		if taken.RowsAffected == 0 {
			return ErrInvitationAnswered
		}
		if state != models.InvitationAccepted {
			return nil
		}
		if _, err := m.register(ctx, tx, inv.ActorID, inv.GroupID, now); err != nil {
			return err
		}
		return m.settlePendingJoinRequest(tx, inv.GroupID, inv.ActorID, now)
	})
	if errors.Is(err, ErrInvitationAnswered) {
		return GroupInvitation{}, ErrInvitationAnswered
	}
	if err != nil {
		return GroupInvitation{}, fmt.Errorf("answer invitation: %w", err)
	}
	inv.State = state
	inv.AnsweredAt = now
	inv.LastUpdated = now
	return inv, nil
}
