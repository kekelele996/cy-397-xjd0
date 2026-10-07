package service

import (
	"testing"
)

func TestExtractContractNo(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want string
	}{
		{name: "labeled chinese", desc: "合同编号 12345 的对方违约", want: "12345"},
		{name: "label with colon", desc: "合同号：88，拖欠工资", want: "88"},
		{name: "label hash", desc: "合同编号#777", want: "777"},
		{name: "english contract no", desc: "regarding contract no. 42 please review", want: "42"},
		{name: "hash fallback", desc: "请处理 #56 这单纠纷", want: "56"},
		{name: "date is not a contract no", desc: "合同日期 2026-01-01 签署", want: ""},
		{name: "plain text no number", desc: "就是来咨询一下，没有合同", want: ""},
		{name: "no labeled number", desc: "金额 3000 元还没给", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractContractNo(tt.desc); got != tt.want {
				t.Fatalf("ExtractContractNo(%q) = %q, want %q", tt.desc, got, tt.want)
			}
		})
	}
}

func TestParseContractID(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    uint64
		wantErr bool
	}{
		{name: "plain", raw: "123", want: 123},
		{name: "prefixed dash", raw: "HT-2026-12", want: 12},
		{name: "letter suffix digits", raw: "C99", want: 99},
		{name: "zero invalid", raw: "0", wantErr: true},
		{name: "letters only", raw: "ABC", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseContractID(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseContractID(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("parseContractID(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}
