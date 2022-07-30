package mysql

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

type modifiers[T any] struct {
	modifiers []T
}

func (h *modifiers[T]) AppendModifier(modifier T) {
	h.modifiers = append(h.modifiers, modifier)
}

func (h modifiers[T]) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, h.modifiers, "", " ", "")
}

type set struct {
	col string
	val any
}

func (s set) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.Express(w, d, start, expr.OP("=", Quote(s.col), s.val))
}

type setMod[Q interface{ addSet(set) }] set

func (s setMod[Q]) Apply(q Q) {
	q.addSet(set(s))
}

type partitionMod[Q interface{ AppendPartition(...string) }] struct{}

func (p partitionMod[Q]) Partition(partitions ...string) query.Mod[Q] {
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendPartition(partitions...)
	})
}

type partitions struct {
	partitions []string
}

func (h *partitions) AppendPartition(partitions ...string) {
	h.partitions = append(h.partitions, partitions...)
}

func (h partitions) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, h.partitions, "PARTITION (", ", ", ")")
}

type intoMod[Q interface{ setInto(into) }] struct{}

// No need for the leading @
func (i intoMod[Q]) Into(var1 string, vars ...string) query.Mod[Q] {
	return mods.QueryModFunc[Q](func(q Q) {
		q.setInto(into{
			vars: append([]string{var1}, vars...),
		})
	})
}

func (i intoMod[Q]) IntoDumpfile(filename string) query.Mod[Q] {
	return mods.QueryModFunc[Q](func(q Q) {
		q.setInto(into{
			dumpfile: filename,
		})
	})
}

func (i intoMod[Q]) IntoOutfile(filename string) *intoChain[Q] {
	return &intoChain[Q]{
		into: into{
			outfile: filename,
		},
	}
}

type into struct {
	vars     []string
	dumpfile string
	outfile  string

	// OUTFILE options
	characterSet string
	hasFieldOpt  bool
	fieldOptions fieldOptions
	hasLineOpt   bool
	lineOptions  lineOptions
}

func (i into) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	// If it has vars, use INTO var_name, var_name ...
	if len(i.vars) > 0 {
		return query.ExpressSlice(w, d, start, i.vars, "INTO @", ", @", "")
	}

	// If dumpfile is present, use INTO DUMPFILE 'file_name'
	if i.dumpfile != "" {
		_, err := fmt.Fprintf(w, "INTO DUMPFILE '%s'", i.dumpfile)
		return nil, err
	}

	// If no outfile, return nothing
	if i.outfile == "" {
		return nil, nil
	}

	_, err := fmt.Fprintf(w, "INTO OUTFILE '%s'", i.dumpfile)
	if err != nil {
		return nil, err
	}

	_, err = query.ExpressIf(w, d, start, i.characterSet,
		i.characterSet != "", "\nCHARACTER SET ", "")
	if err != nil {
		return nil, err
	}

	_, err = query.ExpressIf(w, d, start, i.fieldOptions, i.hasFieldOpt, "\n", "")
	if err != nil {
		return nil, err
	}

	return nil, nil
}

type fieldOptions struct {
	terminatedBy       string
	escapedBy          string
	enclosedBy         string
	enclosedByOptional bool
}

func (f fieldOptions) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	w.Write([]byte("FIELDS"))

	if f.terminatedBy != "" {
		fmt.Fprintf(w, " TERMINATED BY '%s'", f.terminatedBy)
	}

	if f.enclosedBy != "" {
		if f.enclosedByOptional {
			w.Write([]byte(" OPTIONALLY"))
		}
		fmt.Fprintf(w, " ENCLOSED BY '%s'", f.enclosedBy)
	}

	if f.escapedBy != "" {
		fmt.Fprintf(w, " ESCAPED BY '%s'", f.escapedBy)
	}

	return nil, nil
}

type lineOptions struct {
	startingBy   string
	terminatedBy string
}

func (l lineOptions) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	w.Write([]byte("LINES"))

	if l.startingBy != "" {
		fmt.Fprintf(w, "  STARTINGBY '%s'", l.terminatedBy)
	}

	if l.terminatedBy != "" {
		fmt.Fprintf(w, " TERMINATED BY '%s'", l.terminatedBy)
	}

	return nil, nil
}

type intoChain[Q interface{ setInto(into) }] struct {
	into into
}

func (i *intoChain[Q]) Apply(q Q) {
	q.setInto(i.into)
}

func (i *intoChain[Q]) CharacterSet(c string) *intoChain[Q] {
	i.into.characterSet = c
	return i
}

func (i *intoChain[Q]) FieldsTerminatedBy(str string) *intoChain[Q] {
	i.into.hasFieldOpt = true
	i.into.fieldOptions.terminatedBy = str
	return i
}

func (i *intoChain[Q]) FieldsEnclosedBy(str string) *intoChain[Q] {
	i.into.hasFieldOpt = true
	i.into.fieldOptions.enclosedByOptional = false
	i.into.fieldOptions.enclosedBy = str
	return i
}

func (i *intoChain[Q]) FieldsOptionallyEnclosedBy(str string) *intoChain[Q] {
	i.into.hasFieldOpt = true
	i.into.fieldOptions.enclosedByOptional = true
	i.into.fieldOptions.enclosedBy = str
	return i
}

func (i *intoChain[Q]) FieldsEscapedBy(str string) *intoChain[Q] {
	i.into.hasFieldOpt = true
	i.into.fieldOptions.escapedBy = str
	return i
}

func (i *intoChain[Q]) LinesStartingBy(str string) *intoChain[Q] {
	i.into.hasLineOpt = true
	i.into.lineOptions.startingBy = str
	return i
}

func (i *intoChain[Q]) LinesTerminatedBy(str string) *intoChain[Q] {
	i.into.hasLineOpt = true
	i.into.lineOptions.terminatedBy = str
	return i
}
