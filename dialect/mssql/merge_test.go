package mssql_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/dialect/mssql"
	"github.com/stephenafamo/bob/dialect/mssql/mm"
	"github.com/stephenafamo/bob/dialect/mssql/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestMerge(t *testing.T) {
	examples := testutils.Testcases{
		"simple upsert": {
			Doc: "Basic MERGE with WHEN MATCHED UPDATE and WHEN NOT MATCHED INSERT",
			Query: mssql.Merge(
				mm.Into("target", "t"),
				mm.Using(mssql.Quote("source"), "s"),
				mm.OnEQ(mssql.Quote("t", "id"), mssql.Quote("s", "id")),
				mm.WhenMatched().ThenUpdate(
					mssql.Raw("[t].[name] = [s].[name]"),
					mssql.Raw("[t].[value] = [s].[value]"),
				),
				mm.WhenNotMatched().ThenInsert(
					[]string{"id", "name", "value"},
					mssql.Quote("s", "id"),
					mssql.Quote("s", "name"),
					mssql.Quote("s", "value"),
				),
			),
			ExpectedSQL: `MERGE INTO target AS [t]
				USING [source] AS [s]
				ON ([t].[id] = [s].[id])
				WHEN MATCHED THEN
				UPDATE SET
				[t].[name] = [s].[name],
				[t].[value] = [s].[value]
				WHEN NOT MATCHED THEN
				INSERT ([id], [name], [value]) VALUES ([s].[id], [s].[name], [s].[value]);`,
		},
		"merge with delete": {
			Doc: "MERGE with all three WHEN clauses including DELETE",
			Query: mssql.Merge(
				mm.Into("inventory", "inv"),
				mm.Using(mssql.Quote("shipments"), "s"),
				mm.OnEQ(mssql.Quote("inv", "product_id"), mssql.Quote("s", "product_id")),
				mm.WhenMatched().ThenUpdate(
					mssql.Raw("[inv].[quantity] = [inv].[quantity] + [s].[quantity]"),
				),
				mm.WhenNotMatched().ThenInsert(
					[]string{"product_id", "quantity"},
					mssql.Quote("s", "product_id"),
					mssql.Quote("s", "quantity"),
				),
				mm.WhenNotMatchedBySource().ThenDelete(),
			),
			ExpectedSQL: `MERGE INTO inventory AS [inv]
				USING [shipments] AS [s]
				ON ([inv].[product_id] = [s].[product_id])
				WHEN MATCHED THEN
				UPDATE SET
				[inv].[quantity] = [inv].[quantity] + [s].[quantity]
				WHEN NOT MATCHED THEN
				INSERT ([product_id], [quantity]) VALUES ([s].[product_id], [s].[quantity])
				WHEN NOT MATCHED BY SOURCE THEN
				DELETE;`,
		},
		"merge with output": {
			Doc: "MERGE with OUTPUT clause",
			Query: mssql.Merge(
				mm.Into("employees", "e"),
				mm.Using(mssql.Quote("new_employees"), "n"),
				mm.OnEQ(mssql.Quote("e", "emp_id"), mssql.Quote("n", "emp_id")),
				mm.WhenMatched().ThenUpdate(
					mssql.Raw("[e].[name] = [n].[name]"),
				),
				mm.WhenNotMatched().ThenInsert(
					[]string{"emp_id", "name"},
					mssql.Quote("n", "emp_id"),
					mssql.Quote("n", "name"),
				),
				mm.Output("$action", "inserted.emp_id", "deleted.emp_id"),
			),
			ExpectedSQL: `MERGE INTO employees AS [e]
				USING [new_employees] AS [n]
				ON ([e].[emp_id] = [n].[emp_id])
				WHEN MATCHED THEN
				UPDATE SET
				[e].[name] = [n].[name]
				WHEN NOT MATCHED THEN
				INSERT ([emp_id], [name]) VALUES ([n].[emp_id], [n].[name])
				OUTPUT $action, inserted.emp_id, deleted.emp_id;`,
		},
		"merge with conditional when": {
			Doc: "WHEN MATCHED with additional AND condition",
			Query: mssql.Merge(
				mm.Into("prices", "p"),
				mm.Using(mssql.Quote("updates"), "u"),
				mm.OnEQ(mssql.Quote("p", "sku"), mssql.Quote("u", "sku")),
				mm.WhenMatched(mssql.Quote("u", "price").GT(mssql.Arg(0))).ThenUpdate(
					mssql.Raw("[p].[price] = [u].[price]"),
				),
				mm.WhenMatched().ThenDelete(),
			),
			ExpectedSQL: `MERGE INTO prices AS [p]
				USING [updates] AS [u]
				ON ([p].[sku] = [u].[sku])
				WHEN MATCHED AND ([u].[price] > @p1) THEN
				UPDATE SET
				[p].[price] = [u].[price]
				WHEN MATCHED THEN
				DELETE;`,
			ExpectedArgs: []any{0},
		},
		"merge using subquery": {
			Doc: "MERGE using a subquery as source",
			Query: mssql.Merge(
				mm.Into("target"),
				mm.Using(mssql.Select(
					sm.Columns("id", "val"),
					sm.From("staging"),
					sm.Where(mssql.Quote("active").EQ(mssql.Arg(true))),
				), "s"),
				mm.OnEQ(mssql.Quote("target", "id"), mssql.Quote("s", "id")),
				mm.WhenMatched().ThenUpdate(
					mssql.Raw("[target].[val] = [s].[val]"),
				),
			),
			ExpectedSQL: `MERGE INTO target
				USING (SELECT id, val FROM staging WHERE ([active] = @p1)) AS [s]
				ON ([target].[id] = [s].[id])
				WHEN MATCHED THEN
				UPDATE SET
				[target].[val] = [s].[val];`,
			ExpectedArgs: []any{true},
		},
		"merge with conditional not matched by source": {
			Doc: "WHEN NOT MATCHED BY SOURCE with an additional AND condition that filters which target rows the action applies to",
			Query: mssql.Merge(
				mm.Into("inventory", "inv"),
				mm.Using(mssql.Quote("shipments"), "s"),
				mm.OnEQ(mssql.Quote("inv", "product_id"), mssql.Quote("s", "product_id")),
				mm.WhenMatched().ThenUpdate(
					mssql.Raw("[inv].[quantity] = [inv].[quantity] + [s].[quantity]"),
				),
				mm.WhenNotMatchedBySource(mssql.Quote("inv", "quantity").GT(mssql.Arg(0))).ThenUpdate(
					mssql.Raw("[inv].[quantity] = 0"),
				),
				mm.WhenNotMatchedBySource(mssql.Quote("inv", "discontinued").EQ(mssql.Arg(true))).ThenDelete(),
			),
			ExpectedSQL: `MERGE INTO inventory AS [inv]
				USING [shipments] AS [s]
				ON ([inv].[product_id] = [s].[product_id])
				WHEN MATCHED THEN
				UPDATE SET
				[inv].[quantity] = [inv].[quantity] + [s].[quantity]
				WHEN NOT MATCHED BY SOURCE AND ([inv].[quantity] > @p1) THEN
				UPDATE SET
				[inv].[quantity] = 0
				WHEN NOT MATCHED BY SOURCE AND ([inv].[discontinued] = @p2) THEN
				DELETE;`,
			ExpectedArgs: []any{0, true},
		},
		"merge with cte": {
			Doc: "MERGE with CTE as source",
			Query: mssql.Merge(
				mm.With("src").As(mssql.Select(
					sm.Columns("id", "name"),
					sm.From("staging"),
				)),
				mm.Into("target", "t"),
				mm.Using(mssql.Quote("src"), "s"),
				mm.OnEQ(mssql.Quote("t", "id"), mssql.Quote("s", "id")),
				mm.WhenNotMatched().ThenInsert(
					[]string{"id", "name"},
					mssql.Quote("s", "id"),
					mssql.Quote("s", "name"),
				),
			),
			ExpectedSQL: `WITH [src] AS (SELECT id, name FROM staging)
				MERGE INTO target AS [t]
				USING [src] AS [s]
				ON ([t].[id] = [s].[id])
				WHEN NOT MATCHED THEN
				INSERT ([id], [name]) VALUES ([s].[id], [s].[name]);`,
		},
	}

	testutils.RunTests(t, examples, nil)
}

func TestMergeNoWhenClauses(t *testing.T) {
	q := mssql.Merge(
		mm.Into("target", "t"),
		mm.Using(mssql.Quote("source"), "s"),
		mm.OnEQ(mssql.Quote("t", "id"), mssql.Quote("s", "id")),
	)
	gotSQL, _, err := q.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	normalized := strings.Join(strings.Fields(gotSQL), " ")
	if !strings.Contains(normalized, "MERGE INTO target AS [t]") {
		t.Fatalf("expected MERGE INTO target AS [t], got: %s", normalized)
	}
	if !strings.Contains(normalized, "USING [source] AS [s]") {
		t.Fatalf("expected USING clause, got: %s", normalized)
	}
	if !strings.HasSuffix(strings.TrimSpace(gotSQL), ";") {
		t.Fatalf("expected trailing semicolon, got: %s", normalized)
	}
	if strings.Contains(normalized, "WHEN") {
		t.Fatal("expected no WHEN clauses in output")
	}
}
