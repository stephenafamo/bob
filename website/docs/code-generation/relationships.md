---
sidebar_position: 5
description: Working with relationships
---

# Relationships

Related models are stored in the `R` field of the generated structs. For example, the related pilot of a jet will be `jet.R.Pilot`.

## Relationship Types

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

- One
- All
- Cursor
- Count
- Exists
- UpdateAll
- DeleteAll

Naturally, we can add mods to the query:

```go
jet, err := models.FindJet(ctx, db, 1)

// SELECT * FROM "pilots" WHERE "id" = $1 LIMIT 20
jetPilotQuery, err := jet.Pilots(ctx, db, sm.Limit(20))
```

## Modifying Relationships

- InsertXXX: This inserts a new row and sets it as the related model

  ```go
  // to-one
  jet.InsertPilot(ctx, db, &PilotSetter{...})

  // to-many
  pilot.InsertJets(ctx, db, &JetSetter{...}, &JetSetter{...})
  ```

- AttachXXX: This attaches an existing model as a relation

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
models.Preload.Jet.Pilot(opts ...psql.PreloadOption)
```

The mod function accepts options:

1. `OnlyColumns`: In the related model, load only these columns.
1. `ExceptColumns`: In the related model, do not load these columns.
1. `PreloadAs`: Explicitly sets the table alias for the related model to allow using columns of the related model for the query.
1. `Loaders`: Other loaders mods can be given as an option to the preloader to load nested relationships. This works for both other preloaders and then-loaders.

```go
jet, err := models.Jets(
    models.Preload.Jet.Pilot(
        psql.OnlyColumns("id"), // only selects "pilot"."id"
        psql.SelectThenLoad.Pilot.Licences(), // will load the pilot's licences
    ),
).One(ctx, db)
```

```go
jets, err := models.Jets(
	models.Preload.Jet.Pilot(psql.PreloadAs("pilot")), // "LEFT JOIN "pilots" AS "pilot" ON ("jet"."pilot_id" = "pilot"."id")
	models.Preload.Jet.CoPilot(psql.PreloadAs("copilot")), // "LEFT JOIN "pilots" AS "copilot" ON ("jet"."copilot_id" = "copilot"."id")
	sm.OrderBy(psql.Quote("pilot", models.ColumnNames.Pilot.LastName)) // ORDER BY "pilot"."last_name" DESC
).All(ctx, db)
```

### ThenLoad

```go
models.SelectThenLoad.Pilots.Jets(...mods)
models.InsertThenLoad.Pilots.Jets(...mods)
models.UpdateThenLoad.Pilots.Jets(...mods) // not supported for mysql
```

These will accept **ANY** `Select/Insert/UpdateQuery` mods.

```go
// get the first 2 pilots
// then load all related jets with airport_id = 100
pilots, err := models.Pilots(
    models.ThenLoad.Pilots.Jets(
        models.SelectWhere.Jet.AirportID.EQ(100),
    ),
    sm.Limit(2),
).All(ctx, db)
```

## Checking if a relationship has been loaded

Each model exposes a `R.Loaded` struct with one `bool` per relationship that records whether that relationship has been populated. This lets you tell apart `nil` ("not loaded yet") from a genuine empty result ("loaded, but no related rows").

```go
jet, err := models.FindJet(ctx, db, 1)

if !jet.R.Loaded.Pilot {
    // the pilot relationship has not been loaded
}

if err := jet.LoadPilot(ctx, db); err != nil {
    return err
}

// jet.R.Loaded.Pilot is now true. The pilot may still be nil if the
// foreign key was null and no row matched.
if jet.R.Loaded.Pilot && jet.R.Pilot == nil {
    // definitively no pilot
}
```

The same applies to `to-many` relationships:

```go
pilot, err := models.FindPilot(ctx, db, 1)
if err := pilot.LoadJets(ctx, db); err != nil {
    return err
}

// pilot.R.Loaded.Jets is true even if the pilot has no jets.
if pilot.R.Loaded.Jets && len(pilot.R.Jets) == 0 {
    // definitively zero jets
}
```

`R.Loaded.X` is set to `true` by:

- `LoadX`, `Preload.X` and `ThenLoad.X` after they populate `R.X` (including the zero-row case).
- The inverse-side assignment performed during loading (e.g. when `pilots.LoadJets(...)` sets each `jet.R.Pilot`, the corresponding `jet.R.Loaded.Pilot` is also set).
- `AttachX` and `InsertX` for `to-one` relationships, where the relation is fully known after the call.
- Factory `Build` and `Create`, since the factory declares the complete world for the test fixture.

`R.Loaded.X` is **not** changed by `AttachX` and `InsertX` for `to-many` relationships, since appending rows does not turn a partial slice into a complete one.

:::note

If you assign to `R` directly (e.g. `jet.R.Pilot = pilot`) you are also responsible for keeping `R.Loaded` in sync. The generated APIs above maintain it for you; manual mutation does not.

:::

:::note

`Loaded` is a reserved relationship alias. Generation will fail if any relationship is aliased as `Loaded`.

The field name is configurable via [`relation_loaded_name`](./configuration#configuration) in the bobgen config (default `Loaded`). The configured value is used both for the field on `R` and as the suffix of the underlying type (`<table>R<name>`), and is reserved as a relationship alias for that generation run.

```yaml
# bobgen.yaml
relation_loaded_name: LoadInfo  # exposes model.R.LoadInfo instead of model.R.Loaded
```

:::

## Counting Relationships

Bob can also count related models without loading them. This is useful when you need to display counts (e.g., "5 comments") without fetching all the related data. Counts are stored in the `C` field of the generated structs as `*int64` pointers.

:::note

Relationship counts are only available for `to-many` relationships.

:::

### PreloadCount

`PreloadCount` adds a correlated subquery to the main SELECT query. This is efficient when you need both the parent model and the count in a single database round-trip.

```go
models.PreloadCount.Pilot.Jets(...mods)
```

The mod function accepts **ANY** `SelectQuery` mods to filter which related models are counted.

```go
// Get all pilots with their jet counts in a single query
// SQL: SELECT *, (SELECT count(*) FROM jets WHERE jets.pilot_id = pilots.id) AS "__count_Jets" FROM pilots
pilots, err := models.Pilots(
    models.PreloadCount.Pilot.Jets(),
).All(ctx, db)

// Access the count
for _, pilot := range pilots {
    if pilot.C.Jets != nil {
        fmt.Printf("Pilot %s has %d jets\n", pilot.Name, *pilot.C.Jets)
    }
}
```

With filtering:

```go
// Count only active jets
// SQL: SELECT *, (SELECT count(*) FROM jets WHERE jets.pilot_id = pilots.id AND jets.active = true) AS "__count_Jets" FROM pilots
pilots, err := models.Pilots(
    models.PreloadCount.Pilot.Jets(
        models.SelectWhere.Jet.Active.EQ(true),
    ),
).All(ctx, db)
```

### ThenLoadCount

`ThenLoadCount` loads the count in a separate query after the main query completes. This is similar to `ThenLoad` but only retrieves the count.

```go
models.ThenLoadCount.Pilot.Jets(...mods)
```

```go
// Get pilots, then load jet counts in a separate query
pilots, err := models.Pilots(
    models.ThenLoadCount.Pilot.Jets(),
    sm.Limit(10),
).All(ctx, db)

// Access the count
for _, pilot := range pilots {
    if pilot.C.Jets != nil {
        fmt.Printf("Pilot %s has %d jets\n", pilot.Name, *pilot.C.Jets)
    }
}
```

### InsertThenLoadCount

`InsertThenLoadCount` loads the count in a separate query after the main query completes. This is similar to `ThenLoad` but only retrieves the count.

```go
pilot,err := models.Pilots.Insert(
    &models.PilotSetter{
        Name: omit.From("John Doe"),
    },
    models.InsertThenLoadCount.Pilot.Jets(...mods),
).One(ctx, db)

// Access the count
if pilot.C.Jets != nil {
    fmt.Printf("Pilot has %d jets\n", *pilot.C.Jets)
}
```

### Direct Count Loading

You can also load counts directly on existing model instances:

```go
pilot, err := models.FindPilot(ctx, db, pilotID)

// Load the count of jets for this pilot
err = pilot.LoadCountJets(ctx, db)

// With filtering
err = pilot.LoadCountJets(ctx, db,
    models.SelectWhere.Jet.Active.EQ(true),
)

// Access the count
if pilot.C.Jets != nil {
    fmt.Printf("Pilot has %d jets\n", *pilot.C.Jets)
}
```

This also works on slices:

```go
pilots, err := models.Pilots().All(ctx, db)

// Load jet counts for all pilots
err = pilots.LoadCountJets(ctx, db)
```
