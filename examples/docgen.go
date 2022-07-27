package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stephenafamo/bob/dialect"
)

var rgxTabs = regexp.MustCompile(`\t`)
var rgxLeadingSpaces = regexp.MustCompile(`^\s+`)

func main() {
	var base = "./dialect"

	dialects, err := os.ReadDir(base)
	if err != nil {
		panic(err)
	}

	for _, d := range dialects {
		if !d.IsDir() {
			continue
		}
		path := filepath.Join(base, d.Name())
		genDialect(path)
	}
}

func genDialect(path string) {
	fset := token.NewFileSet() // positions are relative to fset

	// Parse src but stop after processing the imports.
	dir, err := parser.ParseDir(fset, path, func(fi fs.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.AllErrors)
	if err != nil {
		fmt.Println(err)
		return
	}

	for dirName, pkg := range dir {
		fmt.Printf("Dirname: %s\n", dirName)
		for fileName, f := range pkg.Files {
			fmt.Printf("filename: %s\n", fileName)
			fileName = strings.ReplaceAll(fileName, "dialect", "examples")
			fileName = strings.ReplaceAll(fileName, "_test.go", ".md")
			ast.Walk(wrapVisitor{next: &topVisitor{fset: fset, destination: fileName}}, f)
		}
	}
}

// To skip the top level
type wrapVisitor struct{ next ast.Visitor }

func (w wrapVisitor) Visit(n ast.Node) ast.Visitor {
	return w.next
}

type topVisitor struct {
	fset          *token.FileSet
	destination   string
	foundFunction bool
	blockFound    bool
}

func (t *topVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch stmt := n.(type) {
	case *ast.FuncDecl:
		t.foundFunction = true
		return t

	case *ast.BlockStmt:
		if t.foundFunction {
			t.blockFound = true
			return t
		}

		return nil

	case *ast.DeclStmt:
		if !t.blockFound {
			return nil
		}

		decl, ok := stmt.Decl.(*ast.GenDecl)
		if !ok {
			return nil
		}

		if !t.foundFunction {
			return nil
		}

		// Not a variable declaration
		if decl.Tok != token.VAR {
			return nil
		}

		// A multi declaration
		if decl.Lparen.IsValid() {
			return nil
		}

		return wrapVisitor{next: varVisitor{fset: t.fset, destination: t.destination}}
	default:
		return nil
	}
}

type varVisitor struct {
	fset        *token.FileSet
	destination string
}

func (v varVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	var buf = bytes.NewBuffer(nil)

	x := n.(*ast.ValueSpec)
	name := x.Names[0]

	printer.Fprint(buf, v.fset, name)
	var nameStr = buf.String()
	buf.Reset()
	if nameStr != "examples" {
		return nil
	}

	value, isComposite := x.Values[0].(*ast.CompositeLit)
	if !isComposite {
		return nil
	}

	printer.Fprint(buf, v.fset, value.Type)
	var typeStr = buf.String()
	_ = typeStr

	if _, final, _ := strings.Cut(nameStr, "."); final == "Testcases" {
		return nil
	}

	return &valueVisitor{fset: v.fset, destination: v.destination}
}

type valueVisitor struct {
	fset        *token.FileSet
	destination string
	name        string
	cases       []testcase
}

func (c *valueVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		toMarkdown(c.destination, c.cases)
		return nil
	}

	switch n.(type) {
	case *ast.KeyValueExpr:
		visitor := &casesVisitor{
			destination: c.destination,
			fset:        c.fset,
		}
		ast.Walk(wrapVisitor{next: visitor}, n)
		c.cases = append(c.cases, visitor.cases...)
		return nil
	default:
		// Keep returning self till you see a KeyValueExpr
		return c
	}
}

type testcase struct {
	name    string
	doc     string
	query   string
	builder string
	args    []any
}

type casesVisitor struct {
	fset        *token.FileSet
	destination string
	name        string
	cases       []testcase
}

func (c *casesVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch x := n.(type) {
	case *ast.BasicLit:
		c.name = x.Value[1 : len(x.Value)-1]
		return nil
	case *ast.CompositeLit:
		visitor := &caseVisitor{
			fset: c.fset,
		}
		ast.Walk(wrapVisitor{next: visitor}, x)
		tc := visitor.testcase
		tc.name = c.name
		c.cases = append(c.cases, tc)
		return nil
	default:
		return nil
	}
}

type caseVisitor struct {
	fset *token.FileSet
	testcase
}

func (c *caseVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	kv, ok := n.(*ast.KeyValueExpr)
	if !ok {
		return nil
	}

	var buf = bytes.NewBuffer(nil)

	printer.Fprint(buf, c.fset, kv.Key)
	var key = buf.String()
	buf.Reset()

	printer.Fprint(buf, c.fset, kv.Value)
	var val = buf.String()

	switch key {
	case "Doc":
		c.doc = dialect.Clean(val[1 : len(val)-1])
	case "Query":
		c.builder = rgxTabs.ReplaceAllLiteralString(val, "  ")
	case "ExpectedSQL":
		c.query = reindent(rgxTabs.ReplaceAllLiteralString(val[1:len(val)-1], "  "))
	case "ExpectedArgs":
		visitor := &argVisitor{fset: c.fset}
		ast.Walk(wrapVisitor{next: visitor}, kv.Value)
		c.args = visitor.args
	}

	return nil
}

type argVisitor struct {
	count int
	args  []any
	fset  *token.FileSet
}

func (a *argVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	a.count++
	if a.count < 2 {
		return nil
	}

	var buf = bytes.NewBuffer(nil)
	printer.Fprint(buf, a.fset, n)
	raw := buf.String()

	a.args = append(a.args, raw)

	return nil
}

func toMarkdown(destination string, cases []testcase) {
	if len(cases) < 1 {
		return
	}

	if destination == "" {
		panic("markdown destination is not set")
	}

	err := os.MkdirAll(filepath.Dir(destination), 0755)
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBuffer(nil)
	for index, c := range cases {
		if index > 0 {
			fmt.Fprint(buf, "\n")
		}

		if c.doc == "" {
			c.doc = strings.Title(c.name)
		}
		// write the sql query
		fmt.Fprintf(buf, "## %s\n\nSQL:\n\n```sql\n%s\n```\n\n", c.doc, c.query)

		if len(c.args) > 0 {
			fmt.Fprintf(buf, "Args:\n\n")
			for _, arg := range c.args {
				fmt.Fprintf(buf, "* `%s`\n", arg)
			}
			fmt.Fprintf(buf, "\n")
		}
		// write the go query
		fmt.Fprintf(buf, "Code:\n\n```go\n%s\n```\n", c.builder)
	}

	file, err := os.Create(destination)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	fmt.Printf("writing to %s\n", destination)

	if _, err = io.Copy(file, buf); err != nil {
		panic(err)
	}
}

func reindent(s string) string {
	var minLead []byte
	var firstline = true

	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		if firstline {
			firstline = false
			continue
		}
		leading := rgxLeadingSpaces.FindString(scanner.Text())

		if minLead == nil || len(leading) < len(minLead) {
			minLead = []byte(leading)
		}
	}

	if scanner.Err() != nil {
		panic(scanner.Err())
	}

	reindented := &strings.Builder{}

	firstline = true
	scanner = bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		if firstline {
			firstline = false
		} else {
			reindented.WriteString("\n")
		}

		_, err := reindented.WriteString(strings.TrimPrefix(scanner.Text(), string(minLead)))
		if err != nil {
			panic(err)
		}
	}
	if scanner.Err() != nil {
		panic(scanner.Err())
	}

	return reindented.String()
}
