package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"

	"github.com/spf13/viper"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-prisma/driver"
)

//nolint:gochecknoglobals
var (
	writeDebugFile = os.Getenv("PRISMA_CLIENT_GO_WRITE_DMMF_FILE") != ""
	prismaCLIPath  = os.Getenv("PRISMA_CLI_PATH")
)

func main() {
	if os.Getenv("PRISMA_GENERATOR_INVOCATION") == "" {
		// prisma CLI
		if err := callPrisma(); err != nil {
			panic(err)
		}

		return
	}

	// exit when signal triggers
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		os.Exit(1)
	}()

	if err := servePrisma(); err != nil {
		log.Fatalf("error occurred when invoking prisma: %s", err)
	}
}

func reply(w io.Writer, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not marshal data %w", err)
	}

	b = append(b, byte('\n'))

	if _, err = w.Write(b); err != nil {
		return fmt.Errorf("could not write data %w", err)
	}

	return nil
}

func servePrisma() error {
	reader := bufio.NewReader(os.Stdin)

	if writeDebugFile {
		dir, _ := os.Getwd()
		log.Printf("current working dir: %s", dir)
	}

	for {
		content, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("could not read bytes from stdin: %w", err)
		}

		if writeDebugFile {
			buf := &bytes.Buffer{}
			if err := json.Indent(buf, content, "", "  "); err != nil {
				log.Print(err)
			}
			if err := os.WriteFile("dmmf.json", buf.Bytes(), 0o600); err != nil {
				log.Print(err)
			}
		}

		var input Request

		if err := json.Unmarshal(content, &input); err != nil {
			return fmt.Errorf("could not open stdin %w", err)
		}

		var response interface{}

		switch input.Method {
		case "getManifest":
			response = ManifestResponse{
				Manifest: Manifest{
					DefaultOutput: path.Join(".", "db"),
					PrettyName:    "Bob Prisma Go Client",
				},
			}

		case "generate":
			response = nil // success

			var params root

			if err := json.Unmarshal(input.Params, &params); err != nil {
				dir, _ := os.Getwd()
				return fmt.Errorf("could not unmarshal params into generator.Root type at %s: %w", dir, err)
			}

			if err := generate(params); err != nil {
				return fmt.Errorf("could not generate code. %w", err)
			}
		default:
			return fmt.Errorf("no such method %s", input.Method)
		}

		if err := reply(os.Stderr, NewResponse(input.ID, response)); err != nil {
			return fmt.Errorf("could not reply %w", err)
		}
	}
}

func generate(root root) error {
	var err error
	var dialect, driverName, driverPkg string

	datasource := root.Datasources[0]
	switch datasource.Provider {
	case ProviderPostgreSQL:
		dialect = "psql"
		driverName = "pgx"
		driverPkg = "github.com/jackc/pgx/v5/stdlib"
	case ProviderSQLite:
		dialect = "psql"
		driverName = "sqlite"
		driverPkg = "modernc.org/sqlite"
	default:
		return fmt.Errorf("Unsupported datasource provider %q", datasource.Provider)
	}

	output := root.Generator.Output.Value
	if output == "" {
		output = viper.GetString("output")
	}

	helpers.ReadConfig(root.Generator.Config.ConfigFile)
	viper.SetDefault("pkgname", "db")
	viper.SetDefault("factory-pkgname", "factory")
	viper.SetDefault("struct-tag-casing", "snake")
	viper.SetDefault("relation-tag", "-")
	viper.SetDefault("concurrency", 10)
	viper.SetDefault("prisma.schema", "public")

	outputs := []*gen.Output{
		{
			OutFolder: output,
			PkgName:   viper.GetString("pkgname"),
			Templates: []fs.FS{gen.ModelTemplates, driver.ModelTemplates},
		},
	}

	if viper.GetBool("with-factory") {
		outputs = append(outputs, &gen.Output{
			OutFolder: path.Join(output, "factory"),
			PkgName:   viper.GetString("factory-pkgname"),
			Templates: []fs.FS{gen.FactoryTemplates, driver.FactoryTemplates},
		})
	}

	cmdConfig, err := helpers.NewConfig("Prisma", driver.New(driver.Config{
		Datamodel:   root.DMMF.Datamodel,
		Schema:      viper.GetString("prisma.schema"),
		Includes:    viper.GetStringSlice("prisma.whitelist"),
		Excludes:    viper.GetStringSlice("prisma.blacklist"),
		Concurrency: viper.GetInt("concurrency"),
		Provider: driver.Provider{
			DriverName:      driverName,
			DriverPkg:       driverPkg,
			DriverSource:    datasource.URL.Value,
			DriverENVSource: datasource.URL.FromEnvVar,
		},
	}), outputs)
	if err != nil {
		return err
	}

	cmdState, err := gen.New(dialect, cmdConfig)
	if err != nil {
		return fmt.Errorf("could not create new state: %w", err)
	}

	return cmdState.Run()
}

// Root describes the generator output root.
// overwritten so I can set the config
type root struct {
	Generator   Generator       `json:"generator"`
	Datasources []Datasource    `json:"datasources"`
	DMMF        driver.Document `json:"DMMF"`
	SchemaPath  string          `json:"schemaPath"`
}

type config struct {
	ConfigFile string `json:"configFile"`
}

// callPrisma the prisma CLI with given arguments
func callPrisma() error {
	if prismaCLIPath == "" {
		prismaCLIPath = "prisma"
	}

	cmd := exec.Command(prismaCLIPath, "generate")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "PRISMA_HIDE_UPDATE_MESSAGE=true")

	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not run %q: %w", "generate", err)
	}

	return nil
}

// Request sets a generic JSONRPC request, which wraps information and params.
type Request struct {
	// JSONRPC describes the version of the JSON RPC protocol. Defaults to `2.0`.
	JSONRPC string `json:"jsonrpc"`
	// ID identifies a unique request.
	ID int `json:"id"`
	// Method describes the intention of the request.
	Method string `json:"method"`
	// Params contains the payload of the request. Usually parsed into a specific struct for further processing.
	Params json.RawMessage `json:"params"`
}

// Response sets a generic JSONRPC response, which wraps information and a result.
type Response struct {
	// JSONRPC describes the version of the JSON RPC protocol. Defaults to `2.0`.
	JSONRPC string `json:"jsonrpc"`
	// ID identifies a unique request.
	ID int `json:"id"`
	// Result contains the payload of the response.
	Result interface{} `json:"result"`
}

// NewResponse forms a new JSON RPC response to reply to the Prisma CLI commands
func NewResponse(id int, result interface{}) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// Manifest describes information for the Prisma Client Go generator for the Prisma CLI.
type Manifest struct {
	PrettyName         string   `json:"prettyName"`
	DefaultOutput      string   `json:"defaultOutput"`
	Denylist           []string `json:"denylist"`
	RequiresGenerators []string `json:"requiresGenerators"`
	RequiresEngines    []string `json:"requiresEngines"`
}

// ManifestResponse sets the response Prisma Client Go returns when Prisma asks for the Manifest.
type ManifestResponse struct {
	Manifest Manifest `json:"manifest"`
}

type Generator struct {
	Name     string `json:"name"`
	Output   Value  `json:"output"`
	Provider Value  `json:"provider"`
	Config   config `json:"config"`
}

// Datasource describes a Prisma data source of any database type.
type Datasource struct {
	Name     string   `json:"name"`
	Provider Provider `json:"provider"`
	URL      Value    `json:"url"`
}

type Value struct {
	FromEnvVar string `json:"fromEnvVar"`
	Value      string `json:"value"`
}

// Provider describes the Database of this generator.
type Provider string

// Provider values
const (
	ProviderMySQL       Provider = "mysql"
	ProviderMongo       Provider = "mongo"
	ProviderSQLite      Provider = "sqlite"
	ProviderSQLServer   Provider = "sqlserver"
	ProviderPostgreSQL  Provider = "postgresql"
	ProviderCockroachDB Provider = "cockroachdb"
)
