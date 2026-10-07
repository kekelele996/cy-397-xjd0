package model

import "testing"

func TestTemplateVariablesValue(t *testing.T) {
	tests := []struct {
		name  string
		input TemplateVariables
		want  string
	}{
		{name: "nil", input: nil, want: "[]"},
		{name: "empty non-nil", input: TemplateVariables{}, want: "[]"},
		{name: "non-empty sorted", input: TemplateVariables{
			{Name: "party_b", Label: "乙方", Required: true},
			{Name: "party_a", Label: "甲方", Required: true},
		}, want: `[{"name":"party_a","label":"甲方","required":true},{"name":"party_b","label":"乙方","required":true}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Value()
			if err != nil {
				t.Fatalf("Value() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Value() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
