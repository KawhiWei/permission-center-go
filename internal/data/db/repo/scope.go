package repo

import "strings"

func bizScope(serviceResource, application string) string {
	if value := strings.TrimSpace(serviceResource); value != "" {
		return value
	}
	return strings.TrimSpace(application)
}
