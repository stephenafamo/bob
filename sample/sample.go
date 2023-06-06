package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/stephenafamo/bob"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

func main() {
	maindb()
}

func main1() {
	query := psql.Select(
		sm.Columns("id", "name"),
		sm.From("users"),
		sm.Where(psql.Quote("id").In(psql.ArgNamed("in1", "in2", "in3"))),
		sm.Where(psql.Raw("id >= ?", psql.NamedArg("id1"))),
	)

	queryStr, args, err := bob.BuildWithNamedArgs(query, map[string]any{
		"in1": 15,
		"in2": 200,
		"in3": 300,
		"x":   "abc",
		"y":   "h",
		"id1": 400,
	})

	// prepared, err := query.BuildPrepared()
	// if err != nil {
	// 	panic(err)
	// }
	//
	//
	// args, err := prepared.Build(map[string]any{
	// 	"in1": 15,
	// 	"in2": 200,
	// 	"in3": 300,
	// 	"x":   "abc",
	// 	"y":   "h",
	// 	"id1": 400,
	// })
	if err != nil {
		panic(err)
	}

	fmt.Println(queryStr)
	fmt.Println(args)

	// SELECT
	// id, name
	// FROM users
	// WHERE ("id" IN ($1, $2, $3)) AND (id >= $4)
	//
	// [15 200 300 400]
}

func main2() {
	query := psql.Insert(
		im.Into("actor", "first_name", "last_name"),
		// im.Values(psql.Arg(psql.NamedArg("in1"), psql.NamedArg("in2"))),
		im.Values(psql.ArgNamed("in1", "in2")),
	)

	queryStr, args, err := bob.BuildWithNamedArgs(query, map[string]any{
		"in1": 15,
		"in2": "LAST_NAME",
	})

	// prepared, err := query.BuildPrepared()
	// if err != nil {
	// 	panic(err)
	// }
	//
	// args, err := prepared.Build(map[string]any{
	// 	"in1": 15,
	// 	"in2": "LAST_NAME",
	// })
	if err != nil {
		panic(err)
	}

	fmt.Println(queryStr)
	fmt.Println(args)

	// INSERT INTO actor ("first_name", "last_name")
	// VALUES ($1, $2)
	//
	// [15 LAST_NAME]
}

type Main3Args struct {
	FirstName string
	LastName  string
}

// func main3() {
// 	query := psql.Insert(
// 		im.Into("actor", "first_name", "last_name"),
// 		im.Values(psql.ArgNamed("first_name", "last_name")),
// 	)
//
// 	prepared, err := query.BuildPrepared()
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	// struct parameters not working yet
// 	args, err := prepared.Build(Main3Args{
// 		FirstName: "JOHN",
// 		LastName:  "CENA",
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(prepared.SQL())
// 	fmt.Println(args)
//
// 	// INSERT INTO actor ("first_name", "last_name")
// 	// VALUES ($1, $2)
// 	//
// 	// [JOHN CENA]
// }

func maindb() {
	db, err := sql.Open("pgx",
		fmt.Sprintf("postgres://postgres:password@%s:%s/%s?sslmode=disable", "localhost", "5478", "sakila"))
	if err != nil {
		panic(err)
	}

	bdb := bob.NewDB(db)

	type Data struct {
		FirstName string
		LastName  string
	}

	dataMapper := scan.StructMapper[Data]()

	for _, items := range [][2]int{{0, 4}, {2, 4}, {50, 12}} {
		fmt.Printf("%s OFFSET %d LIMIT %d %s\n", strings.Repeat("=", 10), items[0], items[1], strings.Repeat("=", 10))

		query := psql.Select(
			sm.Columns("first_name", "last_name"),
			sm.From("actor"),
			sm.OrderBy("first_name"),
			sm.OrderBy("last_name"),
			sm.Offset(psql.ArgNamed("offset")),
			sm.Limit(psql.ArgNamed("limit")),
		)

		// prepared, err := query.BuildPrepared()
		// if err != nil {
		// 	panic(err)
		// }

		// stmt, err := bdb.PrepareContext(context.Background(), prepared.SQL())

		data, err := bob.All(context.Background(), bdb, bob.QueryWithNamedArgs(query, map[string]any{
			"offset": items[0],
			"limit":  items[1],
		}), dataMapper)
		if err != nil {
			panic(err)
		}

		fmt.Println(data)
	}

}
