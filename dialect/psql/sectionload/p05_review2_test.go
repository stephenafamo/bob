package sectionload

// P-05 Copilot外部レビュー(2回目、issue P-05-multi-hop-relationships.md
// §19)の指摘に対応するテスト。
//
// A5 (HIGH #2): 新規公開関数 CarriedKeyName に専用のユニットテストが
// 存在しなかった。CarriedKeyPrefix は既存の "_sl_k0" 系テスト(SQL文字列
// 検査など)で間接的にカバーされていたが、CarriedKeyName(i) 自身が誤った
// 名前を返しても検出できなかった。TestCarriedKeyName はこれを直接検証
// する。

import (
	"strings"
	"testing"
)

// TestCarriedKeyName pins CarriedKeyName's contract: for every i, it
// returns CarriedKeyPrefix followed by the decimal digits of i, with no
// separator — the exact "_sl_k0", "_sl_k1", ... sequence Edge.Carried's
// doc comment (parts.go) promises, and the same value the generated
// template (01_edges.go.tpl) now calls this function to produce instead
// of duplicating the literal.
func TestCarriedKeyName(t *testing.T) {
	tests := map[string]struct {
		i    int
		want string
	}{
		"zero":                     {0, "_sl_k0"},
		"single digit":             {1, "_sl_k1"},
		"multi-digit (two digits)": {10, "_sl_k10"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := CarriedKeyName(tc.i)
			if got != tc.want {
				t.Errorf("CarriedKeyName(%d) = %q, want %q", tc.i, got, tc.want)
			}
			// CarriedKeyName must always be built from CarriedKeyPrefix (the
			// single source of truth parentexpr.go's reservedNameRe also
			// derives from) — a hardcoded "_sl_k" duplicate inside
			// CarriedKeyName that happened to match today's constant value
			// would still pass the exact-string checks above but would
			// silently desync the moment CarriedKeyPrefix changed.
			if !strings.HasPrefix(got, CarriedKeyPrefix) {
				t.Errorf("CarriedKeyName(%d) = %q, want prefix %q (CarriedKeyPrefix)", tc.i, got, CarriedKeyPrefix)
			}
		})
	}
}
