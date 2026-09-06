// Package validate 校验受限的结构化策略 DSL，拒绝任意脚本和畸形条件树。
package validate

import (
	"fmt"
	"github.com/luck/permission-center-go/internal/policy/model"
	"strings"
)

const maxDepth = 8
const maxNodes = 64

// Condition 限制条件树深度、节点数、操作符和值类型，防止无效或过度复杂的策略进入快照。
func Condition(value *model.Condition) error {
	if value == nil {
		return nil
	}
	count := 0
	return condition(value, 1, &count)
}
func condition(value *model.Condition, depth int, count *int) error {
	*count++
	if depth > maxDepth || *count > maxNodes {
		return fmt.Errorf("condition exceeds maximum depth or node count")
	}
	kinds := 0
	if len(value.All) > 0 {
		kinds++
	}
	if len(value.Any) > 0 {
		kinds++
	}
	if value.Not != nil {
		kinds++
	}
	if value.Comparison != nil {
		kinds++
	}
	if kinds != 1 {
		return fmt.Errorf("condition node must contain exactly one operator")
	}
	if len(value.All) > 0 {
		for i := range value.All {
			if err := condition(&value.All[i], depth+1, count); err != nil {
				return err
			}
		}
		return nil
	}
	if len(value.Any) > 0 {
		for i := range value.Any {
			if err := condition(&value.Any[i], depth+1, count); err != nil {
				return err
			}
		}
		return nil
	}
	if value.Not != nil {
		return condition(value.Not, depth+1, count)
	}
	c := value.Comparison
	if !c.Op.Valid() {
		return fmt.Errorf("unsupported comparison operator")
	}
	if err := valueRef(c.Left); err != nil {
		return err
	}
	if c.Op == model.OpExists {
		if c.Right != nil {
			return fmt.Errorf("exists must not define right operand")
		}
		return nil
	}
	if c.Right == nil {
		return fmt.Errorf("comparison requires right operand")
	}
	if err := valueRef(*c.Right); err != nil {
		return err
	}
	if c.Left.Type != c.Right.Type {
		return fmt.Errorf("comparison operands must use the same type")
	}
	return nil
}
func valueRef(value model.ValueRef) error {
	if !value.Source.Valid() || !value.Type.Valid() {
		return fmt.Errorf("invalid value reference")
	}
	if value.Source == model.SourceLiteral {
		if value.Value == nil {
			return fmt.Errorf("literal value is required")
		}
		return nil
	}
	if strings.TrimSpace(value.Path) == "" || strings.Contains(value.Path, "..") {
		return fmt.Errorf("attribute path is invalid")
	}
	return nil
}
