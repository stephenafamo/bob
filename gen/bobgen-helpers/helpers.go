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

func EnumType(types drivers.Types, enum string) string {
	types[enum] = drivers.Type{
		NoRandomizationTest: true, // enums are often not random enough
		RandomExpr: fmt.Sprintf(`all := all%s()
            return all[f.IntBetween(0, len(all)-1)]`, enum),
	}

	return enum
}

const pgtypesImport = `"github.com/stephenafamo/bob/types/pgtypes"`

func AddPgEnumArrayType(types drivers.Types, enumTyp string) string {
	arrTyp := fmt.Sprintf("pgtypes.EnumArray[%s]", enumTyp)

	// premptively add the enum type
	// this is to prevent issues if the enum is only used in an array
	EnumType(types, enumTyp)

	types[arrTyp] = drivers.Type{
		DependsOn:           []string{enumTyp},
		Imports:             importers.List{pgtypesImport},
		NoRandomizationTest: true, // enums are often not random enough
		RandomExpr: fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random_%s(f)
            }
            return arr`, arrTyp, gen.NormalizeType(enumTyp)),
	}

	return arrTyp
}

func AddPgGenericArrayType(types drivers.Types, singleTyp string) string {
	singleTypDef := types[singleTyp]
	singleComparer := strings.ReplaceAll(singleTypDef.CompareExpr, "AAA", "a")
	singleComparer = strings.ReplaceAll(singleComparer, "BBB", "b")
	if singleComparer == "" {
		singleComparer = "a == b"
	}

	typ := fmt.Sprintf("pgtypes.Array[%s]", singleTyp)

	types[typ] = drivers.Type{
		DependsOn: []string{singleTyp},
		Imports:   append(importers.List{pgtypesImport}, singleTypDef.Imports...),
		RandomExpr: fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random_%s(f)
            }
            return arr`, typ, gen.NormalizeType(singleTyp)),
		CompareExpr: fmt.Sprintf(`slices.EqualFunc(AAA, BBB, func(a, b %s) bool {
                return %s
            })`, singleTyp, singleComparer),
		CompareExprImports: append(append(
			importers.List{`"slices"`},
			singleTypDef.CompareExprImports...),
			singleTypDef.Imports...),
	}

	return typ
}

func Types() drivers.Types {
	return drivers.Types{
		"bool": {
			NoRandomizationTest: true,
			RandomExpr:          `return f.Bool()`,
		},
		"int": {
			RandomExpr: `return f.Int()`,
		},
		"int8": {
			RandomExpr: `return f.Int8()`,
		},
		"int16": {
			RandomExpr: `return f.Int16()`,
		},
		"int32": {
			RandomExpr: `return f.Int32()`,
		},
		"int64": {
			RandomExpr: `return f.Int64()`,
		},
		"uint": {
			RandomExpr: `return f.UInt()`,
		},
		"uint8": {
			RandomExpr: `return f.UInt8()`,
		},
		"uint16": {
			RandomExpr: `return f.UInt16()`,
		},
		"uint32": {
			RandomExpr: `return f.UInt32()`,
		},
		"uint64": {
			RandomExpr: `return f.UInt64()`,
		},
		"float32": {
			RandomExpr: `return f.Float32(10, -1_000_000, 1_000_000)`,
		},
		"float64": {
			RandomExpr: `return f.Float64(10, -1_000_000, 1_000_000)`,
		},
		"string": {
			RandomExpr:        `return strings.Join(f.Lorem().Words(f.IntBetween(1, 5)), " ")`,
			RandomExprImports: importers.List{`"strings"`},
		},
		"[]byte": {
			DependsOn:           []string{"string"},
			RandomExpr:          `return []byte(random_string(f))`,
			CompareExpr:         `bytes.Equal(AAA, BBB)`,
			CompareExprImports:  importers.List{`"bytes"`},
			NoScannerValuerTest: true,
		},
		"time.Time": {
			Imports: importers.List{`"time"`},
			RandomExpr: `year := time.Hour * 24 * 365
                min := time.Now().Add(-year)
                max := time.Now().Add(year)
                return f.Time().TimeBetween(min, max)`,
			CompareExpr:         `AAA.Equal(BBB)`,
			NoScannerValuerTest: true,
		},
		"types.Text[netip.Addr, *netip.Addr]": {
			Imports: importers.List{
				`"net/netip"`,
				`"github.com/stephenafamo/bob/types"`,
			},
			RandomExpr: `var addr [4]byte
                rand.Read(addr[:])
                ipAddr := netip.AddrFrom4(addr)
                return types.Text[netip.Addr, *netip.Addr]{Val: ipAddr}`,
			RandomExprImports: importers.List{`"crypto/rand"`},
		},
		"pgtypes.Inet": {
			Imports: importers.List{
				`"github.com/stephenafamo/bob/types/pgtypes"`,
			},
			RandomExpr: `var addr [4]byte
                rand.Read(addr[:])
                ipAddr := netip.AddrFrom4(addr)
                ipPrefix := netip.PrefixFrom(ipAddr, f.IntBetween(0, ipAddr.BitLen()))
                return pgtypes.Inet{Prefix: ipPrefix}`,
			RandomExprImports: importers.List{`"crypto/rand"`},
		},
		"pgtypes.Macaddr": {
			Imports: importers.List{`"github.com/stephenafamo/bob/types/pgtypes"`},
			RandomExpr: `addr, _ := net.ParseMAC(f.Internet().MacAddress())
                return pgtypes.Macaddr{Addr: addr}`,
			RandomExprImports:  importers.List{`"net"`},
			CompareExpr:        `slices.Equal(AAA.Addr, BBB.Addr)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pq.BoolArray": {
			Imports: importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.BoolArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = f.Bool()
                }
                return arr`,
			NoRandomizationTest: true,
		},
		"pq.Int64Array": {
			Imports: importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.Int64Array, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = f.Int64()
                }
                return arr`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pq.ByteaArray": {
			DependsOn: []string{"[]byte"},
			Imports:   importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.ByteaArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random___byte(f)
                }
                return arr`,
			CompareExpr: `slices.EqualFunc(AAA, BBB, func(a, b []byte) bool {
                return bytes.Equal(a, b)
            })`,
			CompareExprImports: importers.List{`"slices"`, `"bytes"`},
		},
		"pq.StringArray": {
			DependsOn: []string{"string"},
			Imports:   importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.StringArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random_string(f)
                }
                return arr`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pq.Float64Array": {
			Imports: importers.List{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.Float64Array, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = f.Float64()
                }
                return arr`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pgeo.Box": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandBox()`,
		},
		"pgeo.Circle": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandCircle()`,
		},
		"pgeo.Line": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandLine()`,
		},
		"pgeo.Lseg": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandLseg()`,
		},
		"pgeo.Path": {
			Imports:     importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr:  `return pgeo.NewRandPath()`,
			CompareExpr: `AAA.Closed == BBB.Closed && slices.Equal(AAA.Points, BBB.Points)`,
		},
		"pgeo.Point": {
			Imports:    importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandPoint()`,
		},
		"pgeo.Polygon": {
			Imports:            importers.List{`"github.com/saulortega/pgeo"`},
			RandomExpr:         `return pgeo.NewRandPolygon()`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"decimal.Decimal": {
			Imports:    importers.List{`"github.com/shopspring/decimal"`},
			RandomExpr: `return decimal.New(f.Int64Between(0, 1000), 0)`,
		},
		"pgtypes.LSN": {
			Imports:    importers.List{`"github.com/stephenafamo/bob/types/pgtypes"`},
			RandomExpr: `return pgtypes.LSN(f.UInt64())`,
		},
		"pgtypes.TxIDSnapshot": {
			Imports: importers.List{`"github.com/stephenafamo/bob/types/pgtypes"`},
			RandomExpr: `active := make([]string, f.IntBetween(1, 5))
                for i := range active {
                    active[i] = strconv.FormatUint(f.UInt64(), 10)
                }
                return pgtypes.TxIDSnapshot{
                    Min: strconv.FormatUint(f.UInt64(), 10),
                    Max: strconv.FormatUint(f.UInt64(), 10),
                    Active: active,
                }`,
			RandomExprImports:  importers.List{`"strconv"`},
			CompareExpr:        `AAA.Min == BBB.Min && AAA.Max == BBB.Max && slices.Equal(AAA.Active, BBB.Active)`,
			CompareExprImports: importers.List{`"slices"`},
		},
		"pgtypes.HStore": {
			DependsOn: []string{"string"},
			Imports:   importers.List{`"github.com/stephenafamo/bob/types/pgtypes"`},
			RandomExpr: `hs := make(pgtypes.HStore)
                for i := 0; i < f.IntBetween(1, 5); i++ {
                    arr[random_string(f)] = null.FromCond(random_string(f), f.Bool())
                }
                return hs`,
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
                return types.NewJSON[json.RawMessage](s.Bytes())`,
			RandomExprImports:  importers.List{`"fmt"`, `"bytes"`},
			CompareExpr:        `bytes.Equal(AAA.Val, BBB.Val)`,
			CompareExprImports: importers.List{`"bytes"`},
		},
		"xml": {
			AliasOf:   "string",
			DependsOn: []string{"string"},
			RandomExpr: `tag := f.Lorem().Word()
      return fmt.Sprintf("<%s>%s</%s>", tag, f.Lorem().Word(), tag)`,
			RandomExprImports: importers.List{`"fmt"`},
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
	return MigrateWithOptions(ctx, db, dir, MigrationOpts{})
}

type MigrationOpts struct {
	SplitFileIntoStatements bool
}

func MigrateWithOptions(ctx context.Context, db *sql.DB, dir fs.FS, opts MigrationOpts) error {
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

		stmts := []string{string(content)}

		if opts.SplitFileIntoStatements {
			stmts = strings.Split(string(content), ";")
		}

		for _, stmt := range stmts {
			if _, err = db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrating %s: %w", path, err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("migrations finished\n")
	return nil
}
