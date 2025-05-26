package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stephenafamo/bob/gen/drivers"
)

type QueryParser interface {
	ParseQueries(ctx context.Context, s string) ([]drivers.Query, error)
}

func ParseFolders(ctx context.Context, parser QueryParser, paths ...string) ([]drivers.QueryFolder, error) {
	allQueries := make([]drivers.QueryFolder, 0, len(paths))
	for _, path := range paths {
		queries, err := parseFolder(ctx, parser, path)
		if err != nil {
			return nil, fmt.Errorf("parse folder %q: %w", path, err)
		}

		allQueries = append(allQueries, queries)
	}

	return allQueries, nil
}

func parseFolder(ctx context.Context, parser QueryParser, path string) (drivers.QueryFolder, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return drivers.QueryFolder{}, fmt.Errorf("read dir: %w", err)
	}

	files := make([]drivers.QueryFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".bob.sql") {
			continue
		}

		file, err := parseFile(ctx, parser, filepath.Join(path, entry.Name()))
		if err != nil {
			return drivers.QueryFolder{}, fmt.Errorf("parse file %q: %w", entry.Name(), err)
		}

		files = append(files, file)
	}

	return drivers.QueryFolder{
		Path:  path,
		Files: files,
	}, nil
}

func parseFile(ctx context.Context, parser QueryParser, path string) (drivers.QueryFile, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return drivers.QueryFile{}, fmt.Errorf("read file: %w", err)
	}

	queries, err := parser.ParseQueries(ctx, string(file))
	if err != nil {
		return drivers.QueryFile{}, fmt.Errorf("parse queries: %w", err)
	}

	return drivers.QueryFile{
		Path:    path,
		Queries: queries,
	}, nil
}
