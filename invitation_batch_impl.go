package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"gorm.io/datatypes"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) CheckInvitationBatch(ctx context.Context, rows []InvitationBatchRow) ([]InvitationBatchCheck, error) {
	out := make([]InvitationBatchCheck, len(rows))
	seen := map[string]bool{}
	for i, r := range rows {
		r = normalizeBatchRow(r)
		check := InvitationBatchCheck{Row: i + 1, Input: r}
		status, err := m.checkBatchRow(ctx, r, seen, &check)
		if err != nil {
			return nil, fmt.Errorf("CheckInvitationBatch: %w", err)
		}
		check.Status = status
		out[i] = check
	}
	return out, nil
}

func normalizeBatchRow(r InvitationBatchRow) InvitationBatchRow {
	return InvitationBatchRow{
		FirstName: strings.TrimSpace(r.FirstName),
		LastName:  strings.TrimSpace(r.LastName),
		Email:     strings.ToLower(strings.TrimSpace(r.Email)),
		GroupCode: normalizePublicCode(r.GroupCode),
		ActorID:   strings.TrimSpace(r.ActorID),
	}
}

func (m *userManager) checkBatchRow(ctx context.Context, r InvitationBatchRow, seen map[string]bool, check *InvitationBatchCheck) (string, error) {
	if r.FirstName == "" || r.LastName == "" {
		return models.BatchRowMissingName, nil
	}
	if !plainEmail(r.Email) {
		return models.BatchRowBadEmail, nil
	}
	group, err := m.groupByCode(ctx, r.GroupCode)
	if errors.Is(err, ErrGroupNotFound) {
		return models.BatchRowUnknownGroup, nil
	}
	if err != nil {
		return "", err
	}
	check.GroupID = group.ID
	key := r.Email + "\x00" + group.ID
	if seen[key] {
		return models.BatchRowDuplicate, nil
	}
	seen[key] = true
	actorID, found := r.ActorID, r.ActorID != ""
	if !found {
		var err error
		actorID, found, err = m.actorIDByEmail(ctx, r.Email)
		if err != nil || !found {
			return models.BatchRowReady, err
		}
	}
	check.ActorID = actorID
	_, member, err := m.findAssignment(ctx, actorID, group.ID)
	if err != nil {
		return "", err
	}
	if member {
		return models.BatchRowAlreadyMember, nil
	}
	return models.BatchRowReady, nil
}

func plainEmail(s string) bool {
	if s == "" || strings.ContainsAny(s, " <>") {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	at := strings.LastIndex(s, "@")
	return at > 0 && strings.Contains(s[at+1:], ".")
}

func (m *userManager) actorIDByEmail(ctx context.Context, email string) (string, bool, error) {
	var ids []string
	err := m.from(m.db.WithContext(ctx), roleActor).
		Where(datatypes.JSONQuery("attributes").Equals(email, "email")).
		Limit(1).Pluck("id", &ids).Error
	if err != nil {
		return "", false, err
	}
	if len(ids) == 0 {
		return "", false, nil
	}
	return ids[0], true, nil
}

func (m *userManager) SendInvitationBatch(ctx context.Context, rows []InvitationBatchRow, invitedBy string) ([]InvitationBatchResult, error) {
	invitedBy = strings.TrimSpace(invitedBy)
	if invitedBy == "" {
		return nil, fmt.Errorf("%w: whoever sends the invitations is required", ErrInvalidInvitation)
	}
	checks, err := m.CheckInvitationBatch(ctx, rows)
	if err != nil {
		return nil, fmt.Errorf("SendInvitationBatch: %w", err)
	}
	out := make([]InvitationBatchResult, len(checks))
	for i, check := range checks {
		out[i] = InvitationBatchResult{InvitationBatchCheck: check}
		if check.Status != models.BatchRowReady {
			continue
		}
		if err := m.sendBatchRow(ctx, invitedBy, &out[i]); err != nil {
			return nil, fmt.Errorf("SendInvitationBatch: row %d: %w", check.Row, err)
		}
	}
	return out, nil
}

func (m *userManager) sendBatchRow(ctx context.Context, invitedBy string, res *InvitationBatchResult) error {
	if res.ActorID == "" {
		res.Status = models.BatchRowNoActor
		return nil
	}
	if _, err := m.GetActor(ctx, res.ActorID); err != nil {
		if errors.Is(err, ErrActorNotFound) {
			res.Status = models.BatchRowNoActor
			return nil
		}
		return err
	}
	if err := m.refuseOutsider(m.db.WithContext(ctx), res.ActorID); err != nil {
		if errors.Is(err, ErrOutsiderRole) {
			res.Status = models.BatchRowOutsider
			return nil
		}
		return err
	}
	existing, found, err := m.findInvitation(ctx, m.db, res.GroupID, res.ActorID)
	if err != nil {
		return err
	}
	if found && existing.State == models.InvitationPending && (existing.ExpiresAt == "" || existing.ExpiresAt > NowRFC3339()) {
		res.Status = models.BatchRowSent
		res.Invitation = &existing
		return nil
	}
	issuedAt := time.Now().UTC()
	now := issuedAt.Format(TimeLayout)
	inv, err := m.issueInvitation(ctx, GroupInvitation{
		GroupID:     res.GroupID,
		ActorID:     res.ActorID,
		State:       models.InvitationPending,
		InvitedBy:   invitedBy,
		InvitedAt:   now,
		ExpiresAt:   issuedAt.Add(m.invitationLifetime).Format(TimeLayout),
		LastUpdated: now,
	}, true)
	if err != nil {
		return err
	}
	res.Status = models.BatchRowSent
	res.Invitation = &inv
	return nil
}
