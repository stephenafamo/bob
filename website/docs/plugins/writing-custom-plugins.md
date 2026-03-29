---
sidebar_position: 2
description: How to write custom plugins for Bob's code generation
---

# Writing Custom Plugins

To use custom plugins, you need to create your own generator binary instead of running the CLI tools (e.g. `bobgen-psql`) directly. This is straightforward - the CLI tools themselves are thin wrappers around the generation engine.

## Getting Started

### Step 1: Create Your Own Generator

Copy the `main.go` from the CLI tool for your database. For example, here is [`bobgen-psql/main.go`](https://github.com/stephenafamo/bob/blob/main/gen/bobgen-psql/main.go) in its entirety:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/stephenafamo/bob/gen"
    helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
    "github.com/stephenafamo/bob/gen/bobgen-psql/driver"
    "github.com/stephenafamo/bob/gen/plugins"
    "github.com/urfave/cli/v2"
)

func main() {
    ctx, cancel := signal.NotifyContext(
        context.Background(),
        syscall.SIGINT,
        syscall.SIGTERM,
    )
    defer cancel()

    app := &cli.App{
        Name:      "bobgen-psql",
        Usage:     "Generate models and factories from your PostgreSQL database",
        UsageText: "bobgen-psql [-c FILE]",
        Version:   helpers.Version(),
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "config",
                Aliases: []string{"c"},
                Value:   helpers.DefaultConfigPath,
                Usage:   "Load configuration from `FILE`",
            },
        },
        Action: run,
    }

    if err := app.RunContext(ctx, os.Args); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func run(c *cli.Context) error {
    config, driverConfig, pluginsConfig, err := helpers.GetConfigFromFile[any, driver.Config](c.String("config"), "psql")
    if err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    outputPlugins := plugins.Setup[any, any, driver.IndexExtra](
        pluginsConfig, gen.PSQLTemplates,
    )

    state := &gen.State[any]{Config: config}
    return gen.Run(c.Context, state, driver.New(driverConfig), outputPlugins...)
}
```

The key part is the `run` function - it loads configuration, sets up the built-in plugins, and calls `gen.Run`. This is where you'll add your custom plugin.

### Step 2: Write Your Plugin

Create a custom plugin that implements one or more of the interfaces in [`gen/plugin.go`](https://github.com/stephenafamo/bob/blob/main/gen/plugin.go). Every plugin must implement the base `Plugin` interface:

```go
type Plugin interface {
    Name() string
}
```

Then implement one or more of these depending on where in the generation lifecycle you need to hook in:

| Interface            | Method                              | Phase                                                     |
| -------------------- | ----------------------------------- | --------------------------------------------------------- |
| `StatePlugin`        | `PlugState(*State)`                 | Before database info is fetched. Register outputs here.   |
| `DBInfoPlugin`       | `PlugDBInfo(*DBInfo)`               | After the driver assembles the database schema.           |
| `TemplateDataPlugin` | `PlugTemplateData(*TemplateData)`   | After template data is assembled, before generation.      |

- **`StatePlugin`** gives you access to `State`, where you can register new outputs, append templates to existing outputs, or modify generation configuration like aliases and template functions.
- **`DBInfoPlugin`** gives you access to `DBInfo`, which contains the full database schema - tables, columns, enums, and query folders. Use this to filter, transform, or enrich schema information.
- **`TemplateDataPlugin`** gives you access to `TemplateData`, which contains the fully processed data (tables, relationships, aliases) that will be passed to templates. Use this to validate or make final adjustments before code generation.

### Step 3: Load Your Plugin

Add your custom plugin to the `run` function alongside the built-in ones:

```go
func run(c *cli.Context) error {
    config, driverConfig, pluginsConfig, err := helpers.GetConfigFromFile[any, driver.Config](c.String("config"), "psql")
    if err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    builtinPlugins := plugins.Setup[any, any, driver.IndexExtra](
        pluginsConfig, gen.PSQLTemplates,
    )

    // Add your custom plugin after the built-in ones
    allPlugins := append(builtinPlugins, &myPlugin[any]{})

    state := &gen.State[any]{Config: config}
    return gen.Run(c.Context, state, driver.New(driverConfig), allPlugins...)
}
```

Then run your generator instead of `bobgen-psql`:

```bash
go run ./cmd/my-generator -c bob.yaml
```

:::tip

Plugins are executed in the order they are passed. If your custom plugin depends on outputs registered by built-in plugins (e.g. the `models` output), make sure it comes after them in the list.

:::

## Examples

### Adding a New Output

If you want to generate an entirely separate package (e.g. REST handlers, GraphQL schemas, or validation code), register a new output in a `StatePlugin`:

```go
//go:embed templates
var myTemplates embed.FS

type myPlugin[C any] struct{}

func (myPlugin[C]) Name() string {
    return "my-plugin"
}

func (myPlugin[C]) PlugState(state *gen.State[C]) error {
    templates, err := fs.Sub(myTemplates, "templates")
    if err != nil {
        return fmt.Errorf("failed to load templates: %w", err)
    }

    state.Outputs = append(state.Outputs, &gen.Output{
        Key:       "my-output",   // unique identifier, used by other plugins to find this output
        OutFolder: "myoutput",    // directory where generated files are written
        PkgName:   "myoutput",    // Go package name for generated files
        Templates: []fs.FS{templates}, // Go templates to render
    })

    return nil
}
```

### Extending an Existing Output

If you want to add functionality to an already generated package (e.g. adding custom helpers to the `models` package), find the output by its key and append your templates:

```go
func (m myPlugin[C]) PlugState(state *gen.State[C]) error {
    for _, output := range state.Outputs {
        if output.Key == "models" {
            output.Templates = append(output.Templates, myTemplates)
            break
        }
    }

    return nil
}
```

:::tip

All of Bob's built-in plugins use these same interfaces. Browse the [`gen/plugins/`](https://github.com/stephenafamo/bob/tree/main/gen/plugins) package for real-world examples of each pattern.

:::
