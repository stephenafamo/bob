{{- if and .Table.Constraints.Uniques (not $.NoFactory)}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key}}
{{$factoryPackage := printf "%s/factory" $.ModelsPackage }}
{{$.Importer.Import "factory" $factoryPackage }}
{{$.Importer.Import "models" $.ModelsPackage}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "errors"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}

func Test{{$tAlias.UpSingular}}UniqueConstraintErrors(t *testing.T) {
	if testDB == nil {
		t.Skip("No database connection provided")
	}

	f := factory.New()
	tests := []struct{
		name        string
		expectedErr   *models.UniqueConstraintError
		conflictMods  func(context.Context, bob.Executor, *models.{{$tAlias.UpSingular}}) factory.{{$tAlias.UpSingular}}ModSlice
	}{
	{{range $index := (prepend $table.Constraints.Uniques $table.Constraints.Primary)}}
		{{- $errName := printf "ErrUnique%s" ($index.Name | camelcase) -}}
		{
			name: "{{$errName}}",
			expectedErr: models.{{$tAlias.UpSingular}}Errors.{{$errName}},
			conflictMods: func(ctx context.Context, exec bob.Executor, obj *models.{{$tAlias.UpSingular}}) factory.{{$tAlias.UpSingular}}ModSlice {
        shouldUpdate := false
        updateMods := make(factory.{{$tAlias.UpSingular}}ModSlice, 0, {{len $index.Columns}})

        {{range $indexColumn := $index.Columns}}
          {{- $colAlias := $tAlias.Column $indexColumn -}}
          {{- $column := $table.GetColumn $indexColumn -}}
          {{if $column.Nullable -}}
          if obj.{{$colAlias}}.IsNull() {
            shouldUpdate = true
            updateMods = append(updateMods, factory.{{$tAlias.UpSingular}}Mods.Random{{$colAlias}}NotNull(nil))
          }
          {{- end}}
        {{end}}

        if shouldUpdate {
          if err := obj.Update(ctx, exec, f.New{{$tAlias.UpSingular}}(updateMods...).BuildSetter()); err != nil {
            t.Fatalf("Error updating object: %v", err)
          }
        }

        return factory.{{$tAlias.UpSingular}}ModSlice{
          {{range $indexColumn := $index.Columns}}
            {{- $colAlias := $tAlias.Column $indexColumn -}}
            {{- $column := $table.GetColumn $indexColumn -}}
            factory.{{$tAlias.UpSingular}}Mods.{{$colAlias}}(obj.{{$colAlias}}),
          {{end}}
        }
			},
		},
	{{end}}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
      ctxTx, cancel := context.WithCancel(context.Background())
      defer cancel()

			tx, err := testDB.BeginTx(ctxTx, nil)
			if err != nil {
				t.Fatalf("Couldn't start database transaction: %v", err)
			}

			obj, err := f.New{{$tAlias.UpSingular}}(factory.{{$tAlias.UpSingular}}Mods.WithOneRelations()).Create(ctxTx, tx)
			if err != nil {
				t.Fatal(err)
			}

			obj2, err := f.New{{$tAlias.UpSingular}}().Create(ctxTx, tx)
			if err != nil {
				t.Fatal(err)
			}

      err = obj2.Update(ctxTx, tx, f.New{{$tAlias.UpSingular}}(tt.conflictMods(ctxTx, tx, obj)...).BuildSetter())
			if !errors.Is(models.ErrUniqueConstraint, err) {
				t.Fatalf("Expected: %s, Got: %v", tt.name, err)
			}
			if !errors.Is(tt.expectedErr, err) {
				t.Fatalf("Expected: %s, Got: %v", tt.name, err)
			}
			if !models.ErrUniqueConstraint.Is(err) {
				t.Fatalf("Expected: %s, Got: %v", tt.name, err)
			}
			if !tt.expectedErr.Is(err) {
				t.Fatalf("Expected: %s, Got: %v", tt.name, err)
			}
			if err = tx.Rollback(); err != nil {
				t.Fatal("Couldn't rollback database transaction")
			}
		})
	}
}
{{end -}}
