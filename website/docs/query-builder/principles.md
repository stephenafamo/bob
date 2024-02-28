---

sidebar_position: 9
description: Custom Crafting & Progressive Enhancement 

---

# Principles

## 1. Custom Crafting

In `bob`, each dialect, and the applicable query mods are custom crafted to be as close to the specification as possible. This is unlike most other query builders that use a common structure and attempt to adapt it to every dialect.

## 2. Progressive enhancement

Most query mods will accept a literal string that will be printed as is.

However, many functions and methods are provided to express even the most complex queries in readable Go code.

```go
// Assuming we're building the following query
/*
SELECT status,
    LEAD(created_date, 1, NOW())
    OVER(PARTITION BY presale_id ORDER BY created_date) -
    created_date AS "difference"
FROM presales_presalestatus
*/

// different ways to express "SELECT status"
psql.Select(sm.Columns("status")) // SELECT status
psql.Select(sm.Columns(sm.Quote("status"))) // SELECT "status"

// Ways to express LEAD(created_date, 1, NOW())
"LEAD(created_date, 1, NOW()"
psql.F("LEAD", "created_date", 1, "NOW()")
psql.F("LEAD", "created_date", 1, sm.F("NOW"))

// Ways to express PARTITION BY presale_id ORDER BY created_date
"PARTITION BY presale_id ORDER BY created_date"
sm.Window("").PartitionBy("presale_id").OrderBy("created_date")

// Expressing LEAD(...) OVER(...)
"LEAD(created_date, 1, NOW()) OVER(PARTITION BY presale_id ORDER BY created_date)"
psql.F("LEAD", "created_date", 1, psql.F("NOW")).
    Over().
    PartitionBy("presale_id").
    OrderBy("created_date")

// The full query
psql.Select(
    sm.Columns(
        "status",
        psql.F("LEAD", "created_date", 1, psql.F("NOW")).
            Over().
            PartitionBy("presale_id").
            OrderBy("created_date").
            Minus("created_date").
            As("difference")),
    sm.From("presales_presalestatus")),
)
```
