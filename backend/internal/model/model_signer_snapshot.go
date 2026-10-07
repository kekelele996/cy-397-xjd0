package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
)

// SignerSnapshot 是登记到工单上的合同签署方摘要（不含签署时间等易变信息）。
type SignerSnapshot struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// Key 返回用于对账的稳定键：姓名 + 角色。
func (s SignerSnapshot) Key() string {
	return s.Role + "\x00" + s.Name
}

// SignerSnapshots 是工单登记的签署方列表，以 JSON 持久化。
// 序列化前按 key 排序，保证对账与展示稳定。
type SignerSnapshots []SignerSnapshot

// FromSigners 从合同签署方记录生成快照（只取姓名与角色）。
func FromSigners(signers []ContractSigner) SignerSnapshots {
	out := make(SignerSnapshots, 0, len(signers))
	for _, s := range signers {
		out = append(out, SignerSnapshot{Name: s.Name, Role: s.Role})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

// EqualSet 判断两份签署方快照是否为同一集合（忽略顺序与重复）。
func (s SignerSnapshots) EqualSet(other SignerSnapshots) bool {
	counts := make(map[string]int, len(s)+len(other))
	for _, item := range s {
		counts[item.Key()]++
	}
	for _, item := range other {
		counts[item.Key()]--
	}
	for _, v := range counts {
		if v != 0 {
			return false
		}
	}
	return true
}

func (s SignerSnapshots) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	sorted := make(SignerSnapshots, len(s))
	copy(sorted, s)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key() < sorted[j].Key() })
	b, err := json.Marshal([]SignerSnapshot(sorted))
	if err != nil {
		return nil, fmt.Errorf("marshal signer snapshots: %w", err)
	}
	return string(b), nil
}

func (s *SignerSnapshots) Scan(src any) error {
	if src == nil {
		*s = SignerSnapshots{}
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported signer snapshots source type %T", src)
	}
	var out []SignerSnapshot
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("unmarshal signer snapshots: %w", err)
	}
	*s = SignerSnapshots(out)
	return nil
}
