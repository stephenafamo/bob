package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path"
	"runtime/debug"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
)

const DefaultConfigPath = "./bobgen.yaml"

func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return ""
}

type Templates struct {
	Models  []fs.FS
	Factory []fs.FS
}

func DefaultOutputs(destination, pkgname string, noFactory bool, templates *Templates) []*gen.Output {
	if templates == nil {
		templates = &Templates{}
	}

	if destination == "" {
		destination = "models"
	}

	if pkgname == "" {
		pkgname = "models"
	}

	outputs := []*gen.Output{
		{
			Key:       "models",
			OutFolder: destination,
			PkgName:   pkgname,
			Templates: append(templates.Models, gen.ModelTemplates),
		},
	}

	if !noFactory {
		outputs = append(outputs, &gen.Output{
			Key:       "factory",
			OutFolder: path.Join(destination, "factory"),
			PkgName:   "factory",
			Templates: append(templates.Factory, gen.FactoryTemplates),
		})
	}

	return outputs
}

func GetConfigFromFile[DriverConfig any](configPath, driverConfigKey string) (gen.Config, DriverConfig, error) {
	var provider koanf.Provider
	var config gen.Config
	var driverConfig DriverConfig

	_, err := os.Stat(configPath)
	if err == nil {
		// set the provider if provided
		provider = file.Provider(configPath)
	}
	if err != nil && !(configPath == DefaultConfigPath && errors.Is(err, os.ErrNotExist)) {
		return config, driverConfig, err
	}

	return GetConfigFromProvider[DriverConfig](provider, driverConfigKey)
}

func GetConfigFromProvider[DriverConfig any](provider koanf.Provider, driverConfigKey string) (gen.Config, DriverConfig, error) {
	var config gen.Config
	var driverConfig DriverConfig

	k := koanf.New(".")

	// Add some defaults
	err := k.Load(confmap.Provider(map[string]any{
		"wipe":              true,
		"struct_tag_casing": "snake",
		"relation_tag":      "-",
		"generator":         fmt.Sprintf("BobGen %s %s", driverConfigKey, Version()),
	}, "."), nil)
	if err != nil {
		return config, driverConfig, err
	}

	if provider != nil {
		// Load YAML config and merge into the previously loaded config (because we can).
		err := k.Load(provider, yaml.Parser())
		if err != nil {
			return config, driverConfig, err
		}
	}

	// Load env variables for ONLY driver config
	envKey := strings.ToUpper(driverConfigKey) + "_"
	err = k.Load(env.Provider(envKey, ".", func(s string) string {
		// replace only the first underscore to make it a flat map[string]any
		return strings.Replace(strings.ToLower(s), "_", ".", 1)
	}), nil)
	if err != nil {
		return config, driverConfig, err
	}

	err = k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, err
	}

	err = k.UnmarshalWithConf(driverConfigKey, &driverConfig, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, err
	}

	return config, driverConfig, nil
}

const parrayImport = `"github.com/stephenafamo/bob/types/parray"`

func AddPgEnumType(types drivers.Types, enum string) string {
	types[enum] = drivers.Type{
		NoRandomizationTest: true, // enums are often not random enough
		RandomExpr: fmt.Sprintf(`all := all%s()
            return any(all[f.IntBetween(0, len(all)-1)]).(T)`, enum),
	}

	return enum
}

func AddPgEnumArrayType(types drivers.Types, enum string) string {
	typ := fmt.Sprintf("parray.EnumArray[%s]", enum)

	types[typ] = drivers.Type{
		Imports:             importers.List{parrayImport},
		NoRandomizationTest: true, // enums are often not random enough
		RandomExpr: fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random[%s](f)
            }
            return any(arr).(T)`, typ, enum),
	}

	return typ
}

func AddPgGenericArrayType(types drivers.Types, singleTyp string) string {
	singleTypDef := types[singleTyp]
	singleComparer := strings.ReplaceAll(singleTypDef.CompareExpr, "AAA", "a")
	singleComparer = strings.ReplaceAll(singleComparer, "BBB", "b")
	if singleComparer == "" {
		singleComparer = "a == b"
	}

	typ := fmt.Sprintf("parray.Array[%s]", singleTyp)

	types[typ] = drivers.Type{
		Imports: append(importers.List{parrayImport}, singleTypDef.Imports...),
		RandomExpr: fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random[%s](f)
            }
            return any(arr).(T)`, typ, singleTyp),
		CompareExpr: fmt.Sprintf(`slices.EqualFunc(AAA, BBB, func(a, b %s) bool {
                return %s
            })`, singleTyp, singleComparer),
		CompareExprImports: append(
			importers.List{`"slices"`},
			singleTypDef.CompareExprImports...),
	}

	return typ
}

func Types() drivers.Types {
	return drivers.Types{
		"[]byte": {
			CompareExpr:        `bytes.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"bytes"`},
		},
		"time.Time": {
			Imports: importers.List{`"time"`},
			RandomExpr: `year := time.Hour * 24 * 365
                min := time.Now().Add(-year)
                max := time.Now().Add(year)
                return any(f.Time().TimeBetween(min, max)).(T)`,
			CompareExpr: `AAA.Equal(BBB)`,
		},
		"netip.Addr": {
			Imports: importers.List{`"net/netip"`},
			RandomExpr: `var addr [4]byte
                rand.Read(addr[:])
                return any(netip.AddrFrom4(addr)).(T)`,
			RandomExprImports: importers.List{`"crypto/rand"`},
		},
		"net.HardwareAddr": {
			Imports: importers.List{`"net"`},
			RandomExpr: `addr, _ := net.ParseMAC(f.Internet().MacAddress())
                return any(addr).(T)`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pq.BoolArray": {
			Imports: importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.BoolArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = f.Bool()
                }
                return any(arr).(T)`,
			NoRandomizationTest: true,
		},
		"pq.Int64Array": {
			Imports: importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.Int64Array, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = f.Int64()
                }
                return any(arr).(T)`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pq.ByteaArray": {
			Imports: importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.ByteaArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random[[]byte](f)
                }
                return any(arr).(T)`,
			CompareExpr: `slices.EqualFunc(AAA, BBB, func(a, b []byte) bool {
                return bytes.Equal(a, b)
            })`,
			CompareExprImports: importers.List{`"slices"`, `"bytes"`},
		},
		"pq.StringArray": {
			Imports: importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.StringArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random[string](f)
                }
                return any(arr).(T)`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pq.Float64Array": {
			Imports: importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.Float64Array, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = f.Float64()
                }
                return any(arr).(T)`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pgeo.Box": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return any(pgeo.NewRandBox()).(T)`,
		},
		"pgeo.Circle": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return any(pgeo.NewRandCircle()).(T)`,
		},
		"pgeo.Line": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return any(pgeo.NewRandLine()).(T)`,
		},
		"pgeo.Lseg": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return any(pgeo.NewRandLseg()).(T)`,
		},
		"pgeo.Path": {
			Imports:     importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr:  `return any(pgeo.NewRandPath()).(T)`,
			CompareExpr: `AAA.Closed == BBB.Closed && slices.Equal(AAA.Points, BBB.Points)`,
		},
		"pgeo.Point": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return any(pgeo.NewRandPoint()).(T)`,
		},
		"pgeo.Polygon": {
			Imports:            importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr:         `return any(pgeo.NewRandPolygon()).(T)`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"decimal.Decimal": {
			Imports:    importers.List{`"github.com/shopspring/decimal"`},
			RandomExpr: `return any(decimal.New(f.Int64Between(0, 1000), 0)).(T)`,
		},
		"types.HStore": {
			Imports: importers.List{`"github.com/stephenafamo/bob/types"`},
			RandomExpr: `hs := make(types.HStore)
                for i := 0; i < f.IntBetween(1, 5); i++ {
                    arr[random[string](f)] = randomNull[string](f)
                }
                return any(hs).(T)`,
		},
		"types.JSON[json.RawMessage]": {
			Imports: importers.List{
				`"encoding/json"`,
				`"github.com/stephenafamo/bob/types"`,
			},
			RandomExpr: `s := &bytes.Buffer{}
                s.WriteRune('{')
                for i := 0; i < f.IntBetween(1, 5); i++ {
                    if i > 0 {
                        fmt.Fprint(s, ", ")
                    }
                    fmt.Fprintf(s, "%q:%q", f.Lorem().Word(), f.Lorem().Word())
                }
                s.WriteRune('}')
                return any(types.NewJSON[json.RawMessage](s.Bytes())).(T)`,
			RandomExprImports:  importers.List{`"fmt"`, `"bytes"`},
			CompareExpr:        `bytes.Equal(AAA.Val, BBB.Val)`,
			CompareExprImports: importers.List{`"bytes"`},
		},
	}
}

func GetFreePort() (int, error) {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("resolve localhost:0: %w", err)
	}

	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return 0, fmt.Errorf("listen on localhost:0: %w", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

func Migrate(ctx context.Context, db *sql.DB, dir fs.FS) error {
	err := fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := fs.ReadFile(dir, path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		fmt.Printf("migrating %s...\n", path)
		if _, err = db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("migrating %s: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("migrations finished\n")
	return nil
}
