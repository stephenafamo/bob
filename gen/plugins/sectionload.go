package plugins

import (
	"bufio"
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
)

// sectionloadRuntimePkg is the import path of the sectionload runtime once
// vendored into bob itself. Unlike modelsPkg (resolved per-run from
// data.OutputPackages, since the destination is user-configured), this path
// is fixed: the runtime always lives at this location inside the bob module.
const sectionloadRuntimePkg = "github.com/stephenafamo/bob/dialect/psql/sectionload"

// Sectionload registers the "dbsections" output: generated code that composes
// several relationships into a single UNION ALL query (see
// dialect/psql/sectionload for the runtime and
// gen/bobgen-psql/templates/sectionload for the templates it renders).
//
// sectionload is PostgreSQL-only. Callers on other dialects pass an fs.FS
// with no matching "templates/sectionload" directory (see
// gen.Templates.Sectionload); PlugState detects that case and registers no
// output at all, so bobgen-mysql and bobgen-sqlite runs are unaffected.
func Sectionload[T, C, I any](config SectionloadConfig, templates ...fs.FS) gen.Plugin {
	config.OutputConfig = config.OutputConfig.WithDefaults("dbsections")
	return &sectionloadPlugin[T, C, I]{
		config:    config,
		templates: templates,
	}
}

// SectionloadConfig is the "sectionload" section of bob.yaml's plugins
// config. It embeds OutputConfig (disabled/destination/pkgname, same as the
// other output plugins) and adds the table exclusion list: tables with
// SELECT-time hooks registered on their bob model must not get sectionload
// parts, because sections are composed as raw SQL and never pass through
// <Table>.Query(), so those hooks would silently stop firing.
//
// Unlike the other outputs, sectionload defaults to disabled: it is a new,
// optional output, and existing bob.yaml files must keep generating exactly
// what they generated before upgrading, with no new "dbsections" package
// appearing unannounced. This is why it carries its own explicit opt-in
// (Enabled), instead of reusing OutputConfig.Disabled the way every other
// output does (those default to *enabled*, matching bob's pre-existing
// behavior for them).
type SectionloadConfig struct {
	OutputConfig `yaml:",squash"`

	// Enabled must be explicitly set to true to turn this output on. Unset
	// (nil) or false means "generate nothing", regardless of Disabled.
	Enabled *bool `yaml:"enabled"`

	// ExcludeFile is the path (relative to the current directory) of a file
	// listing excluded tables, one per line, "#" starts a comment. A
	// missing file is not an error: the list is app-specific and this
	// plugin is generic.
	ExcludeFile string `yaml:"exclude_file"`
	// Exclude is a comma-separated list of excluded tables, merged with
	// ExcludeFile.
	Exclude string `yaml:"exclude"`
}

func mergeSectionloadConfig(c1, c2 SectionloadConfig) SectionloadConfig {
	return SectionloadConfig{
		OutputConfig: mergeOutputConfig(c1.OutputConfig, c2.OutputConfig),
		Enabled:      cmp.Or(c2.Enabled, c1.Enabled),
		ExcludeFile:  cmp.Or(c2.ExcludeFile, c1.ExcludeFile),
		Exclude:      cmp.Or(c2.Exclude, c1.Exclude),
	}
}

type sectionloadPlugin[T, C, I any] struct {
	config    SectionloadConfig
	templates []fs.FS

	// active reports whether PlugState actually registered the "dbsections"
	// output (config enabled AND matching templates present). It gates
	// PlugTemplateData, which otherwise runs unconditionally for every
	// TemplateDataPlugin regardless of what PlugState decided.
	active bool

	// excludes is the set of tables that must not get parts, loaded once in
	// PlugState from config.ExcludeFile/config.Exclude.
	excludes sectionloadExcludeSet
	// probes is filled in PlugTemplateData from the FULL table list (before
	// exclusion filtering), so the generated hook-probe table can also name
	// the excluded tables. Read back by the "hookProbes" template func.
	probes []sectionloadHookProbe
	// modelsPkg is filled in PlugTemplateData (data.OutputPackages is only
	// populated right before TemplateDataPlugins run) and read back by the
	// "modelsPkg" template func, which PlugState registers as a closure
	// over this field.
	modelsPkg string
}

// sectionloadHookProbe is one row of the generated hook-probe table: a table
// name, the bob view variable to inspect at runtime, and whether sectionload
// parts were generated for it.
type sectionloadHookProbe struct {
	Table    string
	ViewVar  string
	Excluded bool
}

var (
	_ gen.StatePlugin[any]                  = (*sectionloadPlugin[any, any, any])(nil)
	_ gen.TemplateDataPlugin[any, any, any] = (*sectionloadPlugin[any, any, any])(nil)
)

// Name implements gen.StatePlugin.
func (*sectionloadPlugin[T, C, I]) Name() string {
	return "Sectionload Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (p *sectionloadPlugin[T, C, I]) PlugState(state *gen.State[C]) error {
	if !internal.ValOrZero(p.config.Enabled) || internal.ValOrZero(p.config.Disabled) {
		// Opt-in: unset/false Enabled means "generate nothing", regardless
		// of Disabled. See SectionloadConfig doc comment.
		return nil
	}
	if err := dependsOn(p.config.Disabled, state, "models"); err != nil {
		return err
	}

	if !p.hasUsableTemplates() {
		// mysql/sqlite: no "templates/sectionload" directory for this
		// dialect. Register nothing rather than an output with zero
		// templates (which would otherwise emit empty generated files for
		// every table).
		return nil
	}

	excludes, err := loadSectionloadExcludes(p.config.ExcludeFile, p.config.Exclude)
	if err != nil {
		return fmt.Errorf("loading sectionload excludes: %w", err)
	}
	p.excludes = excludes
	p.active = true

	if state.CustomTemplateFuncs == nil {
		state.CustomTemplateFuncs = template.FuncMap{}
	}
	// modelsPkg is resolved once PlugTemplateData runs (it reads
	// data.OutputPackages, which gen.Run only populates right before
	// invoking TemplateDataPlugins). sectionloadPkg is a fixed path so it
	// can be returned directly. Both are read by the templates via
	// {{modelsPkg}} / {{sectionloadPkg}} instead of bob's usual
	// {{index $.OutputPackages "models"}}, to keep the templates identical
	// to the ones vetted standalone (see MAPPING.md).
	state.CustomTemplateFuncs["modelsPkg"] = func() string { return p.modelsPkg }
	state.CustomTemplateFuncs["sectionloadPkg"] = func() string { return sectionloadRuntimePkg }
	state.CustomTemplateFuncs["hookProbes"] = func() []sectionloadHookProbe { return p.probes }

	state.Outputs = append(state.Outputs, &gen.Output{
		Key:       "dbsections",
		OutFolder: p.config.Destination,
		PkgName:   p.config.Pkgname,
		Templates: p.templates,
	})

	return nil
}

// hasUsableTemplates reports whether any of the supplied template FSes has
// at least one entry. fs.Sub never fails just because the target directory
// doesn't exist (see gen/templates.go); the failure only surfaces once you
// actually try to read the FS, as fs.ErrNotExist.
func (p *sectionloadPlugin[T, C, I]) hasUsableTemplates() bool {
	for _, t := range p.templates {
		if t == nil {
			continue
		}
		entries, err := fs.ReadDir(t, ".")
		if err == nil && len(entries) > 0 {
			return true
		}
	}
	return false
}

// PlugTemplateData implements gen.TemplateDataPlugin.
func (p *sectionloadPlugin[T, C, I]) PlugTemplateData(data *gen.TemplateData[T, C, I]) error {
	if !p.active {
		return nil
	}

	p.modelsPkg = data.OutputPackages["models"]

	// Record every table (including the excluded ones) for the hook-probe
	// table, then remove the excluded tables from the data the templates
	// see. bob iterates data.Tables to decide which per-table files to
	// write, so dropping a table here means no parts file is produced for
	// it.
	p.probes = make([]sectionloadHookProbe, 0, len(data.Tables))
	kept := make([]drivers.Table[C, I], 0, len(data.Tables))
	for _, t := range data.Tables {
		excluded := p.excludes.has(t.Key)
		p.probes = append(p.probes, sectionloadHookProbe{
			Table:    t.Key,
			ViewVar:  data.Aliases.Table(t.Key).UpPlural,
			Excluded: excluded,
		})
		if !excluded {
			kept = append(kept, t)
		}
	}
	data.Tables = kept

	// Drop every relationship that touches an excluded table, from either
	// end. Without this, a surviving table would emit an edge whose Child
	// parts no longer exist (a build failure in the generated package).
	rels := make(gen.Relationships, len(data.Relationships))
	for table, list := range data.Relationships {
		if p.excludes.has(table) {
			continue
		}
		keptRels := make([]orm.Relationship, 0, len(list))
		for _, r := range list {
			if p.relTouchesExcluded(r) {
				continue
			}
			keptRels = append(keptRels, r)
		}
		if len(keptRels) > 0 {
			rels[table] = keptRels
		}
	}
	data.Relationships = rels

	return nil
}

// relTouchesExcluded reports whether any hop of the relationship starts or
// ends at an excluded table.
func (p *sectionloadPlugin[T, C, I]) relTouchesExcluded(r orm.Relationship) bool {
	for _, side := range r.Sides {
		if p.excludes.has(side.From) || p.excludes.has(side.To) {
			return true
		}
	}
	return false
}

// sectionloadExcludeSet is the set of tables that must NOT get sectionload
// parts. See SectionloadConfig for why such a list exists at all.
type sectionloadExcludeSet map[string]struct{}

func (s sectionloadExcludeSet) has(tableKey string) bool {
	if len(s) == 0 {
		return false
	}
	_, ok := s[normalizeSectionloadTableKey(tableKey)]
	return ok
}

// normalizeSectionloadTableKey makes the driver's table key and a
// human-written list entry comparable. The psql driver leaves the schema
// empty for the default schema, so "public.foo" and "foo" must match.
func normalizeSectionloadTableKey(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return strings.TrimPrefix(s, "public.")
}

// loadSectionloadExcludes reads the exclusion list from a file (one table
// per line, "#" starts a comment) and merges the comma-separated inline
// list. A missing file is not an error.
func loadSectionloadExcludes(path, inline string) (sectionloadExcludeSet, error) {
	set := sectionloadExcludeSet{}

	if path != "" {
		f, err := os.Open(path)
		switch {
		case err == nil:
			defer f.Close()
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := sc.Text()
				if i := strings.IndexByte(line, '#'); i >= 0 {
					line = line[:i]
				}
				if line = strings.TrimSpace(line); line == "" {
					continue
				}
				set[normalizeSectionloadTableKey(line)] = struct{}{}
			}
			if err := sc.Err(); err != nil {
				return nil, fmt.Errorf("reading %s: %w", path, err)
			}
		case os.IsNotExist(err):
			// No list for this app: generate everything.
		default:
			return nil, fmt.Errorf("opening %s: %w", path, err)
		}
	}

	for _, name := range strings.Split(inline, ",") {
		if n := normalizeSectionloadTableKey(name); n != "" {
			set[n] = struct{}{}
		}
	}
	return set, nil
}
