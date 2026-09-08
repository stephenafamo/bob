{{- /*
  Round-trip tests for the Go types generated for Postgres array columns.
  Each distinct array type is sent as a query parameter and scanned back,
  both bare and wrapped in the nullable type, under every configured driver
  (lib/pq, pgx/stdlib and native pgx).
  See https://github.com/stephenafamo/bob/issues/90 and
  https://github.com/stephenafamo/bob/issues/739
*/ -}}
{{- if has "type_monsters" $.TableNames -}}
{{- range $table := $.Tables -}}
{{- if ne $table.Key "type_monsters"}}{{continue}}{{end -}}
{{- $tAlias := $.Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "encoding/json"}}
{{$.Importer.Import "reflect"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/dialect/psql"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}

// jsonArraysEqual compares JSON arrays semantically, since jsonb normalizes its input
func jsonArraysEqual[T any, A ~[]T, B ~[]T](a A, b B, val func(T) []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		var av, bv any
		if err := json.Unmarshal(val(a[i]), &av); err != nil {
			return false
		}
		if err := json.Unmarshal(val(b[i]), &bv); err != nil {
			return false
		}
		if !reflect.DeepEqual(av, bv) {
			return false
		}
	}

	return true
}

// Test{{$tAlias.UpSingular}}ArrayRoundTrip sends random arrays as query parameters and scans them back
func Test{{$tAlias.UpSingular}}ArrayRoundTrip(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	{{$doneTypes := dict -}}
	{{range $column := $table.Columns -}}
		{{- if not (or (hasPrefix "pgtypes.Array[" $column.Type) (hasPrefix "pgtypes.EnumArray[" $column.Type))}}{{continue}}{{end -}}
		{{- if hasKey $doneTypes $column.Type}}{{continue}}{{end -}}
		{{- $_ := set $doneTypes $column.Type nil -}}
		{{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
		{{- $nullTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type true -}}
		{{- $cast := trimPrefix "_" $column.DBType -}}
		{{- $elemTyp := trimPrefix "pgtypes.Array[" $column.Type | trimSuffix "]" -}}
	t.Run("{{$column.DBType}}", func(t *testing.T) {
		want := random_{{normalizeType $column.Type}}(nil, {{$column.LimitsString}})

		got, err := bob.One(ctx, tx, psql.RawQuery("SELECT ?::{{$cast}}", want), scan.SingleColumnMapper[{{$colTyp}}])
		if err != nil {
			t.Fatalf("Error scanning {{$colTyp}}: %v", err)
		}
		{{if hasPrefix "pgtypes.Array[types.JSON[" $column.Type -}}
		if !jsonArraysEqual(want, got, func(v {{$.Types.Get $.CurrentPackage $.Importer $elemTyp}}) []byte { return v.Val }) {
		{{- else -}}
		if !({{$.Types.GetCompareExpr $.CurrentPackage $.Importer $column.Type false false | replace "AAA" "want" | replace "BBB" "got"}}) {
		{{- end}}
			t.Errorf("{{$colTyp}}: got %v, want %v", got, want)
		}

		// wrapped in the nullable type
		gotNull, err := bob.One(ctx, tx, psql.RawQuery("SELECT ?::{{$cast}}", want), scan.SingleColumnMapper[{{$nullTyp}}])
		if err != nil {
			t.Fatalf("Error scanning {{$nullTyp}}: %v", err)
		}
		if !({{$.Types.GetNullTypeValid $.CurrentPackage $column.Type "gotNull"}}) {
			t.Fatalf("{{$nullTyp}}: expected a value, got null")
		}
		gotUnwrapped := {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $column.Type "gotNull" true}}
		{{if hasPrefix "pgtypes.Array[types.JSON[" $column.Type -}}
		if !jsonArraysEqual(want, gotUnwrapped, func(v {{$.Types.Get $.CurrentPackage $.Importer $elemTyp}}) []byte { return v.Val }) {
		{{- else -}}
		if !({{$.Types.GetCompareExpr $.CurrentPackage $.Importer $column.Type false false | replace "AAA" "want" | replace "BBB" "gotUnwrapped"}}) {
		{{- end}}
			t.Errorf("{{$nullTyp}}: got %v, want %v", gotUnwrapped, want)
		}

		// an empty array is not NULL
		gotEmpty, err := bob.One(ctx, tx, psql.RawQuery("SELECT ?::{{$cast}}", {{$colTyp}}{}), scan.SingleColumnMapper[{{$colTyp}}])
		if err != nil {
			t.Fatalf("Error scanning empty {{$colTyp}}: %v", err)
		}
		if gotEmpty == nil || len(gotEmpty) != 0 {
			t.Errorf("{{$colTyp}}: expected an empty array, got %#v", gotEmpty)
		}

		// a nil array is NULL
		gotNil, err := bob.One(ctx, tx, psql.RawQuery("SELECT ?::{{$cast}}", {{$colTyp}}(nil)), scan.SingleColumnMapper[{{$nullTyp}}])
		if err != nil {
			t.Fatalf("Error scanning NULL {{$nullTyp}}: %v", err)
		}
		if {{$.Types.GetNullTypeValid $.CurrentPackage $column.Type "gotNil"}} {
			t.Errorf("{{$nullTyp}}: expected null, got %v", gotNil)
		}
	})
	{{end}}
}
{{end -}}
{{- end -}}
