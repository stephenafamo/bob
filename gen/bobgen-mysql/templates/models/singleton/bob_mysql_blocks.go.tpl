{{- define "helpers/where_variables"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
var (
	SelectWhere = Where[*dialect.SelectQuery]()
	UpdateWhere = Where[*dialect.UpdateQuery]()
	DeleteWhere = Where[*dialect.DeleteQuery]()
)
{{- end -}}

{{define "unique_constraint_error_detection_method" -}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "mysqlDriver" "github.com/go-sql-driver/mysql"}}
func (e *errUniqueConstraint) Is(target error) bool {
  err, ok := target.(*mysqlDriver.MySQLError)
  if !ok {
    return false
  }
  return err.Number == 1062 && strings.Contains(err.Message, e.s)
}
{{end -}}


{{define "one_update" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/um" $.Dialect)}}
// Update uses an executor to update the {{$tAlias.UpSingular}}
func (o *{{$tAlias.UpSingular}}) Update(ctx context.Context, exec bob.Executor, s *{{$tAlias.UpSingular}}Setter) error {
	_, err := {{$tAlias.UpPlural}}.Update(s.UpdateMod(), um.Where(o.pkEQ())).Exec(ctx, exec)
  if err != nil {
    return err
  }

  s.Overwrite(o)

  return nil
}
{{- end}}


{{define "slice_update" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (o {{$tAlias.UpSingular}}Slice) UpdateAll(ctx context.Context, exec bob.Executor, vals {{$tAlias.UpSingular}}Setter) error {
	_, err := {{$tAlias.UpPlural}}.Update(vals.UpdateMod(), o.UpdateMod()).Exec(ctx, exec)

  for i := range o {
    vals.Overwrite(o[i]) 
  }

  return err
}
{{- end}}

