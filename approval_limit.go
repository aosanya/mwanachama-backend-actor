package mwanachamaactor

func WithinLimit(a ActorRoleAssignment, amount int) bool {
	if !a.Active || amount < 0 {
		return false
	}
	return a.ApprovalLimit == nil || amount <= *a.ApprovalLimit
}

func CheckLimit(a ActorRoleAssignment, amount int) error {
	if !WithinLimit(a, amount) {
		return ErrOverApprovalLimit
	}
	return nil
}
