package mwanachamaactor

func (m *userManager) ForRun(runID string) (UserManager, error) {
	scoped, err := m.store.ForRun(runID)
	if err != nil {
		return nil, err
	}
	run := *m
	run.store = scoped
	return &run, nil
}
