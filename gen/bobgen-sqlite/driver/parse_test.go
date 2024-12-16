package driver

import (
	"fmt"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
)

func (p Parser) debugParse(s string) error {
	v := &visitor{
		db:        p.db,
		exprs:     make(map[nodeKey]exprInfo),
		names:     make(map[nodeKey]exprName),
		functions: defaultFunctions,
	}
	input := antlr.NewInputStream(s)

	infos, err := p.parse(v, input)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	for _, info := range infos {
		stmtStart := info.stmt.GetStart().GetStart()
		stmtStop := info.stmt.GetStop().GetStop()
		formatted, err := internal.EditStringSegment(s, stmtStart, stmtStop, info.editRules...)
		if err != nil {
			return fmt.Errorf("format: %w", err)
		}

		fmt.Printf("\n\nName: %s\n", info.comment)
		fmt.Printf("Query:\n%s\n\n", formatted)
		fmt.Printf("Cols:\n%s\n\n", info.columns.Print())
		fmt.Printf(
			"Args:\n%s\n\n",
			v.printExprs(input, stmtStart, stmtStop, info.args...),
		)
	}

	return nil
}

var db = tables{
	{
		Key:  "users",
		Name: "users",
		Columns: []drivers.Column{
			{Name: "id", Type: "int", DBType: "INTEGER"},
			{Name: "email", Type: "string", DBType: "TEXT"},
			{Name: "age", Type: "int", DBType: "INTEGER", Nullable: true},
		},
	},
	{
		Key:  "admins",
		Name: "admins",
		Columns: []drivers.Column{
			{Name: "id", Type: "int", DBType: "INTEGER"},
			{Name: "email", Type: "string", DBType: "TEXT", Nullable: true},
			{Name: "age", Type: "int", DBType: "INTEGER"},
		},
	},
	{
		Key:  "presales",
		Name: "presales",
		Columns: []drivers.Column{
			{Name: "id", Type: "int", DBType: "INTEGER"},
			{Name: "status", Type: "string", DBType: "TEXT", Nullable: true},
			{Name: "created_date", Type: "int", DBType: "INTEGER"},
			{Name: "presale_id", Type: "int", DBType: "INTEGER"},
		},
	},
}

func TestParsing(t *testing.T) {
	if err := newParser(db).debugParse(`-- this is a comment
	       -- FirstSelect
	       SELECT u.email, 'hello' as hello FROM admins u
	       WHERE ?1 IS NULL AND id = ?2 OR  age = ?3 OR (?5 - age) > id + cast(?4 as INTEGER);
	       -- SecondSelect
	       WITH u (id, email, age) AS (SELECT id, email, age FROM users)
	       SELECT * FROM u INNER JOIN users ON u.id = users.id WHERE id = ?1 AND email = ?2 OR age = ?3
	       ORDER BY u.id DESC, email ASC;
	           `,
	); err != nil {
		t.Error(err)
	}

	// if _, err := Parse(db, `
	//        SELECT hello.*, 1, 'string' FROM users
	//        WHERE id = ?
	//            AND "email" = @email
	//            OR name = @name
	//            AND int_with_underscore = 1_000
	//            AND big_int = 92233720368547758070
	//            AND hex_int = 0xFF
	//            AND big_hex = 0x80000000000000000
	//            AND float = 202E2
	//            AND floater = 2.002
	//            AND hello NOT REGEXP 'world'
	//            AND hi IS NOT DISTINCT FROM b`,
	// ); err != nil {
	// 	t.Error(err)
	// }
}
