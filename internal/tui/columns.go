package tui

import "github.com/grahamdaw/yaama/internal/db/generated"

func newStatusColumns() []column {
	type statusColumn struct {
		key   string
		title string
	}

	statuses := []statusColumn{
		{key: "idle", title: "Idle"},
		{key: "running", title: "Running"},
		{key: "blocked", title: "Blocked"},
		{key: "review", title: "Review"},
		{key: "done", title: "Done"},
	}

	columns := make([]column, 0, len(statuses))
	for _, status := range statuses {
		columns = append(columns, column{
			key:   status.key,
			title: status.title,
			cards: []generated.Agent{},
		})
	}

	return columns
}
