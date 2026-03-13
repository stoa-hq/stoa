package plugin

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// KnownPlugins maps short names to their full Go import paths.
var KnownPlugins = map[string]string{
	"n8n":    "github.com/stoa-hq/stoa-plugins/n8n",
	"stripe": "github.com/stoa-hq/stoa-plugins/stripe",
}

// ResolvePackage resolves a short plugin name or full Go import path.
// Local paths (starting with ./, ../, or /) are returned unchanged.
// Unknown names are returned as-is (treated as a full import path).
func ResolvePackage(nameOrPath string) string {
	if IsLocalPath(nameOrPath) {
		return nameOrPath
	}
	if pkg, ok := KnownPlugins[nameOrPath]; ok {
		return pkg
	}
	return nameOrPath
}

// IsLocalPath reports whether s refers to a local directory.
func IsLocalPath(s string) bool {
	return strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || filepath.IsAbs(s)
}

// readModuleName extracts the module name from the go.mod in dir.
func readModuleName(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("reading go.mod in %s: %w", dir, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in %s/go.mod", dir)
}

// FindModuleRoot walks up from dir until it finds a go.mod file.
// Returns an error when no go.mod is found (i.e. not a Go module root).
func FindModuleRoot(dir string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found; run this command from the Stoa source directory")
		}
		dir = parent
	}
}

// pluginsModFile is the gitignored modfile used for user-specific plugin deps.
// It keeps go.mod and go.sum clean so plugin installations are never committed.
const pluginsModFile = "go.plugins.mod"

// Installer manages plugin installation within a Go module source tree.
type Installer struct {
	moduleRoot string
	binaryPath string
}

// NewInstaller creates an Installer for the given module root.
// binaryPath is the path of the running binary to be replaced after rebuild.
func NewInstaller(moduleRoot, binaryPath string) *Installer {
	return &Installer{moduleRoot: moduleRoot, binaryPath: binaryPath}
}

// ensurePluginsModFile creates go.plugins.mod as a copy of go.mod (and
// go.plugins.sum as a copy of go.sum) if they do not exist yet.
// This gives us an isolated modfile for plugin deps while keeping the
// existing checksums so that `go build -modfile=go.plugins.mod` succeeds.
func (i *Installer) ensurePluginsModFile() error {
	dstMod := filepath.Join(i.moduleRoot, pluginsModFile)
	if _, err := os.Stat(dstMod); err != nil {
		src, err := os.ReadFile(filepath.Join(i.moduleRoot, "go.mod"))
		if err != nil {
			return fmt.Errorf("reading go.mod: %w", err)
		}
		if err := os.WriteFile(dstMod, src, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", pluginsModFile, err)
		}
	}

	pluginsSumFile := strings.TrimSuffix(pluginsModFile, ".mod") + ".sum"
	dstSum := filepath.Join(i.moduleRoot, pluginsSumFile)
	if _, err := os.Stat(dstSum); err != nil {
		src, err := os.ReadFile(filepath.Join(i.moduleRoot, "go.sum"))
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reading go.sum: %w", err)
		}
		if err := os.WriteFile(dstSum, src, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", pluginsSumFile, err)
		}
	}

	return nil
}

// Install fetches the plugin package, adds it to the generated imports file,
// and rebuilds the binary in-place.
// pkg may be a remote import path, a short name, or a local directory path.
func (i *Installer) Install(pkg string) error {
	if IsLocalPath(pkg) {
		return i.installLocal(pkg)
	}
	return i.installRemote(pkg)
}

func (i *Installer) installRemote(pkg string) error {
	if err := i.ensurePluginsModFile(); err != nil {
		return err
	}

	imports, err := i.readImports()
	if err != nil {
		return err
	}
	for _, imp := range imports {
		if imp == pkg {
			return fmt.Errorf("plugin %q is already installed", pkg)
		}
	}

	fmt.Printf("Fetching %s...\n", pkg)
	if err := i.run("go", "get", "-modfile="+pluginsModFile, pkg+"@latest"); err != nil {
		return fmt.Errorf("go get: %w", err)
	}

	if err := i.writePluginsFile(append(imports, pkg)); err != nil {
		return fmt.Errorf("updating plugins file: %w", err)
	}

	return i.rebuild()
}

func (i *Installer) installLocal(localPath string) error {
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// If the path is inside the Stoa module, it is already part of the same
	// module — no replace directive or go get needed.
	if strings.HasPrefix(absPath, i.moduleRoot+string(filepath.Separator)) {
		return i.installInternal(absPath)
	}

	return i.installExternal(absPath)
}

// internalImportPath derives the Go import path for a plugin inside the Stoa module.
func (i *Installer) internalImportPath(absPath string) (string, error) {
	stoaModule, err := readModuleName(i.moduleRoot)
	if err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(i.moduleRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("computing relative path: %w", err)
	}
	return stoaModule + "/" + filepath.ToSlash(relPath), nil
}

// installInternal handles plugins that live inside the Stoa source tree.
// The import path is derived from the Stoa module name + relative path.
func (i *Installer) installInternal(absPath string) error {
	importPath, err := i.internalImportPath(absPath)
	if err != nil {
		return err
	}

	imports, err := i.readImports()
	if err != nil {
		return err
	}
	for _, imp := range imports {
		if imp == importPath {
			return fmt.Errorf("plugin %q is already installed", importPath)
		}
	}

	fmt.Printf("Installing internal plugin %s\n", importPath)
	if err := i.writePluginsFile(append(imports, importPath)); err != nil {
		return fmt.Errorf("updating plugins file: %w", err)
	}

	return i.rebuild()
}

// installExternal handles plugins outside the Stoa source tree.
// A replace directive is added to go.plugins.mod so go.mod stays clean.
func (i *Installer) installExternal(absPath string) error {
	if err := i.ensurePluginsModFile(); err != nil {
		return err
	}

	moduleName, err := readModuleName(absPath)
	if err != nil {
		return err
	}

	imports, err := i.readImports()
	if err != nil {
		return err
	}
	for _, imp := range imports {
		if imp == moduleName {
			return fmt.Errorf("plugin %q is already installed", moduleName)
		}
	}

	relPath, err := filepath.Rel(i.moduleRoot, absPath)
	if err != nil {
		return fmt.Errorf("computing relative path: %w", err)
	}

	fmt.Printf("Linking external plugin %s → %s\n", moduleName, relPath)
	modfile := "-modfile=" + pluginsModFile
	if err := i.run("go", "mod", "edit", modfile, "-require="+moduleName+"@v0.0.0"); err != nil {
		return fmt.Errorf("go mod edit -require: %w", err)
	}
	if err := i.run("go", "mod", "edit", modfile, "-replace="+moduleName+"="+relPath); err != nil {
		return fmt.Errorf("go mod edit -replace: %w", err)
	}

	if err := i.writePluginsFile(append(imports, moduleName)); err != nil {
		return fmt.Errorf("updating plugins file: %w", err)
	}

	return i.rebuild()
}

// Remove removes the plugin import, runs go mod tidy, and rebuilds.
func (i *Installer) Remove(pkg string) error {
	imports, err := i.readImports()
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(imports))
	found := false
	for _, imp := range imports {
		if imp == pkg {
			found = true
			continue
		}
		filtered = append(filtered, imp)
	}
	if !found {
		return fmt.Errorf("plugin %q is not installed", pkg)
	}

	if err := i.writePluginsFile(filtered); err != nil {
		return fmt.Errorf("updating plugins file: %w", err)
	}

	// Drop replace directive if one exists (no-op for remote plugins).
	modfile := "-modfile=" + pluginsModFile
	_ = i.run("go", "mod", "edit", modfile, "-dropreplace="+pkg)

	fmt.Println("Running go mod tidy...")
	if err := i.run("go", "mod", "tidy", modfile); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return i.rebuild()
}

// ListInstalled returns the import paths of all installed plugins.
func (i *Installer) ListInstalled() ([]string, error) {
	return i.readImports()
}

// pluginsFiles returns the paths to all plugins_generated.go files that the
// installer maintains — one per Stoa binary. All files are kept in sync so
// that every binary (stoa, stoa-store-mcp, …) loads the same set of plugins.
func (i *Installer) pluginsFiles() []string {
	return []string{
		filepath.Join(i.moduleRoot, "cmd", "stoa", "plugins_generated.go"),
		filepath.Join(i.moduleRoot, "cmd", "stoa-store-mcp", "plugins_generated.go"),
	}
}

// pluginsFile returns the canonical plugins file used to read the current
// import list (cmd/stoa is the source of truth).
func (i *Installer) pluginsFile() string {
	return i.pluginsFiles()[0]
}

func (i *Installer) readImports() ([]string, error) {
	data, err := os.ReadFile(i.pluginsFile())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading plugins file: %w", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", data, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parsing plugins file: %w", err)
	}

	var imports []string
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, path)
	}
	return imports, nil
}

func (i *Installer) writePluginsFile(imports []string) error {
	for _, path := range i.pluginsFiles() {
		var sb strings.Builder
		sb.WriteString("// Code generated by \"stoa plugin install\". DO NOT EDIT.\n")
		sb.WriteString("package main\n")
		if len(imports) > 0 {
			sb.WriteString("\nimport (\n")
			for _, imp := range imports {
				fmt.Fprintf(&sb, "\t_ %q\n", imp)
			}
			sb.WriteString(")\n")
		}

		src, err := format.Source([]byte(sb.String()))
		if err != nil {
			return fmt.Errorf("formatting plugins file %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", path, err)
		}
		if err := os.WriteFile(path, src, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return nil
}

func (i *Installer) rebuild() error {
	fmt.Printf("Building stoa → %s\n", i.binaryPath)
	args := []string{"build", "-modfile=" + pluginsModFile, "-o", i.binaryPath, "./cmd/stoa"}
	if err := i.run("go", args...); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	fmt.Println("Done. Restart stoa to apply changes.")
	return nil
}

func (i *Installer) run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = i.moduleRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
