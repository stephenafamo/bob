package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/stephenafamo/bob/gen"
	"golang.org/x/mod/modfile"
)

func GetConfig[T any](configPath, driverConfigKey string, driverDefaults map[string]any) (gen.Config, T, error) {
	var config gen.Config
	var driverConfig T

	k := koanf.New(".")

	// Add some defaults
	if err := k.Load(confmap.Provider(map[string]any{
		"wipe":              true,
		"struct_tag_casing": "snake",
		"relation_tag":      "-",
		"generator":         fmt.Sprintf("BobGen %s %s", driverConfigKey, Version()),
		driverConfigKey:     driverDefaults,
	}, "."), nil); err != nil {
		return config, driverConfig, err
	}

	if configPath != "" {
		// Load YAML config and merge into the previously loaded config (because we can).
		err := k.Load(file.Provider(configPath), yaml.Parser())
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return config, driverConfig, err
			}

			fmt.Printf("No such file: %q\n", configPath)
		}
	}

	// Load env variables for ONLY driver config
	envKey := strings.ToUpper(driverConfigKey) + "_"
	if err := k.Load(env.Provider(envKey, ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	}), nil); err != nil {
		return config, driverConfig, err
	}

	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return config, driverConfig, err
	}

	if err := k.UnmarshalWithConf(driverConfigKey, &driverConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return config, driverConfig, err
	}

	return config, driverConfig, nil
}

func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return ""
}

func ModelsPackage(modelsFolder string) (string, error) {
	modRoot, modFile, err := GoModInfo()
	if err != nil {
		return "", fmt.Errorf("getting mod details: %w", err)
	}

	relPath := modelsFolder
	if filepath.IsAbs(modelsFolder) {
		relPath = strings.TrimPrefix(modelsFolder, modRoot)
	}

	return path.Join(modFile.Module.Mod.Path, relPath), nil
}

// GoModInfo returns the main module's root directory
// and the parsed contents of the go.mod file.
func GoModInfo() (string, *modfile.File, error) {
	goModPath, err := findGoMod()
	if err != nil {
		return "", nil, fmt.Errorf("cannot find main module: %w", err)
	}

	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", nil, fmt.Errorf("cannot read main go.mod file: %w", err)
	}

	modf, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", nil, fmt.Errorf("could not parse go.mod: %w", err)
	}

	return filepath.Dir(goModPath), modf, nil
}

func findGoMod() (string, error) {
	var outData, errData bytes.Buffer

	c := exec.Command("go", "env", "GOMOD")
	c.Stdout = &outData
	c.Stderr = &errData
	c.Dir = "."
	err := c.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && errData.Len() > 0 {
			return "", errors.New(strings.TrimSpace(errData.String()))
		}

		return "", fmt.Errorf("cannot run go env GOMOD: %w", err)
	}

	out := strings.TrimSpace(outData.String())
	if out == "" {
		return "", errors.New("no go.mod file found in any parent directory")
	}

	return out, nil
}
