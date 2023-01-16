{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import	"_" $.ExtraInfo.Provider.DriverPkg}}
{{if $.ExtraInfo.Provider.DriverENVSource}}{{$.Importer.Import	"os"}}{{end}}

type DB = bob.DB

func New() (DB, error) {
  {{if $.ExtraInfo.Provider.DriverENVSource -}}
    return bob.Open("{{$.ExtraInfo.Provider.DriverName}}", os.Getenv("{{$.ExtraInfo.Provider.DriverENVSource}}"))
  {{- else -}}
    return bob.Open("{{$.ExtraInfo.Provider.DriverName}}", "{{$.ExtraInfo.Provider.DriverSource}}")
  {{- end}}
}

