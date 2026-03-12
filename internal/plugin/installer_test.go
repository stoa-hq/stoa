package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePackage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"n8n", "github.com/stoa-hq/stoa-plugins/n8n"},
		{"github.com/example/my-plugin", "github.com/example/my-plugin"},
		{"unknown-name", "unknown-name"},
	}
	for _, tt := range tests {
		got := ResolvePackage(tt.input)
		if got != tt.want {
			t.Errorf("ResolvePackage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFindModuleRoot(t *testing.T) {
	// Create a temp dir hierarchy: root/a/b/c with go.mod in root
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(tmp, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := FindModuleRoot(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tmp {
		t.Errorf("FindModuleRoot = %q, want %q", got, tmp)
	}
}

func TestFindModuleRoot_NotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := FindModuleRoot(tmp)
	if err == nil {
		t.Error("expected error when no go.mod exists")
	}
}

func TestInstallerReadWriteImports(t *testing.T) {
	tmp := t.TempDir()
	// Simulate the cmd/stoa directory structure
	cmdDir := filepath.Join(tmp, "cmd", "stoa")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(tmp, "")

	// Initially no file → empty imports
	imports, err := installer.readImports()
	if err != nil {
		t.Fatalf("readImports on missing file: %v", err)
	}
	if len(imports) != 0 {
		t.Errorf("expected empty imports, got %v", imports)
	}

	// Write two imports and read back
	want := []string{
		"github.com/stoa-hq/stoa-plugins/n8n",
		"github.com/example/my-plugin",
	}
	if err := installer.writePluginsFile(want); err != nil {
		t.Fatalf("writePluginsFile: %v", err)
	}
	got, err := installer.readImports()
	if err != nil {
		t.Fatalf("readImports: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d imports, want %d: %v", len(got), len(want), got)
	}
	wantSet := map[string]bool{}
	for _, w := range want {
		wantSet[w] = true
	}
	for _, g := range got {
		if !wantSet[g] {
			t.Errorf("unexpected import %q", g)
		}
	}
}

func TestInstallerWriteEmpty(t *testing.T) {
	tmp := t.TempDir()
	cmdDir := filepath.Join(tmp, "cmd", "stoa")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(tmp, "")

	// Write empty → file must still be valid Go (no import block)
	if err := installer.writePluginsFile(nil); err != nil {
		t.Fatalf("writePluginsFile(nil): %v", err)
	}
	data, err := os.ReadFile(installer.pluginsFile())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !contains(content, "package main") {
		t.Error("expected 'package main' in generated file")
	}
	if contains(content, "import") {
		t.Error("expected no import block when no plugins installed")
	}
}

func TestIsLocalPath(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"./my-plugin", true},
		{"../my-plugin", true},
		{"/home/user/my-plugin", true},
		{"n8n", false},
		{"github.com/example/plugin", false},
	}
	for _, tt := range cases {
		if got := IsLocalPath(tt.input); got != tt.want {
			t.Errorf("IsLocalPath(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestReadModuleName(t *testing.T) {
	tmp := t.TempDir()
	gomod := "module github.com/example/my-plugin\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readModuleName(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github.com/example/my-plugin" {
		t.Errorf("readModuleName = %q, want %q", got, "github.com/example/my-plugin")
	}
}

func TestInternalImportPath(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module github.com/stoa-hq/stoa\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pluginDir := filepath.Join(tmp, "plugins", "meinplugin")
	installer := NewInstaller(tmp, "")

	got, err := installer.internalImportPath(pluginDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "github.com/stoa-hq/stoa/plugins/meinplugin"
	if got != want {
		t.Errorf("internalImportPath = %q, want %q", got, want)
	}
}

func TestResolvePackageLocalPath(t *testing.T) {
	// Local paths must not be looked up in KnownPlugins.
	got := ResolvePackage("./my-local-plugin")
	if got != "./my-local-plugin" {
		t.Errorf("ResolvePackage(local) = %q, want %q", got, "./my-local-plugin")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
