package gen

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/orm"
)

type testImporter map[string]struct{}

func (i testImporter) Import(pkgs ...string) string {
	for _, pkg := range pkgs {
		i[pkg] = struct{}{}
	}
	return ""
}

func (i testImporter) ImportList(pkgs []string) string {
	for _, pkg := range pkgs {
		i[pkg] = struct{}{}
	}
	return ""
}

func (i testImporter) ToList() []string {
	out := make([]string, 0, len(i))
	for pkg := range i {
		out = append(out, pkg)
	}
	return out
}

func Test_enumValToIdentifier(t *testing.T) {
	tests := []struct {
		val      string
		expected string
	}{
		{"in_progress", "InProgress"},
		{"in-progress", "InProgress"},
		{"in progress", "InProgress"},
		{"IN_PROGRESS", "InProgress"},
		{"in___-__progress", "InProgress"},
		{" in progress ", "InProgress"},
		// This is OK, because enum values are prefixed with the type name, e.g. TaskStatus1InProgress
		{"1-in-progress", "1InProgress"},
		{"start < end", "StartU3CEnd"},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			if actual := enumValToIdentifier(tt.val); actual != tt.expected {
				t.Errorf("enumValToIdentifier(%q) = %q; want %q", tt.val, actual, tt.expected)
			}
		})
	}
}

func Test_enumValToScreamingSnakeCase(t *testing.T) {
	tests := []struct {
		val      string
		expected string
	}{
		{"in_progress", "IN_PROGRESS"},
		{"in-progress", "IN_PROGRESS"},
		{"in progress", "IN_PROGRESS"},
		{"IN_PROGRESS", "IN_PROGRESS"},
		{"in___-__progress", "IN______PROGRESS"},
		{" in progress ", "_IN_PROGRESS_"},
		{"1-in-progress", "1_IN_PROGRESS"},
		{"start < end", "START_U3c_END"},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			if actual := enumValToScreamingSnakeCase(tt.val); actual != tt.expected {
				t.Errorf("enumValToScreamingSnakeCase(%q) = %q; want %q", tt.val, actual, tt.expected)
			}
		})
	}
}

func TestTablePackageJoinsOmitGlobalsWithoutRelationships(t *testing.T) {
	t.Parallel()

	data := expandDrivenThenLoadTemplateData()
	data.Tables = drivers.Tables[any, any]{data.AllTables[3]}
	data.ModelSplit.Mode = modelPackageSplitModeTablePackages
	data.ModelSplit.Generation = modelSplitGenerationComponent
	data.ModelSplit.CurrentComponent = data.ModelSplit.TableComponents["profiles"]

	content, err := fs.ReadFile(templates, "templates/joins/bob_joins.bob.go.tpl")
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := template.New("joins").Funcs(sprig.GenericFuncMap()).Funcs(templateFunctions).Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "BuildProfileJoins") {
		t.Fatalf("did not expect join globals without relationships:\n%s", out.String())
	}
}

func TestTablePackageSingletonJoinsAreFlattened(t *testing.T) {
	t.Parallel()

	data := expandDrivenThenLoadTemplateData()
	data.Tables = drivers.Tables[any, any]{data.AllTables[0]}
	data.ModelSplit.Mode = modelPackageSplitModeTablePackages
	data.ModelSplit.Generation = modelSplitGenerationComponent
	data.ModelSplit.CurrentComponent = data.ModelSplit.TableComponents["users"]

	content, err := fs.ReadFile(templates, "templates/joins/bob_joins.bob.go.tpl")
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := template.New("joins").Funcs(sprig.GenericFuncMap()).Funcs(templateFunctions).Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "SelectJoins = BuildJoinSet[UserJoins[*dialect.SelectQuery]]") {
		t.Fatalf("expected flattened table-package joins, got:\n%s", got)
	}
	if strings.Contains(got, "type joins[") {
		t.Fatalf("table-package joins must not retain the all-table wrapper:\n%s", got)
	}
}

func TestSQLiteTablePackageSingletonJoinsAreFlattened(t *testing.T) {
	t.Parallel()

	data := expandDrivenThenLoadTemplateData()
	data.Dialect = "sqlite"
	data.Tables = drivers.Tables[any, any]{data.AllTables[0]}
	data.ModelSplit.Mode = modelPackageSplitModeTablePackages
	data.ModelSplit.Generation = modelSplitGenerationComponent
	data.ModelSplit.CurrentComponent = data.ModelSplit.TableComponents["users"]

	base, err := fs.ReadFile(templates, "templates/joins/bob_joins.bob.go.tpl")
	if err != nil {
		t.Fatal(err)
	}
	override, err := fs.ReadFile(sqliteTemplates, "bobgen-sqlite/templates/joins/bob_sqlite_blocks.bob.go.tpl")
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := template.New("joins").Funcs(sprig.GenericFuncMap()).Funcs(templateFunctions).Parse(string(base))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tpl.Parse(string(override)); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "SelectJoins = BuildJoinSet[UserJoins[*dialect.SelectQuery]]") {
		t.Fatalf("expected flattened SQLite table-package joins, got:\n%s", got)
	}
	if strings.Contains(got, "getJoins[") {
		t.Fatalf("SQLite table-package joins must not retain the all-table helper:\n%s", got)
	}
}

func TestTablePackageExpandMethodsUseCommonInterface(t *testing.T) {
	t.Parallel()

	data := expandDrivenThenLoadTemplateData()
	data.Tables = drivers.Tables[any, any]{data.AllTables[0]}
	data.ModelSplit.Mode = modelPackageSplitModeTablePackages
	data.ModelSplit.Generation = modelSplitGenerationComponent
	data.ModelSplit.CurrentComponent = data.ModelSplit.TableComponents["users"]

	content, err := fs.ReadFile(templates, "templates/loaders/table/110_loaders.go.tpl")
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := template.New("loaders").Funcs(sprig.GenericFuncMap()).Funcs(templateFunctions).Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"ForExpandMap(expands map[string]struct{}, maxDepth int, computedTerminal func(string) bool)",
		"ForExpandPaths(paths []string, maxDepth int, computedTerminal func(string) bool)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected table-package loader to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "opts ...ExpandLoadOption") {
		t.Fatalf("table-package loaders must not expose package-specific option types:\n%s", got)
	}
}

func TestTablePackageLoadersOmitGlobalsWithoutRelationships(t *testing.T) {
	t.Parallel()

	data := expandDrivenThenLoadTemplateData()
	data.Tables = drivers.Tables[any, any]{data.AllTables[3]}
	data.ModelSplit.Mode = modelPackageSplitModeTablePackages
	data.ModelSplit.Generation = modelSplitGenerationComponent
	data.ModelSplit.CurrentComponent = data.ModelSplit.TableComponents["profiles"]

	content, err := fs.ReadFile(templates, "templates/loaders/bob_loaders.bob.go.tpl")
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := template.New("loaders").Funcs(sprig.GenericFuncMap()).Funcs(templateFunctions).Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"BuildProfilePreloader", "BuildProfileThenLoader"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("did not expect %q without relationships:\n%s", unwanted, out.String())
		}
	}
}

func TestTablePackageSingletonTemplatesAreFlattened(t *testing.T) {
	t.Parallel()

	data := expandDrivenThenLoadTemplateData()
	data.Tables = drivers.Tables[any, any]{data.AllTables[0]}
	data.ModelSplit.Mode = modelPackageSplitModeTablePackages
	data.ModelSplit.Generation = modelSplitGenerationComponent
	data.ModelSplit.CurrentComponent = data.ModelSplit.TableComponents["users"]

	tests := []struct {
		name       string
		path       string
		contains   []string
		notContain []string
	}{
		{
			name: "where",
			path: "templates/where/bob_where.bob.go.tpl",
			contains: []string{
				"SelectWhere = BuildUserWhere[*dialect.SelectQuery](Users.Columns)",
				"func Where[Q psql.Filterable]() UserWhere[Q]",
			},
			notContain: []string{"SelectWhere = Where[*dialect.SelectQuery]()", "Users UserWhere[Q]"},
		},
		{
			name: "loaders",
			path: "templates/loaders/bob_loaders.bob.go.tpl",
			contains: []string{
				"var Preload = BuildUserPreloader()",
				"SelectThenLoad = BuildUserThenLoader[*dialect.SelectQuery]()",
			},
			notContain: []string{"type preloaders struct", "User UserPreloader"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := fs.ReadFile(templates, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			tpl, err := template.New(tt.name).Funcs(sprig.GenericFuncMap()).Funcs(templateFunctions).Parse(string(content))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := tpl.Execute(&out, &data); err != nil {
				t.Fatal(err)
			}
			got := out.String()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("expected %q in generated output:\n%s", want, got)
				}
			}
			for _, unwanted := range tt.notContain {
				if strings.Contains(got, unwanted) {
					t.Fatalf("did not expect %q in generated output:\n%s", unwanted, got)
				}
			}
		})
	}
}

func TestRelationshipMutationMethodsTemplateCanBeDisabled(t *testing.T) {
	content, err := fs.ReadFile(templates, "templates/models/table/011_rel_ops.go.tpl")
	if err != nil {
		t.Fatal(err)
	}

	tpl, err := template.New("rel_ops").
		Funcs(sprig.GenericFuncMap()).
		Funcs(templateFunctions).
		Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	data := TemplateData[any, any, any]{
		Importer: testImporter{},
		Table: drivers.Table[any, any]{
			Constraints: drivers.Constraints[any]{
				Primary: &drivers.Constraint[any]{Columns: []string{"id"}},
			},
		},
		RelationshipMutationMethods: false,
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		t.Fatal(err)
	}

	if got := strings.TrimSpace(out.String()); got != "" {
		t.Fatalf("expected relationship mutation methods template to be empty, got:\n%s", got)
	}
}

func TestSliceMutationMethodsTemplateCanBeDisabled(t *testing.T) {
	content, err := fs.ReadFile(templates, "templates/models/table/007_slice_methods.go.tpl")
	if err != nil {
		t.Fatal(err)
	}

	tpl, err := template.New("slice_methods").
		Funcs(sprig.GenericFuncMap()).
		Funcs(templateFunctions).
		Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	data := TemplateData[any, any, any]{
		Dialect:  "psql",
		Importer: testImporter{},
		Table: drivers.Table[any, any]{
			Key: "widget",
			Constraints: drivers.Constraints[any]{
				Primary: &drivers.Constraint[any]{Columns: []string{"id"}},
			},
		},
		Aliases: drivers.Aliases{
			"widget": {
				UpPlural:   "Widgets",
				UpSingular: "Widget",
			},
		},
		SliceMutationMethods: false,
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.Contains(got, "func (o WidgetSlice) AfterQueryHook(") {
		t.Fatalf("expected slice AfterQueryHook to remain, got:\n%s", got)
	}

	for _, removed := range []string{"UpdateAll", "DeleteAll", "ReloadAll", "UpdateMod", "DeleteMod", "pkIN", "copyMatchingRows"} {
		if strings.Contains(got, removed) {
			t.Fatalf("expected %s to be omitted, got:\n%s", removed, got)
		}
	}
}

func expandDrivenThenLoadTemplateData() TemplateData[any, any, any] {
	videoRel := orm.Relationship{
		Name: "user_videos",
		Sides: []orm.RelSide{{
			From:        "users",
			To:          "videos",
			FromColumns: []string{"id"},
			ToColumns:   []string{"user_id"},
			Modify:      "to",
		}},
	}
	profileRel := orm.Relationship{
		Name: "user_profile",
		Sides: []orm.RelSide{{
			From:        "users",
			To:          "profiles",
			FromColumns: []string{"profile_id"},
			ToColumns:   []string{"id"},
			Modify:      "from",
		}},
	}
	commentRel := orm.Relationship{
		Name: "video_comments",
		Sides: []orm.RelSide{{
			From:        "videos",
			To:          "comments",
			FromColumns: []string{"id"},
			ToColumns:   []string{"video_id"},
			Modify:      "to",
		}},
	}

	return TemplateData[any, any, any]{
		Dialect:  "psql",
		Importer: testImporter{},
		Table: drivers.Table[any, any]{
			Key:     "users",
			Columns: []drivers.Column{{Name: "id"}, {Name: "profile_id"}},
		},
		Tables: drivers.Tables[any, any]{
			{Key: "users", Columns: []drivers.Column{{Name: "id"}, {Name: "profile_id"}}},
			{Key: "videos", Columns: []drivers.Column{{Name: "id"}, {Name: "user_id"}}},
			{Key: "comments", Columns: []drivers.Column{{Name: "id"}, {Name: "video_id"}}},
			{Key: "profiles", Columns: []drivers.Column{{Name: "id"}}},
		},
		AllTables: drivers.Tables[any, any]{
			{Key: "users", Columns: []drivers.Column{{Name: "id"}, {Name: "profile_id"}}},
			{Key: "videos", Columns: []drivers.Column{{Name: "id", Generated: true}, {Name: "user_id"}}},
			{Key: "comments", Columns: []drivers.Column{{Name: "id"}, {Name: "video_id"}}},
			{Key: "profiles", Columns: []drivers.Column{{Name: "id"}}},
		},
		Aliases: drivers.Aliases{
			"users": {
				UpPlural:     "Users",
				UpSingular:   "User",
				DownSingular: "user",
				Columns: map[string]string{
					"id":         "ID",
					"profile_id": "ProfileID",
				},
				Relationships: map[string]string{
					"user_videos":  "Videos",
					"user_profile": "Profile",
				},
			},
			"videos": {
				UpPlural:     "Videos",
				UpSingular:   "Video",
				DownPlural:   "videos",
				DownSingular: "video",
				Columns: map[string]string{
					"id":      "ID",
					"user_id": "UserID",
				},
				Relationships: map[string]string{
					"video_comments": "Comments",
				},
			},
			"comments": {
				UpPlural:     "Comments",
				UpSingular:   "Comment",
				DownPlural:   "comments",
				DownSingular: "comment",
			},
			"profiles": {
				UpPlural:     "Profiles",
				UpSingular:   "Profile",
				DownPlural:   "profiles",
				DownSingular: "profile",
				Columns: map[string]string{
					"id": "ID",
				},
			},
		},
		Relationships: Relationships{
			"users":  {profileRel, videoRel},
			"videos": {commentRel},
		},
		ModelSplit: &ModelSplitData{
			Enabled:          true,
			Generation:       modelSplitGenerationComponent,
			Components:       []*ModelSplitComponent{{ID: "users", Package: "cusers", PackagePath: "example.com/models/internal/components/cusers", TableKeys: []string{"users"}}, {ID: "videos", Package: "cvideos", PackagePath: "example.com/models/internal/components/cvideos", TableKeys: []string{"videos"}}},
			TableComponents:  map[string]*ModelSplitComponent{"users": {ID: "users", Package: "cusers", PackagePath: "example.com/models/internal/components/cusers", TableKeys: []string{"users"}}, "videos": {ID: "videos", Package: "cvideos", PackagePath: "example.com/models/internal/components/cvideos", TableKeys: []string{"videos"}}},
			CurrentComponent: &ModelSplitComponent{ID: "users", Package: "cusers", PackagePath: "example.com/models/internal/components/cusers", TableKeys: []string{"users"}},
		},
	}
}

func TestLoadersTemplateGeneratesFacadeExpandThenLoadMethods(t *testing.T) {
	content, err := fs.ReadFile(templates, "templates/loaders/bob_loaders.bob.go.tpl")
	if err != nil {
		t.Fatal(err)
	}

	tpl, err := template.New("loaders").
		Funcs(sprig.GenericFuncMap()).
		Funcs(templateFunctions).
		Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	data := expandDrivenThenLoadTemplateData()
	data.ModelSplit.Generation = modelSplitGenerationFacade
	data.ModelSplit.CurrentComponent = nil

	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		"User expandUserThenLoader[Q]",
		"type expandUserThenLoader[Q orm.Loadable] struct {",
		"cusers.UserThenLoader[Q]",
		"func (l expandUserThenLoader[Q]) ForExpandMap(expands map[string]struct{}, opts ...ExpandLoadOption) ([]bob.Mod[Q], error)",
		"case \"videos\":",
		"childMods, err := SelectThenLoad.Video.forExpandTree(child, depth+1, opts)",
		"mods = append(mods, l.Videos(childMods...))",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected generated facade loaders to contain %q, got:\n%s", want, got)
		}
	}
}

func TestLoadersTemplateGeneratesFacadeExpandPreloadMethods(t *testing.T) {
	content, err := fs.ReadFile(templates, "templates/loaders/bob_loaders.bob.go.tpl")
	if err != nil {
		t.Fatal(err)
	}

	tpl, err := template.New("loaders").
		Funcs(sprig.GenericFuncMap()).
		Funcs(templateFunctions).
		Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	data := expandDrivenThenLoadTemplateData()
	data.ModelSplit.Generation = modelSplitGenerationFacade
	data.ModelSplit.CurrentComponent = nil

	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		"User expandUserPreloader",
		"type expandUserPreloader struct {",
		"cusers.UserPreloader",
		"func (l expandUserPreloader) ForExpandMap(expands map[string]struct{}, opts ...ExpandLoadOption) ([]bob.Mod[*dialect.SelectQuery], error)",
		"case \"profile\":",
		"mods = append(mods, l.Profile(append(childOpts, psql.PreloadAs(\"profile\"))...))",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected generated facade preloaders to contain %q, got:\n%s", want, got)
		}
	}
}

func TestLoadersTemplateUsesAllTablesForCrossComponentLoadStitching(t *testing.T) {
	content, err := fs.ReadFile(templates, "templates/loaders/table/110_loaders.go.tpl")
	if err != nil {
		t.Fatal(err)
	}

	tpl, err := template.New("loaders").
		Funcs(sprig.GenericFuncMap()).
		Funcs(templateFunctions).
		Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	data := expandDrivenThenLoadTemplateData()
	data.Tables = drivers.Tables[any, any]{data.AllTables[0]}

	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}
}

func TestLoadersTemplateGeneratesExpandDrivenThenLoadMethods(t *testing.T) {
	content, err := fs.ReadFile(templates, "templates/loaders/table/110_loaders.go.tpl")
	if err != nil {
		t.Fatal(err)
	}

	tpl, err := template.New("loaders").
		Funcs(sprig.GenericFuncMap()).
		Funcs(templateFunctions).
		Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	data := expandDrivenThenLoadTemplateData()

	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		"func (l UserThenLoader[Q]) ForExpandMap(expands map[string]struct{}, opts ...ExpandLoadOption) ([]bob.Mod[Q], error)",
		"func (l UserThenLoader[Q]) ForExpandPaths(paths []string, opts ...ExpandLoadOption) ([]bob.Mod[Q], error)",
		"case \"profile\":",
		"if len(child.children) > 0 {",
		"mods = append(mods, l.Profile())",
		"case \"videos\":",
		"expand path %q cannot be nested because Video is generated in another model component",
		"mods = append(mods, l.Videos())",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected generated loaders to contain %q, got:\n%s", want, got)
		}
	}

	for _, unwanted := range []string{
		"cvideos.SelectThenLoad",
		"example.com/models/internal/components/cvideos",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("expected generated loaders not to contain cross-component reference %q, got:\n%s", unwanted, got)
		}
	}
}

func TestLoadersTemplateGeneratesExpandDrivenPreloadMethods(t *testing.T) {
	content, err := fs.ReadFile(templates, "templates/loaders/table/110_loaders.go.tpl")
	if err != nil {
		t.Fatal(err)
	}

	tpl, err := template.New("loaders").
		Funcs(sprig.GenericFuncMap()).
		Funcs(templateFunctions).
		Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	data := expandDrivenThenLoadTemplateData()

	var out bytes.Buffer
	if err := tpl.Execute(&out, &data); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		"func (l UserPreloader) ForExpandMap(expands map[string]struct{}, opts ...ExpandLoadOption) ([]bob.Mod[*dialect.SelectQuery], error)",
		"func (l UserPreloader) ForExpandPaths(paths []string, opts ...ExpandLoadOption) ([]bob.Mod[*dialect.SelectQuery], error)",
		"case \"profile\":",
		"mods = append(mods, l.Profile(append(childOpts, psql.PreloadAs(\"profile\"))...))",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected generated preloaders to contain %q, got:\n%s", want, got)
		}
	}

	for _, unwanted := range []string{
		"cvideos.Preload",
		"example.com/models/internal/components/cvideos",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("expected generated preloaders not to contain cross-component reference %q, got:\n%s", unwanted, got)
		}
	}
}
