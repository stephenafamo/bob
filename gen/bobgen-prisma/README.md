# Bob Gen for Prisma

Generates an ORM based on a prisma schema file

## How to Generate

1. Initialise a new Go project.
    The generator must be run from inside a go module, so either the current directory or a parent directory should contain a `go.mod` file.
    If you don't have a Go project yet, initialise one using Go modules:

    ```shell script
    mkdir demo && cd demo
    go mod init demo
    ```

1. Get the Bob Prisma Generator. Install the Go module in your project by running:

    ```shell script
    go get github.com/stephenafamo/bob/gen/bobgen-prisma
    ```

1. Prepare your database schema in a `schema.prisma` file. For example, a simple schema with a postgres database and the Bob Prisma generator with two models would look like this:

    ```prisma
    datasource db {
        // only postgresql is supported for now
        provider = "postgresql"
        url      = env("DATABASE_URL")
    }

    generator db {
        provider   = "go run github.com/stephenafamo/bob/gen/bobgen-prisma"
        output     = "./prisma" // Optional. Default: ./db
        configFile = "./config/bobgen.yml" // Optional. Default ./bobgen.yaml
    }

    model Post {
        id          String           @id @default(dbgenerated("gen_random_uuid()")) @db.Uuid
        created_at  DateTime         @default(now()) @db.Timestamptz(6)
        updated_at  DateTime         @db.Timestamptz(6)
        title       String
        published   Boolean
        desc        String?
    }
    ```

1. Generate the models using `prisma generate`

## How to Use

[Detailed documentation on how to use the Bob ORM can be found here](..)

A small taste

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    db "random/prisma/bob"

    "github.com/aarondl/opt/omit"
    "github.com/aarondl/opt/omitnull"
)

func main() {
    if err := run(context.Background()); err != nil {
        panic(err)
    }
}

func run(ctx context.Context) error {
    client, err := db.New()
    if err != nil {
        return err
    }
    defer client.Close()

    // create a post
    createdPost, err := db.PostsTable.Insert(ctx, client, &db.OptionalPost{
        Title:     omit.From("Hi from Prisma!"),
        Published: omit.From(true),
        Desc:      omitnull.From("Prisma is a database toolkit and makes databases easy."),
    })
    if err != nil {
        return err
    }

    result, _ := json.MarshalIndent(createdPost, "", "  ")
    fmt.Printf("created post: %s\n", result)

    // find a single post
    post, err := db.FindPost(ctx, client, createdPost.ID)
    if err != nil {
        return err
    }

    result, _ = json.MarshalIndent(post, "", "  ")
    fmt.Printf("post: %s\n", result)

    // For optional/nullable values, the field will have a generic null wrapper
    // with some helpful methods
    if post.Desc.IsNull() {
        return fmt.Errorf("post's description is null")
    }

    fmt.Printf("The posts's description is: %s\n", post.Desc.MustGet())

    return nil
}
```

[Detailed documentation on how to use the Bob ORM can be found here](..)

## Driver Configuration

The configuration for the prisma driver must all be prefixed by the driver name.  
You must use a configuration file or environment variables for configuring the database driver.

In the configuration file for prisma for example you would do:

```yaml
prisma:
  schema: "public"
```

When you use an environment variable it must also be prefixed by the driver
name:

```sh
PRISMA_SCHEMA="public"
```

The values that exist for the drivers:

| Name      | Required  | Prisma Default |
| --------- | --------- | -------------- |
| schema    | no        | ""             |
| whitelist | no        | []             |
| blacklist | no        | []             |

Example of whitelist/blacklist:

```yaml
prisma:
    # Removes migrations table, the name column from the addresses table, and
    # secret_col of any table from being generated. Foreign keys that reference tables
    # or columns that are no longer generated because of whitelists or blacklists may
    # cause problems.
    blacklist: ["migrations", "addresses.name", "*.secret_col"]
```

## Known Issues

### `@map` attributes on columns are ignored

This is because `@map` information is currently not given to generators by the `prisma` command. [See Issues #3998](https://github.com/prisma/prisma/issues/3998)
