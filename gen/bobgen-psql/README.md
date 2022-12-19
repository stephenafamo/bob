# Bob Gen for Postgres

Generates an ORM based on a postgres database schema

## Usage

```sh
PSQL_DSN=postgres://user:pass@host:port/dbname go run github.com/stephenafamo/bob/gen/bobgen-psql@latest
```

### Driver Configuration

#### [Link to general configuration][gen/README.md]

The configuration for the postgres driver must all be prefixed by the driver name.  
You must use a configuration file or environment variables for configuring the database driver;  
there are no command-line options for providing driver-specific configuration.

In the configuration file for postgresql for example you would do:

```yaml
psql:
  dsn: "postgres://user:pass@host:port/dbname"
```

When you use an environment variable it must also be prefixed by the driver
name:

```sh
PSQL_DSN="postgres://user:pass@host:port/dbname"
```

The values that exist for the drivers:

| Name | Required | Postgres Default |
| ---- | -------- | ---------------- |
| schema    | no        | "public"  |
| dsn       | yes       | none      |
| whitelist | no        | []        |
| blacklist | no        | []        |

Example of whitelist/blacklist:

```yaml
psql:
    # Removes migrations table, the name column from the addresses table, and
    # secret_col of any table from being generated. Foreign keys that reference tables
    # or columns that are no longer generated because of whitelists or blacklists may
    # cause problems.
    blacklist: ["migrations", "addresses.name", "*.secret_col"]
```
