package view

import (
	"fmt"

	"github.com/tashirka1/k2/internal/metrics/model"
)

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func isNumericField(field string) bool {
	switch field {
	case "pid", "cpu", "ram", "ram_bytes":
		return true
	}
	return false
}

func sortDir(s model.Sort, field string) string {
	if s.Field == field {
		if s.Desc {
			return "asc"
		}
		return "desc"
	}
	if isNumericField(field) {
		return "desc"
	}
	return "asc"
}

func sortURL(category string, s model.Sort, field string) string {
	return fmt.Sprintf("/metrics/search/%s?sort=%s&dir=%s", category, field, sortDir(s, field))
}

func searchURL(category string, s model.Sort) string {
	u := "/metrics/search/" + category
	if s.Field != "" {
		u += "?sort=" + s.Field
		if s.Desc {
			u += "&dir=desc"
		}
	}
	return u
}

func sortIndicator(s model.Sort, field string) string {
	if s.Field != field {
		return ""
	}
	if s.Desc {
		return " ▼"
	}
	return " ▲"
}
