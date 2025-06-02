{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

// Set the testDB to enable tests that use the database
var testDB bob.Transactor

type (
	{{range $enum := $.Enums -}}
		{{$enum.Type}} = models.{{$enum.Type}}
	{{end}}
)

{{- range $table := .Tables}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
  // Make sure the type {{$tAlias.UpSingular}} runs hooks after queries
	var _ bob.HookableType = &models.{{$tAlias.UpSingular}}{}
{{end}}

{{$doneTypes := dict }}
{{- range $table := .Tables}}
{{- $tAlias := $.Aliases.Table $table.Key}}
  {{range $column := $table.Columns -}}
    {{/*
    * We are in a test
    * We know that the test is in a separate package
    * We also know that there is no way to define a type that is ONLY used in tests
    * So we use backslashes as the package name which will never match a package
      to prevent assuming that the type is in the current package
    */}}
    {{- $colTyp := $.Types.GetWithoutImporting `\\\\\\\\\\\\` $column.Type -}}
    {{- if hasKey $doneTypes $column.Type}}{{continue}}{{end -}}
    {{- $_ :=  set $doneTypes $column.Type nil -}}
    {{- $typInfo :=  $.Types.Index $colTyp -}}
    {{- if $typInfo.NoScannerValuerTest}}{{continue}}{{end -}}
    {{- if isPrimitiveType $colTyp}}{{continue}}{{end -}}
      {{$.Importer.ImportList $typInfo.Imports -}}
      {{$.Importer.Import "database/sql"}}
      {{$.Importer.Import "database/sql/driver"}}
      // Make sure the type {{$colTyp}} satisfies database/sql.Scanner
      var _ sql.Scanner = (*{{$colTyp}})(nil)

      // Make sure the type {{$colTyp}} satisfies database/sql/driver.Valuer
      var _ driver.Valuer = *new({{$colTyp}})

  {{end -}}
{{- end}}
