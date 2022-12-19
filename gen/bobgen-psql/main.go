package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/bobgen-psql/driver"
)

var version = func() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return ""
}()

//nolint:gochecknoglobals
var (
	flagConfigFile string
	cmdState       *gen.State[driver.Extra]
	cmdConfig      *gen.Config[driver.Extra]
)

func initConfig() {
	if len(flagConfigFile) != 0 {
		viper.SetConfigFile(flagConfigFile)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println("Can't read config:", err)
			os.Exit(1)
		}
		return
	}

	var err error
	viper.SetConfigName("bobgen")

	configHome := os.Getenv("XDG_CONFIG_HOME")
	homePath := os.Getenv("HOME")
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	configPaths := []string{wd}
	if len(configHome) > 0 {
		configPaths = append(configPaths, filepath.Join(configHome, "bobgen"))
	} else {
		configPaths = append(configPaths, filepath.Join(homePath, ".config/bobgen"))
	}

	for _, p := range configPaths {
		viper.AddConfigPath(p)
	}

	// Ignore errors here, fallback to other validation methods.
	// Users can use environment variables if a config is not found.
	_ = viper.ReadInConfig()
}

func main() {
	// Too much happens between here and cobra's argument handling, for
	// something so simple just do it immediately.
	for _, arg := range os.Args {
		if arg == "--version" {
			fmt.Println("BobGen Psql v" + version)
			return
		}
	}

	// Set up the cobra root command
	rootCmd := &cobra.Command{
		Use:   "bobgen-psql [flags]",
		Short: "BobGen Psql generates models for your postgres database.",
		Long: "BobGen Psql generates models for your postgres database.\n" +
			`Complete documentation is available at http://github.com/stephenafamo/bob/gen/psql`,
		Example:       `bobgen-psql`,
		PreRunE:       preRun,
		RunE:          run,
		PostRunE:      postRun,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cobra.OnInitialize(initConfig)

	// Set up the cobra root command flags
	rootCmd.PersistentFlags().StringVarP(&flagConfigFile, "config", "c", "", "Filename of config file to override default lookup")
	rootCmd.PersistentFlags().StringP("output", "o", "models", "The name of the folder to output to")
	rootCmd.PersistentFlags().StringP("pkgname", "p", "models", "The name you wish to assign to your generated package")
	rootCmd.PersistentFlags().BoolP("no-tests", "", false, "Disable generated go test files")
	rootCmd.PersistentFlags().BoolP("version", "", false, "Print the version")
	rootCmd.PersistentFlags().BoolP("wipe", "", false, "Delete the output folder (rm -rf) before generation to ensure sanity")
	rootCmd.PersistentFlags().StringP("struct-tag-casing", "", "snake", "Decides the casing for go structure tag names. camel, title or snake (default snake)")
	rootCmd.PersistentFlags().StringP("relation-tag", "r", "-", "Relationship struct tag name")
	rootCmd.PersistentFlags().StringSliceP("tag-ignore", "", nil, "List of column names that should have tags values set to '-' (ignored during parsing)")
	rootCmd.PersistentFlags().IntP("concurrency", "", 10, "How many tables to fetch in parallel")
	rootCmd.PersistentFlags().BoolP("no-back-referencing", "", false, "Disable back referencing in the loaded relationship structs")

	// Driver config flags
	rootCmd.PersistentFlags().StringP("psql.dsn", "d", "", "The database connection string")
	rootCmd.PersistentFlags().StringP("psql.schema", "s", "public", "The database schema to use")
	rootCmd.PersistentFlags().StringSliceP("psql.whitelist", "", nil, "List of tables that will be included. Others are ignored")
	rootCmd.PersistentFlags().StringSliceP("psql.blacklist", "", nil, "List of tables that will be should be ignored")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// hide flags not recommended for use
	if err := rootCmd.PersistentFlags().MarkHidden("no-tests"); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		var cmdErr commandError
		if errors.As(err, &cmdErr) {
			fmt.Printf("Error: %v\n\n", string(cmdErr))
			rootCmd.Help() //nolint:errcheck
		} else if !viper.GetBool("debug") {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Error: %+v\n", err)
		}

		os.Exit(1)
	}
}

type commandError string

func (c commandError) Error() string {
	return string(c)
}

func preRun(cmd *cobra.Command, args []string) error {
	var err error

	dsn := viper.GetString("psql.dsn")
	if dsn == "" {
		return errors.New("database dsn is not set")
	}
	modelsPkg, err := gen.ModelsPackage(viper.GetString("output"))
	if err != nil {
		return fmt.Errorf("getting models package: %w", err)
	}

	cmdConfig = &gen.Config[driver.Extra]{
		Driver: driver.New(driver.Config{
			Dsn:         dsn,
			Schema:      viper.GetString("psql.schema"),
			Includes:    viper.GetStringSlice("psql.whitelist"),
			Excludes:    viper.GetStringSlice("psql.blacklist"),
			Concurrency: viper.GetInt("concurrency"),
		}),
		Outputs: []*gen.Output{
			{
				OutFolder: viper.GetString("output"),
				PkgName:   viper.GetString("pkgname"),
				Templates: []fs.FS{gen.ModelTemplates, driver.ModelTemplates},
			},
		},
		Wipe:              viper.GetBool("wipe"),
		StructTagCasing:   strings.ToLower(viper.GetString("struct-tag-casing")), // camel | snake | title
		TagIgnore:         viper.GetStringSlice("tag-ignore"),
		RelationTag:       viper.GetString("relation-tag"),
		NoBackReferencing: viper.GetBool("no-back-referencing"),

		Aliases:       gen.ConvertAliases(viper.Get("aliases")),
		Replacements:  gen.ConvertReplacements(viper.Get("replacements")),
		Relationships: gen.ConvertRelationships(viper.Get("relationships")),
		Inflections: gen.Inflections{
			Plural:        viper.GetStringMapString("inflections.plural"),
			PluralExact:   viper.GetStringMapString("inflections.plural_exact"),
			Singular:      viper.GetStringMapString("inflections.singular"),
			SingularExact: viper.GetStringMapString("inflections.singular_exact"),
			Irregular:     viper.GetStringMapString("inflections.irregular"),
		},

		Generator:     "BobGen Psql " + version,
		ModelsPackage: modelsPkg,
	}

	cmdState, err = gen.New("psql", cmdConfig)
	return err
}

func run(cmd *cobra.Command, args []string) error {
	return cmdState.Run()
}

func postRun(cmd *cobra.Command, args []string) error {
	return cmdState.Cleanup()
}
