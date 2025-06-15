package helpers

import "github.com/stephenafamo/bob/gen/drivers"

//nolint:maintidx
func Types() drivers.Types {
	m := map[string]drivers.Type{
		"bool": {
			NoRandomizationTest: true,
			RandomExpr:          `return f.Bool()`,
		},
		"int": {
			RandomExpr: `return f.Int()`,
		},
		"int8": {
			NoRandomizationTest: true,
			RandomExpr:          `return f.Int8()`,
		},
		"int16": {
			RandomExpr: `return f.Int16()`,
		},
		"int32": {
			RandomExpr: `return f.Int32()`,
		},
		"rune": {
			RandomExpr: `return f.Int32()`,
		},
		"int64": {
			RandomExpr: `return f.Int64()`,
		},
		"uint": {
			RandomExpr: `return f.UInt()`,
		},
		"uint8": {
			NoRandomizationTest: true,
			RandomExpr:          `return f.UInt8()`,
		},
		"byte": {
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
			RandomExpr: `
				var precision int64 = 5 
				var scale int64 = 2 

				if len(limits) > 0 {
					precision, _ = strconv.ParseInt(limits[0], 10, 32)
				}

				if len(limits) > 1 {
					scale, _ = strconv.ParseInt(limits[1], 10, 32)
				}

				scaleFloat := math.Pow10(int(scale))

				val := f.Float64(10, -1, 1)*math.Pow10(int(precision))
				val = math.Trunc(val)/scaleFloat

				return float32(val)
			`,
			RandomExprImports: []string{`"strconv"`, `"math"`},
		},
		"float64": {
			RandomExpr: `
				var precision int64 = 5 
				var scale int64 = 2 

				if len(limits) > 0 {
					precision, _ = strconv.ParseInt(limits[0], 10, 32)
				}

				if len(limits) > 1 {
					scale, _ = strconv.ParseInt(limits[1], 10, 32)
				}

				scaleFloat := math.Pow10(int(scale))

				val := f.Float64(10, -1, 1)*math.Pow10(int(precision))
				val = math.Trunc(val)/scaleFloat

				return val
			`,
			RandomExprImports: []string{`"strconv"`, `"math"`},
		},
		"string": {
			RandomExpr: `
			val := strings.Join(f.Lorem().Words(f.IntBetween(1, 5)), " ")
			if len(limits) == 0 {
				return val
			}
			limitInt, _ := strconv.Atoi(limits[0])
			if limitInt > 0 && limitInt < len(val) {
				val = val[:limitInt]
			}
			return val
			`,
			RandomExprImports: []string{`"strconv"`, `"strings"`},
		},
		"[]byte": {
			DependsOn:           []string{"string"},
			RandomExpr:          `return []byte(random_string(f, limits...))`,
			CompareExpr:         `bytes.Equal(AAA, BBB)`,
			CompareExprImports:  []string{`"bytes"`},
			NoScannerValuerTest: true,
		},
		"time.Time": {
			Imports: []string{`"time"`},
			RandomExpr: `year := time.Hour * 24 * 365
                min := time.Now().Add(-year)
                max := time.Now().Add(year)
                return f.Time().TimeBetween(min, max)`,
			CompareExpr:         `AAA.Equal(BBB)`,
			NoScannerValuerTest: true,
		},
		"types.Time": {
			Imports:   []string{`"github.com/stephenafamo/bob/types"`},
			DependsOn: []string{"time.Time"},
			RandomExpr: `
				return types.Time{Time: random_time_Time(f, limits...)}`,
			CompareExpr:         `AAA.Time.Equal(BBB.Time)`,
			NoScannerValuerTest: true,
		},
		"types.Text[netip.Addr, *netip.Addr]": {
			Imports: []string{
				`"net/netip"`,
				`"github.com/stephenafamo/bob/types"`,
			},
			RandomExpr: `var addr [4]byte
                rand.Read(addr[:])
                ipAddr := netip.AddrFrom4(addr)
                return types.Text[netip.Addr, *netip.Addr]{Val: ipAddr}`,
			RandomExprImports: []string{`"crypto/rand"`},
		},
		"types.Text[netip.Prefix, *netip.Prefix]": {
			Imports: []string{
				`"net/netip"`,
				`"github.com/stephenafamo/bob/types"`,
			},
			RandomExpr: `var addr [4]byte
                rand.Read(addr[:])
                ipAddr := netip.AddrFrom4(addr)
                ipPrefix := netip.PrefixFrom(ipAddr, ipAddr.BitLen())
                return types.Text[netip.Prefix, *netip.Prefix]{Val: ipPrefix}`,
			RandomExprImports: []string{`"crypto/rand"`},
		},
		"pgtypes.Inet": {
			Imports: []string{
				`"github.com/stephenafamo/bob/types/pgtypes"`,
			},
			RandomExpr: `var addr [4]byte
                rand.Read(addr[:])
                ipAddr := netip.AddrFrom4(addr)
                ipPrefix := netip.PrefixFrom(ipAddr, f.IntBetween(0, ipAddr.BitLen()))
                return pgtypes.Inet{Prefix: ipPrefix}`,
			RandomExprImports: []string{`"crypto/rand"`, `"net/netip"`},
		},
		"pgtypes.Macaddr": {
			Imports: []string{`"github.com/stephenafamo/bob/types/pgtypes"`},
			RandomExpr: `addr, _ := net.ParseMAC(f.Internet().MacAddress())
                return pgtypes.Macaddr{Addr: addr}`,
			RandomExprImports:  []string{`"net"`},
			CompareExpr:        `slices.Equal(AAA.Addr, BBB.Addr)`,
			CompareExprImports: []string{`"slices"`},
		},
		"pq.BoolArray": {
			DependsOn: []string{"bool"},
			Imports:   []string{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.BoolArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random_bool(f, limits...)
                }
                return arr`,
			NoRandomizationTest: true,
		},
		"pq.Int32Array": {
			DependsOn: []string{"int32"},
			Imports:   []string{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.Int32Array, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random_int32(f, limits...)
                }
                return arr`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: []string{`"slices"`},
		},
		"pq.Int64Array": {
			DependsOn: []string{"int64"},
			Imports:   []string{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.Int64Array, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random_int64(f, limits...)
                }
                return arr`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: []string{`"slices"`},
		},
		"pq.ByteaArray": {
			DependsOn: []string{"[]byte"},
			Imports:   []string{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.ByteaArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random___byte(f, limits...)
                }
                return arr`,
			CompareExpr: `slices.EqualFunc(AAA, BBB, func(a, b []byte) bool {
                return bytes.Equal(a, b)
            })`,
			CompareExprImports: []string{`"slices"`, `"bytes"`},
		},
		"pq.StringArray": {
			DependsOn: []string{"string"},
			Imports:   []string{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.StringArray, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random_string(f, limits...)
                }
                return arr`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: []string{`"slices"`},
		},
		"pq.Float64Array": {
			DependsOn: []string{"float64"},
			Imports:   []string{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.Float64Array, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random_float64(f, limits...)
                }
                return arr`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: []string{`"slices"`},
		},
		"pq.Float32Array": {
			DependsOn: []string{"float32"},
			Imports:   []string{`"github.com/lib/pq"`},
			RandomExpr: `arr := make(pq.Float32Array, f.IntBetween(1, 5))
                for i := range arr {
                    arr[i] = random_float32(f, limits...)
                }
                return arr`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: []string{`"slices"`},
		},
		"pgeo.Box": {
			Imports:    []string{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandBox()`,
		},
		"pgeo.Circle": {
			Imports:    []string{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandCircle()`,
		},
		"pgeo.Line": {
			Imports:    []string{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandLine()`,
		},
		"pgeo.Lseg": {
			Imports:    []string{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandLseg()`,
		},
		"pgeo.Path": {
			Imports:     []string{`"github.com/saulortega/pgeo"`},
			RandomExpr:  `return pgeo.NewRandPath()`,
			CompareExpr: `AAA.Closed == BBB.Closed && slices.Equal(AAA.Points, BBB.Points)`,
		},
		"pgeo.Point": {
			Imports:    []string{`"github.com/saulortega/pgeo"`},
			RandomExpr: `return pgeo.NewRandPoint()`,
		},
		"pgeo.Polygon": {
			Imports:            []string{`"github.com/saulortega/pgeo"`},
			RandomExpr:         `return pgeo.NewRandPolygon()`,
			CompareExpr:        `slices.Equal(AAA, BBB)`,
			CompareExprImports: []string{`"slices"`},
		},
		"decimal.Decimal": {
			Imports: []string{`"github.com/shopspring/decimal"`},
			RandomExpr: `
			var precision int64 = 5 
			var scale int64 = 2 

			if len(limits) > 0 {
				precision, _ = strconv.ParseInt(limits[0], 10, 32)
			}

			if len(limits) > 1 {
				scale, _ = strconv.ParseInt(limits[1], 10, 32)
			}

			precisionDecimal, _ := decimal.NewFromInt(10).PowInt32(int32(precision))
			return decimal.
				NewFromFloat32(f.Float32(10, -1, 1)).
				Mul(precisionDecimal).
				Shift(int32(-1 * scale)).
				RoundDown(int32(scale))
			`,
			RandomExprImports: []string{`"strconv"`},
			CompareExpr:       `AAA.Equal(BBB)`,
		},
		"pgtypes.LSN": {
			Imports:    []string{`"github.com/stephenafamo/bob/types/pgtypes"`},
			RandomExpr: `return pgtypes.LSN(f.UInt64())`,
		},
		"pgtypes.Snapshot": {
			Imports: []string{`"github.com/stephenafamo/bob/types/pgtypes"`},
			RandomExpr: `
				min := f.UInt32()
				max := f.UInt32Between(min, math.MaxUint32)

				active := make([]uint64, f.IntBetween(0, 5))
				for i := range active {
					if i == 0 {
						active[i] = uint64(f.UInt32Between(min, max))
					} else {
						active[i] = uint64(f.UInt32Between(uint32(active[i-1]), max))
					}
				}
				return pgtypes.Snapshot{
					Min:    uint64(min),
					Max:    uint64(max),
					Active: active,
				}
			`,
			RandomExprImports:  []string{`"math"`, `"strconv"`},
			CompareExpr:        `AAA.Min == BBB.Min && AAA.Max == BBB.Max && slices.Equal(AAA.Active, BBB.Active)`,
			CompareExprImports: []string{`"slices"`},
		},
		"pgtypes.HStore": {
			DependsOn: []string{"string"},
			Imports:   []string{`"github.com/stephenafamo/bob/types/pgtypes"`},
			RandomExpr: `hs := make(pgtypes.HStore)
                for range f.IntBetween(1, 5) {
					hs[random_string(f)] = sql.Null[string]{V: random_string(f, limits...), Valid: f.Bool()}
                }
                return hs`,
			RandomExprImports: []string{`"database/sql"`},
			CompareExpr:       `AAA.String() == BBB.String()`,
		},
		"types.JSON[json.RawMessage]": {
			Imports: []string{
				`"encoding/json"`,
				`"github.com/stephenafamo/bob/types"`,
			},
			RandomExpr: `s := &bytes.Buffer{}
                s.WriteRune('{')
                for i := range f.IntBetween(1, 5) {
                    if i > 0 {
                        fmt.Fprint(s, ", ")
                    }
                    fmt.Fprintf(s, "%q:%q", f.Lorem().Word(), f.Lorem().Word())
                }
                s.WriteRune('}')
                return types.NewJSON[json.RawMessage](s.Bytes())`,
			RandomExprImports:  []string{`"fmt"`, `"bytes"`},
			CompareExpr:        `bytes.Equal(AAA.Val, BBB.Val)`,
			CompareExprImports: []string{`"bytes"`},
		},
		"xml": {
			AliasOf:   "string",
			DependsOn: []string{"string"},
			RandomExpr: `tag := f.Lorem().Word()
      return fmt.Sprintf("<%s>%s</%s>", tag, f.Lorem().Word(), tag)`,
			RandomExprImports: []string{`"fmt"`},
		},
		"money": {
			AliasOf:           "string",
			DependsOn:         []string{"string"},
			RandomExpr:        `return fmt.Sprintf("%.2f", f.Float32(2, 0, 1000))`,
			RandomExprImports: []string{`"fmt"`},
		},
	}

	var types drivers.Types
	types.RegisterAll(m)
	return types
}
