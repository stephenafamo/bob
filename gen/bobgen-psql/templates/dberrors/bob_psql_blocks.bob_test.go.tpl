{{- $hasChecks := false -}}
{{- range $table := .Tables -}}
{{- if $table.Constraints.Checks -}}{{- $hasChecks = true -}}{{- end -}}
{{- end -}}
{{if $hasChecks}}
{{if or (eq $.Driver "github.com/lib/pq") (hasPrefix "github.com/jackc/pgx/v5" $.Driver)}}
{{$.Importer.Import "errors"}}
{{$.Importer.Import "testing"}}
{{if eq $.Driver "github.com/lib/pq"}}
{{$.Importer.Import "github.com/lib/pq"}}
{{else}}
{{$.Importer.Import "github.com/jackc/pgx/v5/pgconn"}}
{{end}}

func TestCheckConstraintErrors(t *testing.T) {
	{{if eq $.Driver "github.com/lib/pq"}}
	newCheckErr := func(code, constraintName string) error {
		return &pq.Error{Code: pq.ErrorCode(code), Constraint: constraintName}
	}
	{{else}}
	newCheckErr := func(code, constraintName string) error {
		return &pgconn.PgError{Code: code, ConstraintName: constraintName}
	}
	{{end}}

	checkErr := newCheckErr("23514", "some_constraint")
	if !errors.Is(ErrCheckConstraint, checkErr) {
		t.Fatal("expected ErrCheckConstraint to match check violation")
	}
	if !ErrCheckConstraint.Is(checkErr) {
		t.Fatal("expected ErrCheckConstraint.Is to match check violation")
	}

	uniqueErr := newCheckErr("23505", "some_constraint")
	if errors.Is(ErrCheckConstraint, uniqueErr) {
		t.Fatal("expected ErrCheckConstraint not to match unique violation")
	}
	if ErrCheckConstraint.Is(uniqueErr) {
		t.Fatal("expected ErrCheckConstraint.Is not to match unique violation")
	}

	{{range $table := .Tables}}
	{{if $table.Constraints.Checks}}
	{{$tAlias := $.Aliases.Table $table.Key}}
	{{range $check := $table.Constraints.Checks}}
	{{- $errName := printf "ErrCheck%s" ($check.Name | camelcase) -}}
	t.Run("{{$tAlias.UpSingular}}_{{$errName}}", func(t *testing.T) {
		matchingErr := newCheckErr("23514", {{printf "%q" $check.Name}})
		if !errors.Is({{$tAlias.UpSingular}}Errors.{{$errName}}, matchingErr) {
			t.Fatalf("expected {{$errName}} to match constraint %q", {{printf "%q" $check.Name}})
		}
		if !{{$tAlias.UpSingular}}Errors.{{$errName}}.Is(matchingErr) {
			t.Fatalf("expected {{$errName}}.Is to match constraint %q", {{printf "%q" $check.Name}})
		}

		nonMatchingErr := newCheckErr("23514", "other_constraint")
		if errors.Is({{$tAlias.UpSingular}}Errors.{{$errName}}, nonMatchingErr) {
			t.Fatal("expected {{$errName}} not to match different constraint")
		}
		if {{$tAlias.UpSingular}}Errors.{{$errName}}.Is(nonMatchingErr) {
			t.Fatal("expected {{$errName}}.Is not to match different constraint")
		}
	})
	{{end}}
	{{end}}
	{{end}}
}
{{end}}
{{end}}
