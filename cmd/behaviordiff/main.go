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
	"sort"
	"strings"
)

const behaviorFile = "cmd/api/behavior_integration_test.go"

func main() {
	ref := "HEAD"
	if len(os.Args) > 1 {
		ref = os.Args[1]
	}

	oldSrc, err := gitShow(ref, behaviorFile)
	if err != nil {
		// File didn't exist at that ref — all current tests are new
		oldSrc = ""
	}

	newSrc, err := os.ReadFile(behaviorFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", behaviorFile, err)
		os.Exit(1)
	}

	oldFuncs := parseBehaviorFuncs([]byte(oldSrc))
	newFuncs := parseBehaviorFuncs(newSrc)

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
			fmt.Printf(" + `%s`\n", name)
		}
		for _, name := range diff.modified {
			fmt.Printf(" ~ `%s`\n", name)
		}
		for _, name := range diff.removed {
			fmt.Printf(" - `%s`\n", name)
		}
		fmt.Println()
	}
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
