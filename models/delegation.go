package models

type Delegation struct {
	ID         string `json:"id"`
	DelegateID string `json:"delegate_id"`
	SubjectID  string `json:"subject_id"`
	GrantedBy  string `json:"granted_by"`
	GrantedAt  string `json:"granted_at"`
	EndsAt     string `json:"ends_at"`
	RevokedBy  string `json:"revoked_by"`
	RevokedAt  string `json:"revoked_at"`
}

type DelegationGrant struct {
	DelegateID string `json:"delegate_id"`
	SubjectID  string `json:"subject_id"`
	EndsAt     string `json:"ends_at"`
}

func (d Delegation) LiveAt(now string) bool {
	return d.RevokedAt == "" && (d.EndsAt == "" || now < d.EndsAt)
}
