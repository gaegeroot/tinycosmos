package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDotEnv(t *testing.T) {
	const prefix = "TC_DOTENV_TEST_"
	keys := []string{"PLAIN", "DOUBLE", "SINGLE", "EXPORTED", "COMMENTED", "EMPTY", "CRLF", "EXISTING"}
	for _, k := range keys {
		t.Setenv(prefix+k, "") // registers cleanup
		os.Unsetenv(prefix + k)
	}
	t.Setenv(prefix+"EXISTING", "from-shell")

	p := writeFile(t, strings.Join([]string{
		"# a comment",
		"",
		prefix + "PLAIN=123:abc-DEF_ghi",
		prefix + `DOUBLE="has # hash and spaces"`,
		prefix + "SINGLE='single'",
		"export " + prefix + "EXPORTED=yes",
		prefix + "COMMENTED=value # trailing comment",
		prefix + "EMPTY=",
		prefix + "CRLF=windows\r",
		prefix + "EXISTING=from-file",
	}, "\n"))

	read, err := LoadDotEnv(p)
	if err != nil || !read {
		t.Fatalf("LoadDotEnv = %v, %v", read, err)
	}
	want := map[string]string{
		"PLAIN":     "123:abc-DEF_ghi",
		"DOUBLE":    "has # hash and spaces",
		"SINGLE":    "single",
		"EXPORTED":  "yes",
		"COMMENTED": "value",
		"EMPTY":     "",
		"CRLF":      "windows",
		"EXISTING":  "from-shell", // the real environment wins
	}
	for k, v := range want {
		got, set := os.LookupEnv(prefix + k)
		if !set || got != v {
			t.Errorf("%s = %q (set=%v), want %q", k, got, set, v)
		}
	}
}

func TestLoadDotEnvMissingFile(t *testing.T) {
	read, err := LoadDotEnv(filepath.Join(t.TempDir(), "nope"))
	if err != nil || read {
		t.Fatalf("missing file = %v, %v; want false, nil", read, err)
	}
}

func TestLoadDotEnvErrorsDoNotLeakValues(t *testing.T) {
	for name, content := range map[string]string{
		"no equals":          "SUPERSECRETVALUE",
		"unterminated quote": `TOKEN="SUPERSECRETVALUE`,
		"space in key":       "MY KEY=SUPERSECRETVALUE",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := LoadDotEnv(writeFile(t, content))
			if err == nil {
				t.Fatal("want error")
			}
			if strings.Contains(err.Error(), "SUPERSECRETVALUE") {
				t.Fatalf("error leaks the value: %v", err)
			}
			if !strings.Contains(err.Error(), "line 1") {
				t.Fatalf("error should name the line: %v", err)
			}
		})
	}
}
