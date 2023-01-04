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

	"github.com/spf13/viper"
	"golang.org/x/mod/modfile"
)

func ReadConfig(configFile string) {
	if len(configFile) != 0 {
		viper.SetConfigFile(configFile)
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

func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return ""
}

func ModelsPackage(relPath string) (string, error) {
	modFile, err := goModInfo()
	if err != nil {
		return "", fmt.Errorf("getting mod details: %w", err)
	}

	return path.Join(modFile.Module.Mod.Path, relPath), nil
}

// goModInfo returns the main module's root directory
// and the parsed contents of the go.mod file.
func goModInfo() (*modfile.File, error) {
	goModPath, err := findGoMod()
	if err != nil {
		return nil, fmt.Errorf("cannot find main module: %w", err)
	}

	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read main go.mod file: %w", err)
	}

	modf, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("could not parse go.mod: %w", err)
	}

	return modf, nil
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
