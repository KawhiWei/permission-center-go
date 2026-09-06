// Package compile 将控制面的策略整理成稳定、不可变的发布快照。
package compile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/luck/permission-center-go/internal/policy/model"
	"github.com/luck/permission-center-go/internal/policy/validate"
)

// Snapshot 在策略发布前验证条件，并规范化角色和端点顺序，保证同一策略生成稳定内容。
func Snapshot(value model.Snapshot) (model.Snapshot, error) {
	if err := validate.Condition(value.Condition); err != nil {
		return model.Snapshot{}, err
	}
	value.ServiceResource = strings.TrimSpace(value.ServiceResource)
	if !value.AuthorizationType.Valid() {
		return model.Snapshot{}, fmt.Errorf("authorization_type must be api or data")
	}
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
