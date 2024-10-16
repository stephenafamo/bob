package sm

import (
	"context"
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

type intoChain struct {
	into into
}

func (i *intoChain) Apply(q *dialect.SelectQuery) {
	q.SetInto(i.into)
}

func (i *intoChain) CharacterSet(c string) *intoChain {
	i.into.characterSet = c
	return i
}

func (i *intoChain) FieldsTerminatedBy(str string) *intoChain {
	i.into.hasFieldOpt = true
	i.into.fieldOptions.terminatedBy = str
	return i
}

func (i *intoChain) FieldsEnclosedBy(str string) *intoChain {
	i.into.hasFieldOpt = true
	i.into.fieldOptions.enclosedByOptional = false
	i.into.fieldOptions.enclosedBy = str
	return i
}

func (i *intoChain) FieldsOptionallyEnclosedBy(str string) *intoChain {
	i.into.hasFieldOpt = true
	i.into.fieldOptions.enclosedByOptional = true
	i.into.fieldOptions.enclosedBy = str
	return i
}

func (i *intoChain) FieldsEscapedBy(str string) *intoChain {
	i.into.hasFieldOpt = true
	i.into.fieldOptions.escapedBy = str
	return i
}

func (i *intoChain) LinesStartingBy(str string) *intoChain {
	i.into.hasLineOpt = true
	i.into.lineOptions.startingBy = str
	return i
}

func (i *intoChain) LinesTerminatedBy(str string) *intoChain {
	i.into.hasLineOpt = true
	i.into.lineOptions.terminatedBy = str
	return i
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

func (i into) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	// If it has vars, use INTO var_name, var_name ...
	if len(i.vars) > 0 {
		return bob.ExpressSlice(ctx, w, d, start, i.vars, "INTO @", ", @", "")
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

	_, err = bob.ExpressIf(ctx, w, d, start, i.characterSet,
		i.characterSet != "", "\nCHARACTER SET ", "")
	if err != nil {
		return nil, err
	}

	_, err = bob.ExpressIf(ctx, w, d, start, i.fieldOptions, i.hasFieldOpt, "\n", "")
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

func (f fieldOptions) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
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

func (l lineOptions) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	w.Write([]byte("LINES"))

	if l.startingBy != "" {
		fmt.Fprintf(w, "  STARTINGBY '%s'", l.terminatedBy)
	}

	if l.terminatedBy != "" {
		fmt.Fprintf(w, " TERMINATED BY '%s'", l.terminatedBy)
	}

	return nil, nil
}
