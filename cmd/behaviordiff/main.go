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

	var added, removed, modified []string

	for name, body := range newFuncs {
		if oldBody, exists := oldFuncs[name]; !exists {
			added = append(added, name)
		} else if body != oldBody {
			modified = append(modified, name)
		}
	}
	for name := range oldFuncs {
		if _, exists := newFuncs[name]; !exists {
			removed = append(removed, name)
		}
	}

	if len(added)+len(removed)+len(modified) == 0 {
		fmt.Println("No behavior test changes.")
		return
	}

	fmt.Println("## Behavior Changes")
	fmt.Println()

	if len(added) > 0 {
		fmt.Println("### Added")
		for _, name := range added {
			fmt.Printf("- `%s`\n", name)
		}
		fmt.Println()
	}

	if len(modified) > 0 {
		fmt.Println("### Modified")
		for _, name := range modified {
			fmt.Printf("- `%s`\n", name)
		}
		fmt.Println()
	}

	if len(removed) > 0 {
		fmt.Println("### Removed")
		for _, name := range removed {
			fmt.Printf("- `%s`\n", name)
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
