package tui

func seedColumns(showEmptyState bool) []column {
	statuses := []string{"Idle", "Running", "Blocked", "Review", "Done"}
	columns := make([]column, 0, len(statuses))

	for _, status := range statuses {
		columns = append(columns, column{
			title: status,
			cards: []string{},
		})
	}

	if showEmptyState {
		return columns
	}

	columns[0].cards = []string{"spike-ui"}
	columns[1].cards = []string{"refactor-db", "feat-auth", "bug-1234"}
	columns[2].cards = []string{"add-tests"}
	columns[3].cards = []string{"docs-pass"}

	return columns
}
