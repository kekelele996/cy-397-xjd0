package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// StringSlice 用于将 []string 以 JSON 数组形式持久化到 MySQL JSON 字段。
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(s))
	if err != nil {
		return nil, fmt.Errorf("marshal string slice: %w", err)
	}
	return string(b), nil
}

func (s *StringSlice) Scan(src any) error {
	if src == nil {
		*s = StringSlice{}
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported string slice source type %T", src)
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("unmarshal string slice: %w", err)
	}
	*s = StringSlice(out)
	return nil
}

// JSONMap 用于将 map[string]string 以 JSON 对象形式持久化到 MySQL JSON 字段。
type JSONMap map[string]string

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]string(m))
	if err != nil {
		return nil, fmt.Errorf("marshal json map: %w", err)
	}
	return string(b), nil
}

func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = JSONMap{}
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported json map source type %T", src)
	}
	out := make(map[string]string)
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("unmarshal json map: %w", err)
	}
	*m = JSONMap(out)
	return nil
}

// TemplateVariable 描述合同模板中的一个占位变量。
type TemplateVariable struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}

// TemplateVariables 为模板变量列表，按变量名排序后序列化，保证稳定输出。
type TemplateVariables []TemplateVariable

func (v TemplateVariables) Value() (driver.Value, error) {
	if v == nil {
		return "[]", nil
	}
	sorted := make(TemplateVariables, len(v))
	copy(sorted, v)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	b, err := json.Marshal(sorted)
	if err != nil {
		return nil, fmt.Errorf("marshal template variables: %w", err)
	}
	return string(b), nil
}

func (v *TemplateVariables) Scan(src any) error {
	if src == nil {
		*v = TemplateVariables{}
		return nil
	}
	var raw []byte
	switch val := src.(type) {
	case []byte:
		raw = val
	case string:
		raw = []byte(val)
	default:
		return fmt.Errorf("unsupported template variables source type %T", src)
	}
	var out []TemplateVariable
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("unmarshal template variables: %w", err)
	}
	*v = TemplateVariables(out)
	return nil
}

// VariableNames 返回模板包含的变量名集合。
func (v TemplateVariables) VariableNames() []string {
	names := make([]string, 0, len(v))
	for _, item := range v {
		names = append(names, item.Name)
	}
	sort.Strings(names)
	return names
}

// RequiredVariables 返回必填变量名集合。
func (v TemplateVariables) RequiredVariables() []string {
	names := make([]string, 0, len(v))
	for _, item := range v {
		if item.Required {
			names = append(names, item.Name)
		}
	}
	sort.Strings(names)
	return names
}

// NormalizeVariableName 将中文变量名等占位符清理为合法模板键。
func NormalizeVariableName(name string) string {
	return strings.TrimSpace(strings.ReplaceAll(name, " ", "_"))
}
