// Package compile constructs immutable, canonicalized policy snapshots.
package compile

import (
	"github.com/luck/permission-center-go/internal/policy/model"
	"github.com/luck/permission-center-go/internal/policy/validate"
	"sort"
	"strings"
)

func Snapshot(value model.Snapshot) (model.Snapshot, error) {
	if err := validate.Condition(value.Condition); err != nil {
		return model.Snapshot{}, err
	}
	value.ServiceResource = strings.TrimSpace(value.ServiceResource)
	value.RoleIDs = uniqueStrings(value.RoleIDs)
	sort.Strings(value.RoleIDs)
	sort.Slice(value.EndpointIDs, func(i, j int) bool { return value.EndpointIDs[i].String() < value.EndpointIDs[j].String() })
	return value, nil
}
func uniqueStrings(values []string) []string {
	found := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := found[value]; ok {
			continue
		}
		found[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
