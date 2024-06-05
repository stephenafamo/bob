---
sidebar_position: 6
description: Using model factories to speed up testing
---

# Factories

A sub-package containing factories for your models are also generated.

:::note

To disable factory generation, set `no_factory: true` in your [configuration](./configuration)

:::

## What are bob factories?

A factory is an object that can create templates for new models.

```go
f := factory.New()

// Create a new template from the factory
jetTemplate := f.NewJet(...factory.JetMod)
```

While we can always add mods when getting a new template, it is often useful to set mods on the factory that will always be applied to any new template:

```go
f := factory.New()

f.AddBaseJetMods(
    factory.JetMods.RandomID(),
    factory.JetMods.RandomAirportID(),
)

// The jet templates will generate models with random IDs and AirportIDs
jetTemplate1 := f.NewJet()
jetTemplate2 := f.NewJet()

// We can also clear the base mods
f.ClearBaseJetMods()
```

## Mods

Factory mods affect how the template will generate models.

```go
type JetMod interface {
	Apply(*JetTemplate)
}
```

:::tip

For very custom use cases, you can write your own `JetMod`.  
As long as it implements the interface, it can be used in the same way as the generated mods.

:::

### Column Mods

Mods are generated to change how columns are set from the template.

```go
f.NewJet(
   // generate models with ID set to 10
   factory.JetMods.ID(10),

   // generate models with ID set by a function that returns a random integer
   factory.JetMods.IDFunc(func() int { return rand.Int() }),

   // Clear any previously set ID
   factory.JetMods.UnsetID(),

   // Generate a random value for the column
   // Uses a faker from https://github.com/jaswdr/faker
   // pass nil to use the default faker
   factory.JetMods.RandomID(nil),

   // Set random values on all columns
   factory.JetMods.RandomizeAllColumns(nil),
)
```

### Relationship Mods

Mods are generated to modify the relationship of models from the template.

```go
// FOR to-one relationships
f.NewJet(
    // Clear the related pilot
    factory.JetMods.WithoutPilot(),

    // add a pilot from the given template
    factory.JetMods.WithPilot(&PilotTemplate),

    // generate a new pilot template from the same factory and use it
    factory.JetMods.WithNewPilot(...factory.PilotMod),
)

// FOR to-many relationships
f.NewPilot(
    // Clear all related jets
    factory.PilotMods.WithoutJets(),

    //---------------------------------------------------
    // SET variants overwrite any existing relationships
    //---------------------------------------------------

    // set exactly 5 jets generated from the given template
    factory.PilotMods.WithJets(5, &JetTemplate{}),

    // set exactly 5 jets generated from the same factory with the given mods
    factory.PilotMods.WithNewJets(5, ...factory.JetMod),

    //---------------------------------------------------
    // ADD variants append to existing relationships
    //---------------------------------------------------

    // add 5 jets generated from the given template
    factory.PilotMods.AddJets(5, &JetTemplate{}),

    // add 5 jets generated from the same factory with the given mods
    factory.PilotMods.AddNewJets(5, ...factory.JetMod),
)
```

## Using a Template

From a template we can:

### Build

:::note

Build variants DO NOT interact with the database

:::

```go
// Build a new jet setter from the template
// naturally, this ignores relationships
jetSetter := jetTemplate.BuildSetter()

// Build a slice of 5 jet setters using the template
// also ignores relationships
jetSetters := jetTemplate.BuildManySetter(5)

// Build a new jet from the template
// related templates are built and put in the `R` struct
jet := jetTemplate.Build()

// Build a slice of 5 jets using the template
jets := jetTemplate.BuildMany(5)
```

### Create

:::note

Create variants insert the models into the database  
Any required relation (i.e. a non-nullable foreign key), will also be created even if the template did not define one.

:::

```go
// Create a new jet from the template
jet, err := jetTemplate.Create(ctx, db)

// Create a slice of 5 jets using the template
jets, err := jetTemplate.CreateMany(ctx, db, 5)

// Must variants panic on error
jet := jetTemplate.MustCreate(ctx, db)
jets := jetTemplate.MustCreateMany(ctx, db, 5)

// OrFail variants will fail the test or benchmark if an error occurs
jet := jetTemplate.CreateOrFail(t, db)
jets := jetTemplate.CreateManyOrFail(t, db, 5)
```
