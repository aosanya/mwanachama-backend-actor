package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

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
	actorID, found, err := m.actorIDByEmail(ctx, r.Email)
	if err != nil || !found {
		return models.BatchRowReady, err
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
	err := m.db.WithContext(ctx).Table(m.table(roleActor)).
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
