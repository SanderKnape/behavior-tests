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

type funcInfo struct {
	body      string
	hasAssert bool
}

func main() {
	ref := "HEAD"
	if len(os.Args) > 1 {
		ref = os.Args[1]
	}

	oldFuncs := parseDirAtRef(ref)   // map[string]string — body only for comparison
	newFuncs := parseDirOnDisk()     // map[string]funcInfo — body + assertion flag

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

	var warnings []string

	for name, info := range newFuncs {
		d := domainOf(name)
		ensure(d)
		if oldBody, exists := oldFuncs[name]; !exists {
			byDomain[d].added = append(byDomain[d].added, name)
			if !info.hasAssert {
				warnings = append(warnings, name)
			}
		} else if info.body != oldBody {
			byDomain[d].modified = append(byDomain[d].modified, name)
			if !info.hasAssert {
				warnings = append(warnings, name)
			}
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

	noAssert := map[string]bool{}
	for _, name := range warnings {
		noAssert[name] = true
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
			if noAssert[name] {
				fmt.Printf("- [+] `%s` ⚠️ no assertions\n", name)
			} else {
				fmt.Printf("- [+] `%s`\n", name)
			}
		}
		for _, name := range diff.modified {
			if noAssert[name] {
				fmt.Printf("- [~] `%s` ⚠️ no assertions\n", name)
			} else {
				fmt.Printf("- [~] `%s`\n", name)
			}
		}
		for _, name := range diff.removed {
			fmt.Printf("- [-] `%s`\n", name)
		}
		fmt.Println()
	}

	if len(warnings) > 0 {
		sort.Strings(warnings)
		fmt.Println("---")
		fmt.Println()
		fmt.Printf("⚠️ %d test(s) added or modified without assertions:\n", len(warnings))
		for _, name := range warnings {
			fmt.Printf("- `%s`\n", name)
		}
		os.Exit(1)
	}
}

func parseDirOnDisk() map[string]funcInfo {
	entries, err := os.ReadDir(behaviorDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", behaviorDir, err)
		os.Exit(1)
	}

	funcs := map[string]funcInfo{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(behaviorDir, entry.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", entry.Name(), err)
			continue
		}
		for name, info := range parseBehaviorFuncInfos(src) {
			funcs[name] = info
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
		src, err := gitShow(ref, name)
		if err != nil {
			continue
		}
		for fname, info := range parseBehaviorFuncInfos([]byte(src)) {
			funcs[fname] = info.body
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

func parseBehaviorFuncInfos(src []byte) map[string]funcInfo {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return map[string]funcInfo{}
	}

	funcs := map[string]funcInfo{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestBehavior_") {
			continue
		}
		funcs[fn.Name.Name] = funcInfo{
			body:      printNode(fset, fn.Body),
			hasAssert: hasAssertions(fn.Body),
		}
	}
	return funcs
}

// hasAssertions reports whether the function body contains any direct assertion
// call: t.Fatal*, t.Error*, t.Fail*, or any call on require.* / assert.*.
func hasAssertions(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		switch recv.Name {
		case "t":
			switch sel.Sel.Name {
			case "Fatal", "Fatalf", "Error", "Errorf", "Fail", "FailNow":
				found = true
			}
		case "require", "assert":
			found = true
		}
		return true
	})
	return found
}

func printNode(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, node) //nolint:errcheck
	return buf.String()
}
