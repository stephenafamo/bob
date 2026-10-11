package sectionload

// P-07: mapKey は引数 vals の []byte 要素を string に書き換えてから
// キーを組み立てていた。渡されたスライスがその場限りの使い捨てなら無害
// だが、hooks.go の runThen は mapKey に渡した kv をそのまま keys へ
// 追加し(hooks.go:51)、その keys が buildEdgeSelect の match 値として
// Then 境界の SQL パラメータになる(hooks.go:93)。つまり書き換えは
// 関数内で閉じず、SQL パラメータへ流れ出る経路が現行コードに既にある。
// stitch.go:126 が渡す kv も execIsland が cloneKey して蓄積した永続
// スライス(res.parentKeys)の要素であり、使い捨てではない。
//
// このファイルは「mapKey は引数を書き換えない」という契約を DB 無しで
// 直接固定する。clonekey_test.go が cloneKey の契約(キーを保存するとき
// []byte を深くコピーする)を固定しているのと対になる。
//
// 【重要】このテストの前提は「渡した要素が本当に []byte であること」。
// 前提が崩れる([]byte 以外を渡してしまう)と修正前のコードでも素通り
// してしまい、検査自体が壊れていることに気づけない。呼び出しの前に
// mapKeyRequireByteElem がその前提を毎回確認する。

import "testing"

// mapKeyRequireByteElem は「テストの前提」を確認する。mapKey を呼ぶ前に
// 使う。
func mapKeyRequireByteElem(t *testing.T, vals []any, i int) {
	t.Helper()
	if _, ok := vals[i].([]byte); !ok {
		t.Fatalf("テストの前提が壊れている: vals[%d] は []byte でなければならないが %T だった", i, vals[i])
	}
}

// mapKeyAssertByteElem は「mapKey 呼び出し後も要素が []byte のまま、値も
// そのまま」であることを確認する。修正前のコードではここで落ちる。
func mapKeyAssertByteElem(t *testing.T, vals []any, i int, want string) {
	t.Helper()
	b, ok := vals[i].([]byte)
	if !ok {
		t.Fatalf("mapKey は引数を書き換えてはならない: vals[%d] の型が []byte から %T へ変わった(値: %v)", i, vals[i], vals[i])
	}
	if string(b) != want {
		t.Fatalf("mapKey は引数を書き換えてはならない: vals[%d] の値が %q から %q へ変わった", i, want, string(b))
	}
}

// ① 単一キー: []byte 要素を渡しても引数は不変。
func TestMapKey_DoesNotMutateSingleByteKeyArgument(t *testing.T) {
	vals := []any{[]byte("ab")}
	mapKeyRequireByteElem(t, vals, 0)

	k, err := mapKey(vals)
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}
	if k != "ab" {
		t.Fatalf("mapKey() = %#v, want %q", k, "ab")
	}
	mapKeyAssertByteElem(t, vals, 0, "ab")
}

// ② 複合キー: []byte を先頭以外(index 1・2)に置く。index 0 だけを検査
// すると「先頭要素しか守らない」実装でも通ってしまうため。[]byte 以外の
// 要素(index 0 の int)が触られていないことも併せて確認する。
func TestMapKey_DoesNotMutateCompositeKeyArgument(t *testing.T) {
	vals := []any{1, []byte("ab"), []byte("cd")}
	mapKeyRequireByteElem(t, vals, 1)
	mapKeyRequireByteElem(t, vals, 2)

	k, err := mapKey(vals)
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}
	if want := ([3]any{1, "ab", "cd"}); k != want {
		t.Fatalf("mapKey() = %#v, want %#v", k, want)
	}
	if _, ok := vals[0].(int); !ok {
		t.Fatalf("mapKey は []byte 以外の要素も書き換えてはならない: vals[0] の型が int から %T へ変わった", vals[0])
	}
	mapKeyAssertByteElem(t, vals, 1, "ab")
	mapKeyAssertByteElem(t, vals, 2, "cd")
}

// ③-a 戻り値の仕様(幅 1〜4 のパック形)が従来どおりであること。
// 修正前・修正後のどちらでも PASS する(挙動不変を固定するテスト)。
func TestMapKey_ReturnsComparableKeyForEveryWidth(t *testing.T) {
	cases := []struct {
		name string
		vals []any
		want any
	}{
		{"width1", []any{[]byte("ab")}, "ab"},
		{"width2", []any{1, []byte("ab")}, [2]any{1, "ab"}},
		{"width3", []any{1, "x", []byte("ab")}, [3]any{1, "x", "ab"}},
		{"width4", []any{1, "x", []byte("ab"), 2}, [4]any{1, "x", "ab", 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, err := mapKey(tc.vals)
			if err != nil {
				t.Fatalf("mapKey(%#v) error = %v, want nil", tc.vals, err)
			}
			if k != tc.want {
				t.Fatalf("mapKey(%#v) = %#v, want %#v", tc.vals, k, tc.want)
			}
		})
	}
}

// ③-b 同じ内容の []byte(裏の配列は別物)、および同値の string が、
// 同一の map エントリに落ちること。これが mapKey の存在理由であり、
// 修正で壊してはいけない一番の性質。修正前・修正後どちらでも PASS する。
func TestMapKey_EqualByteKeysMapToTheSameEntry(t *testing.T) {
	k1, err := mapKey([]any{[]byte("ab"), 7})
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}
	k2, err := mapKey([]any{append([]byte(nil), 'a', 'b'), 7})
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}
	k3, err := mapKey([]any{"ab", 7})
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}

	m := map[any]int{}
	m[k1]++
	m[k2]++
	m[k3]++
	if len(m) != 1 || m[k1] != 3 {
		t.Fatalf("同値のキーは 1 エントリに集約されなければならない: map = %#v (k1=%#v k2=%#v k3=%#v)", m, k1, k2, k3)
	}
}

// ④ キー幅超過: エラーメッセージと戻り値が従来どおりで、かつエラーを
// 返す前に引数を書き換えないこと(修正前はループが先に走るため書き換え
// られていた)。
func TestMapKey_UnsupportedWidthErrsWithoutTouchingArgument(t *testing.T) {
	contents := []string{"a", "b", "c", "d", "e"}
	vals := make([]any, len(contents))
	for i, c := range contents {
		vals[i] = []byte(c)
		mapKeyRequireByteElem(t, vals, i)
	}

	k, err := mapKey(vals)
	if k != nil {
		t.Fatalf("mapKey() key = %#v, want nil on error", k)
	}
	if err == nil {
		t.Fatal("mapKey() error = nil, want an unsupported-width error")
	}
	const want = "unsupported key width 5 (max 4)"
	if err.Error() != want {
		t.Fatalf("mapKey() error = %q, want %q", err, want)
	}
	for i, c := range contents {
		mapKeyAssertByteElem(t, vals, i, c)
	}
}

// ④-b キー幅 0(nil を含む)も従来どおりエラーであること。修正前・修正後
// どちらでも PASS する。
func TestMapKey_ZeroWidthErrs(t *testing.T) {
	k, err := mapKey(nil)
	if k != nil {
		t.Fatalf("mapKey(nil) key = %#v, want nil on error", k)
	}
	if err == nil {
		t.Fatal("mapKey(nil) error = nil, want an unsupported-width error")
	}
	const want = "unsupported key width 0 (max 4)"
	if err.Error() != want {
		t.Fatalf("mapKey(nil) error = %q, want %q", err, want)
	}
}
