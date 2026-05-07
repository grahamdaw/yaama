package tui

import "github.com/grahamdaw/yaama/internal/db/generated"

type statusColumn struct {
	key   string
	title string
}

var boardStatuses = []statusColumn{
	{key: "idle", title: "Idle"},
	{key: "running", title: "Running"},
	{key: "blocked", title: "Blocked"},
	{key: "review", title: "Review"},
	{key: "done", title: "Done"},
}

func newStatusColumns() []column {
	columns := make([]column, 0, len(boardStatuses))
	for _, status := range boardStatuses {
		columns = append(columns, column{
			key:   status.key,
			title: status.title,
			cards: []generated.Agent{},
		})
	}

	return columns
}

func statusKeys() []string {
	keys := make([]string, 0, len(boardStatuses))
	for _, status := range boardStatuses {
		keys = append(keys, status.key)
	}
	return keys
}

func statusTitle(statusKey string) string {
	for _, status := range boardStatuses {
		if status.key == statusKey {
			return status.title
		}
	}
	return statusKey
}
