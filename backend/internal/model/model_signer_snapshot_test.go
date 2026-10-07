package model

import "testing"

func TestSignerSnapshotsEqualSet(t *testing.T) {
	tests := []struct {
		name string
		a    SignerSnapshots
		b    SignerSnapshots
		want bool
	}{
		{name: "both empty", a: nil, b: SignerSnapshots{}, want: true},
		{
			name: "same set different order",
			a:    SignerSnapshots{{Name: "张三", Role: "甲方"}, {Name: "李四", Role: "乙方"}},
			b:    SignerSnapshots{{Name: "李四", Role: "乙方"}, {Name: "张三", Role: "甲方"}},
			want: true,
		},
		{
			name: "same name different role",
			a:    SignerSnapshots{{Name: "张三", Role: "甲方"}},
			b:    SignerSnapshots{{Name: "张三", Role: "乙方"}},
			want: false,
		},
		{
			name: "missing signer",
			a:    SignerSnapshots{{Name: "张三", Role: "甲方"}, {Name: "李四", Role: "乙方"}},
			b:    SignerSnapshots{{Name: "张三", Role: "甲方"}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.EqualSet(tt.b); got != tt.want {
				t.Fatalf("EqualSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignerSnapshotsValueScan(t *testing.T) {
	in := SignerSnapshots{{Name: "李四", Role: "乙方"}, {Name: "张三", Role: "甲方"}}
	val, err := in.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	var out SignerSnapshots
	if err := out.Scan(val); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if !in.EqualSet(out) {
		t.Fatalf("round trip mismatch: %v vs %v", in, out)
	}
}

func TestFromSigners(t *testing.T) {
	got := FromSigners([]ContractSigner{
		{Name: "李四", Role: "乙方", SignInfo: "x"},
		{Name: "张三", Role: "甲方"},
	})
	// 排序键为 role+name，按字节序「乙方」在「甲方」之前；签署信息不进快照。
	if len(got) != 2 || got[0].Role != "乙方" || got[1].Role != "甲方" {
		t.Fatalf("FromSigners() = %+v, want role-sorted snapshots without sign_info", got)
	}
}
