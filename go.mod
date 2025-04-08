module github.com/stephenafamo/bob

go 1.23.0

require (
	github.com/DATA-DOG/go-txdb v0.1.6
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/aarondl/opt v0.0.0-20230114172057-b91f370c41f0
	github.com/antlr4-go/antlr/v4 v4.13.1
	github.com/fergusstrange/embedded-postgres v1.26.0
	github.com/go-sql-driver/mysql v1.7.2-0.20231213112541-0004702b931d
	github.com/google/go-cmp v0.6.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/knadh/koanf/parsers/yaml v0.1.0
	github.com/knadh/koanf/providers/confmap v0.1.0
	github.com/knadh/koanf/providers/env v0.1.0
	github.com/knadh/koanf/providers/file v0.1.0
	github.com/knadh/koanf/v2 v2.1.0
	github.com/lib/pq v1.10.7
	github.com/nsf/jsondiff v0.0.0-20210926074059-1e845ec5d249
	github.com/qdm12/reprint v0.0.0-20200326205758-722754a53494
	github.com/stephenafamo/scan v0.6.2
	github.com/stephenafamo/sqlparser v0.0.0-20250408111851-b937299b5b7d
	github.com/tursodatabase/libsql-client-go v0.0.0-20240902231107-85af5b9d094d
	github.com/urfave/cli/v2 v2.23.7
	github.com/volatiletech/strmangle v0.0.6
	github.com/wasilibs/go-pgquery v0.0.0-20240319230125-b9b2e95c69a7
	golang.org/x/mod v0.24.0
	golang.org/x/text v0.17.0
	golang.org/x/tools v0.31.0
	modernc.org/sqlite v1.20.3
	mvdan.cc/gofumpt v0.7.0
)

require (
	dario.cat/mergo v1.0.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/aarondl/json v0.0.0-20221020222930-8b0db17ef1bf // indirect
	github.com/coder/websocket v1.8.12 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.0.0-alpha.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pganalyze/pg_query_go/v5 v5.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	github.com/tetratelabs/wazero v1.7.0 // indirect
	github.com/volatiletech/inflect v0.0.1 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/uint128 v1.2.0 // indirect
	modernc.org/cc/v3 v3.41.0 // indirect
	modernc.org/ccgo/v3 v3.17.0 // indirect
	modernc.org/libc v1.49.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)

// replace github.com/pingcap/tidb => github.com/pingcap/tidb v1.1.0-beta.0.20230311041313-145b7cdf72fe
// replace github.com/stephenafamo/sqlparser => ../sqlparser
