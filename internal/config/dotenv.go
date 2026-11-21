package config

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// DefaultDotEnvPath is the file LoadDotEnv reads by default, relative to the
// working directory.
const DefaultDotEnvPath = ".env"

// LoadDotEnv reads KEY=VALUE lines from path into the process environment.
// Variables that are already set are left alone, so the real environment
// always wins. A missing file is not an error; it reports whether a file was
// read.
//
// Supported syntax: blank lines, # comments, an optional "export " prefix,
// values in matching single or double quotes, and " # comment" after an
// unquoted value. There is no variable expansion or multi-line value.
func LoadDotEnv(path string) (bool, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("config: open %s: %w", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		key, value, ok, err := parseDotEnvLine(sc.Text())
		if err != nil {
			// Report the line number only: the line may contain a secret.
			return true, fmt.Errorf("config: %s line %d: %w", path, n, err)
		}
		if !ok {
			continue
		}
		if _, set := os.LookupEnv(key); set {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return true, fmt.Errorf("config: %s line %d: set %s: %w", path, n, key, err)
		}
	}
	if err := sc.Err(); err != nil {
		return true, fmt.Errorf("config: read %s: %w", path, err)
	}
	return true, nil
}

func parseDotEnvLine(line string) (key, value string, ok bool, err error) {
	line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}
	line = strings.TrimPrefix(line, "export ")

	key, value, found := strings.Cut(line, "=")
	key = strings.TrimSpace(key)
	if !found || key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false, errors.New("expected KEY=VALUE")
	}

	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
		if value[len(value)-1] != value[0] {
			return "", "", false, errors.New("unterminated quoted value")
		}
		return key, value[1 : len(value)-1], true, nil
	}
	if i := strings.Index(value, " #"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	return key, value, true, nil
}
