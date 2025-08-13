{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

var {{$tAlias.UpPlural}} = Table[
  {{$tAlias.DownSingular}}Columns,
  {{$tAlias.DownSingular}}Indexes,
  {{$tAlias.DownSingular}}ForeignKeys,
  {{$tAlias.DownSingular}}Uniques,
  {{$tAlias.DownSingular}}Checks,
]{
  Schema:      {{quote $table.Schema}},
  Name:        {{quote $table.Name}},
  Columns:     {{$tAlias.DownSingular}}Columns{
    {{range $column := $table.Columns}}
    {{$tAlias.Column $column.Name}}: column{
      Name:      {{quote $column.Name}},
      DBType:    {{quote $column.DBType}},
      Default:   {{quote $column.Default}},
      Comment:   {{quote $column.Comment}},
      Nullable:  {{$column.Nullable}},
      Generated: {{$column.Generated}},
      AutoIncr:  {{$column.AutoIncr}},
    },
    {{- end}}
  },
  {{if $table.Indexes -}}
  {{$.Importer.Import "github.com/aarondl/opt/null" -}}
  Indexes:     {{$tAlias.DownSingular}}Indexes{
    {{range $index := $table.Indexes}}
    {{$tAlias.Index $index.Name}}: index{
      Type:    {{quote $index.Type}},
      Name:    {{quote $index.Name}},
      Columns: []indexColumn{
        {{range $column := $index.Columns}}
        {
          Name:         {{quote $column.Name}},
          Desc:         null.FromCond({{$column.Desc.GetOrZero}}, {{$column.Desc.IsValue}}),
          IsExpression: {{$column.IsExpression}},
        },
        {{- end}}
      },
      Unique:  {{$index.Unique}},
      Comment: {{quote $index.Comment}},
      {{block "index_extra_values" $index.Extra -}}
      {{- end}}
    },
    {{- end}}
  },
  {{- end}}
  {{if $table.Constraints.Primary -}}
  PrimaryKey: &constraint{
    Name:    {{quote $table.Constraints.Primary.Name}},
    Columns: {{printf "%#v" $table.Constraints.Primary.Columns}},
    Comment: {{quote $table.Constraints.Primary.Comment}},
    {{block "constraint_extra_values" $table.Constraints.Primary.Extra -}}
    {{- end}}
  },
  {{- end}}
  {{if $table.Constraints.Foreign -}}
  ForeignKeys: {{$tAlias.DownSingular}}ForeignKeys{
    {{range $foreign := $table.Constraints.Foreign}}
    {{$tAlias.Constraint $foreign.Name}}: foreignKey{
      constraint: constraint{
        Name:    {{quote $foreign.Name}},
        Columns: {{printf "%#v" $foreign.Columns}},
        Comment: {{quote $foreign.Comment}},
        {{block "constraint_extra_values" $foreign.Extra -}}
        {{- end}}
      },
      ForeignTable:   {{quote $foreign.ForeignTable}},
      ForeignColumns: {{printf "%#v" $foreign.ForeignColumns}},
    },
    {{- end}}
  },
  {{- end}}
  {{if $table.Constraints.Uniques -}}
  Uniques: {{$tAlias.DownSingular}}Uniques{
    {{range $unique := $table.Constraints.Uniques}}
    {{$tAlias.Constraint $unique.Name}}: constraint{
      Name:    {{quote $unique.Name}},
      Columns: {{printf "%#v" $unique.Columns}},
      Comment: {{quote $unique.Comment}},
      {{block "constraint_extra_values" $unique.Extra -}}
      {{- end}}
    },
    {{- end}}
  },
  {{- end}}
  {{if $table.Constraints.Checks -}}
  Checks: {{$tAlias.DownSingular}}Checks{
    {{range $check := $table.Constraints.Checks}}
    {{$tAlias.Constraint $check.Name}}: check{
      constraint: constraint{
        Name:    {{quote $check.Name}},
        Columns: {{printf "%#v" $check.Columns}},
        Comment: {{quote $check.Comment}},
        {{block "constraint_extra_values" $check.Extra -}}
        {{- end}}
      },
      Expression: {{quote $check.Expression}},
    },
    {{- end}}
  },
  {{- end}}
  Comment:     {{quote $table.Comment}},
}

type {{$tAlias.DownSingular}}Columns struct {
  {{range $column := $table.Columns -}}
  {{$tAlias.Column $column.Name}} column
  {{end -}}
}

func (c {{$tAlias.DownSingular}}Columns) AsSlice() []column {
  return []column{
    {{range $column := $table.Columns -}}
    c.{{$tAlias.Column $column.Name}},
    {{- end}}
  }
}

type {{$tAlias.DownSingular}}Indexes struct {
  {{range $index := $table.Indexes -}}
  {{$tAlias.Index $index.Name}} index
  {{end -}}
}

func (i {{$tAlias.DownSingular}}Indexes) AsSlice() []index {
  return []index{
    {{range $index := $table.Indexes -}}
    i.{{$tAlias.Index $index.Name}},
    {{- end}}
  }
}

type {{$tAlias.DownSingular}}ForeignKeys struct {
  {{range $foreign := $table.Constraints.Foreign -}}
  {{$tAlias.Constraint $foreign.Name}} foreignKey
  {{end -}}
}

func (f {{$tAlias.DownSingular}}ForeignKeys) AsSlice() []foreignKey {
  return []foreignKey{
    {{range $foreign := $table.Constraints.Foreign -}}
    f.{{$tAlias.Constraint $foreign.Name}},
    {{- end}}
  }
}

type {{$tAlias.DownSingular}}Uniques struct {
  {{range $unique := $table.Constraints.Uniques -}}
  {{$tAlias.Constraint $unique.Name}} constraint
  {{end -}}
}

func (u {{$tAlias.DownSingular}}Uniques) AsSlice() []constraint {
  return []constraint{
    {{range $unique := $table.Constraints.Uniques -}}
    u.{{$tAlias.Constraint $unique.Name}},
    {{- end}}
  }
}

type {{$tAlias.DownSingular}}Checks struct {
  {{range $check := $table.Constraints.Checks -}}
  {{$tAlias.Constraint $check.Name}} check
  {{end -}}
}

func (c {{$tAlias.DownSingular}}Checks) AsSlice() []check {
  return []check{
    {{range $check := $table.Constraints.Checks -}}
    c.{{$tAlias.Constraint $check.Name}},
    {{- end}}
  }
}
