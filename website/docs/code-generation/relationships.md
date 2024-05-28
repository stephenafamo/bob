---

sidebar_position: 5
description: Working with relationships

---

# Relationships

Related models are stored in the `R` field of the generated structs. For example, the related pilot of a jet will be `jet.R.Pilot`.

## Relationshp Types

All relationship types have similar usage. The main difference is that `to-one` relations are mapped to a single struct while `to-many` relations are mapped to a slice.

## Querying relationships

Querying related relationships start with a named method on the model.

```go
jet, err := models.FindJet(ctx, db, 1)

// SELECT * FROM "pilots" WHERE "id" = $1
// $1 => jet.PilotID
jetPilotQuery, err := jet.Pilots(ctx, db)
```

In the above code example, `jetPilotQuery` is a [TableQuery](../models/table#queries) and has access to all the expected finishers:

* One
* All
* Cursor
* Count
* Exists
* UpdateAll
* DeleteAll

Naturally, we can add mods to the query:

```go
jet, err := models.FindJet(ctx, db, 1)

// SELECT * FROM "pilots" WHERE "id" = $1 LIMIT 20
jetPilotQuery, err := jet.Pilots(ctx, db, sm.Limit(20))
```

## Modifying Relationships

* InsertXXX: This inserts a new row and sets it as the related model

    ```go
    // to-one
    jet.InsertPilot(ctx, db, &PilotSetter{...})

    // to-many
    pilot.InsertJets(ctx, db, &JetSetter{...}, &JetSetter{...})
    ```

* AttachXXX: This attaches an existing model as a relation

    ```go
    // to-one
    jet.AttachPilot(ctx, db, &Pilot{...})

    // to-many
    pilot.AttachJets(ctx, db, &Jet{...}, &Jet{...})
    ```

## Loading related models

Bob generates 2 ways to load models:

1. **Preload**: Load the relationship in **the same** query using a `LEFT JOIN`.
1. **ThenLoad**: Load the relationship in an additional query using all the primary keys of the first.

### Preload

:::note

At this time, Preload only works for `to-one` relationships.

:::

```go
models.PreloadJetPilot(opts ...psql.PreloadOption)
```

The mod function accepts options:

1. `OnlyColumns`: In the related model, load only these columns.
1. `ExceptColumns`: In the related model, do not load these columns.
1. `PreloadAs`: Explicitly sets the table alias for the related model to allow using columns of the related model for the query.
1. `Loaders`: Other loaders mods can be given as an option to the preloader to load nested relationships. This works for both other preloaders and then-loaders.

```go
jet, err := models.Jets(ctx, db, 
    models.PreloadJetPilot(
        psql.OnlyColumns("id"), // only selects "pilot"."id"
        psql.ThenLoadPilotLicences(), // will load the pilot's licences
    ),
).One()
```

```go
jets, err := models.Jets(ctx, db,
	models.PreloadJetPilot(psql.PreloadAs("pilot")), // "LEFT JOIN "pilots" AS "pilot" ON ("jet"."pilot_id" = "pilot"."id") 
	models.PreloadJetCoPilot(psql.PreloadAs("copilot")), // "LEFT JOIN "pilots" AS "copilot" ON ("jet"."copilot_id" = "copilot"."id") 
	sm.OrderBy(psql.Quote("pilot", models.ColumnNames.Pilot.LastName)) // ORDER BY "pilot"."last_name" DESC 
).All()
```

### ThenLoad

```go
models.ThenLoadPilotJets(...mods)
```

These will accept **ANY** `SelectQuery` mods.

```go
// get the first 2 pilots
// then load all related jets with airport_id = 100
pilots, err := models.Pilots(ctx, db, 
    models.ThenLoadPilotJets(
        models.SelectWhere.Jet.AirportID.EQ(100),
    ),
    sm.Limit(2),
).All()
```

