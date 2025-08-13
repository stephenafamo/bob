package plugins

import "github.com/stephenafamo/bob/internal"

//nolint:gochecknoglobals
var PresetAll = Config{
	DBInfo:   OutputConfig{Destination: "dbinfo", Pkgname: "dbinfo"},
	Enums:    OutputConfig{Destination: "enums", Pkgname: "enums"},
	Models:   OutputConfig{Destination: "models", Pkgname: "models"},
	Factory:  OutputConfig{Destination: "factory", Pkgname: "factory"},
	DBErrors: OutputConfig{Destination: "dberrors", Pkgname: "dberrors"},
	Where:    OnOffConfig{},
	Loaders:  OnOffConfig{},
	Joins:    OnOffConfig{},
}

//nolint:gochecknoglobals
var PresetDefault = Config{
	DBInfo:   OutputConfig{Destination: "dbinfo", Pkgname: "dbinfo"},
	Enums:    OutputConfig{Destination: "enums", Pkgname: "enums"},
	Models:   OutputConfig{Destination: "models", Pkgname: "models"},
	Factory:  OutputConfig{Destination: "factory", Pkgname: "factory"},
	DBErrors: OutputConfig{Destination: "dberrors", Pkgname: "dberrors"},
	Where:    OnOffConfig{},
	Loaders:  OnOffConfig{},
	Joins:    OnOffConfig{},
}

//nolint:gochecknoglobals
var PresetNone = Config{
	DBInfo:   OutputConfig{Disabled: internal.Pointer(true)},
	Enums:    OutputConfig{Disabled: internal.Pointer(true)},
	Models:   OutputConfig{Disabled: internal.Pointer(true)},
	Factory:  OutputConfig{Disabled: internal.Pointer(true)},
	DBErrors: OutputConfig{Disabled: internal.Pointer(true)},
	Where:    OnOffConfig{Disabled: internal.Pointer(true)},
	Loaders:  OnOffConfig{Disabled: internal.Pointer(true)},
	Joins:    OnOffConfig{Disabled: internal.Pointer(true)},
}
