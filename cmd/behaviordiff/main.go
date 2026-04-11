package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const behaviorDir = "cmd/api/behavior"

func main() {
	ref := "HEAD"
	if len(os.Args) > 1 {
		ref = os.Args[1]
	}

	oldFuncs := parseDirAtRef(ref)
	newFuncs := parseDirOnDisk()

	type domainDiff struct {
		added    []string
		modified []string
		removed  []string
	}
	byDomain := map[string]*domainDiff{}

	domainOf := func(name string) string {
		parts := strings.SplitN(name, "_", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
		return "Other"
	}
	ensure := func(d string) {
		if _, ok := byDomain[d]; !ok {
			byDomain[d] = &domainDiff{}
		}
	}

	for name, body := range newFuncs {
		d := domainOf(name)
		ensure(d)
		if oldBody, exists := oldFuncs[name]; !exists {
			byDomain[d].added = append(byDomain[d].added, name)
		} else if body != oldBody {
			byDomain[d].modified = append(byDomain[d].modified, name)
		}
	}
	for name := range oldFuncs {
		if _, exists := newFuncs[name]; !exists {
			d := domainOf(name)
			ensure(d)
			byDomain[d].removed = append(byDomain[d].removed, name)
		}
	}

	total := 0
	for _, diff := range byDomain {
		total += len(diff.added) + len(diff.modified) + len(diff.removed)
	}
	if total == 0 {
		fmt.Println("No behavior test changes.")
		return
	}

	domains := make([]string, 0, len(byDomain))
	for d := range byDomain {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	fmt.Println("## Behavior Changes")
	fmt.Println()
	for _, d := range domains {
		diff := byDomain[d]
		sort.Strings(diff.added)
		sort.Strings(diff.modified)
		sort.Strings(diff.removed)

		fmt.Printf("**%s**\n", d)
		for _, name := range diff.added {
			fmt.Printf("- [+] `%s`\n", name)
		}
		for _, name := range diff.modified {
			fmt.Printf("- [~] `%s`\n", name)
		}
		for _, name := range diff.removed {
			fmt.Printf("- [-] `%s`\n", name)
		}
		fmt.Println()
	}
}

func parseDirOnDisk() map[string]string {
	entries, err := os.ReadDir(behaviorDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", behaviorDir, err)
		os.Exit(1)
	}

	funcs := map[string]string{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(behaviorDir, entry.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", entry.Name(), err)
			continue
		}
		for name, body := range parseBehaviorFuncs(src) {
			funcs[name] = body
		}
	}
	return funcs
}

func parseDirAtRef(ref string) map[string]string {
	out, err := exec.Command("git", "ls-tree", "--name-only", ref, behaviorDir+"/").Output()
	if err != nil {
		// Directory didn't exist at that ref — treat all current tests as new
		return map[string]string{}
	}

	funcs := map[string]string{}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name == "" || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := gitShow(ref, behaviorDir+"/"+name)
		if err != nil {
			continue
		}
		for fname, body := range parseBehaviorFuncs([]byte(src)) {
			funcs[fname] = body
		}
	}
	return funcs
}

func gitShow(ref, path string) (string, error) {
	out, err := exec.Command("git", "show", fmt.Sprintf("%s:%s", ref, path)).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func parseBehaviorFuncs(src []byte) map[string]string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return map[string]string{}
	}

	funcs := map[string]string{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestBehavior_") {
			continue
		}
		funcs[fn.Name.Name] = printNode(fset, fn.Body)
	}
	return funcs
}

func printNode(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, node) //nolint:errcheck
	return buf.String()
}
