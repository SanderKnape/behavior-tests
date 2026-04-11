package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type pkgStats struct {
	total   int
	covered int
}

func (s *pkgStats) pct() float64 {
	if s.total == 0 {
		return 0
	}
	return float64(s.covered) * 100 / float64(s.total)
}

func main() {
	threshold := flag.Float64("threshold", 0, "minimum coverage percentage (0 = no threshold check)")
	exclude := flag.String("exclude", "", "exclude packages whose path contains this string from the threshold (still shown in output)")
	filter := flag.String("filter", "", "only include packages whose path contains this string")
	label := flag.String("label", "coverage", "label for the aggregate line")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: coverreport [flags] <profile>")
		os.Exit(2)
	}

	module := readModule()
	stats, err := parseProfile(flag.Arg(0), module, *filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading profile: %v\n", err)
		os.Exit(2)
	}

	pkgs := sortedKeys(stats)

	fmt.Println("Package coverage:")
	for _, pkg := range pkgs {
		note := ""
		if *exclude != "" && strings.Contains(pkg, *exclude) {
			note = " (excluded from threshold)"
		}
		fmt.Printf("  %-44s %5.1f%%%s\n", pkg, stats[pkg].pct(), note)
	}
	fmt.Println()

	var totalStmts, coveredStmts int
	for _, pkg := range pkgs {
		if *exclude != "" && strings.Contains(pkg, *exclude) {
			continue
		}
		totalStmts += stats[pkg].total
		coveredStmts += stats[pkg].covered
	}

	if totalStmts == 0 {
		fmt.Fprintln(os.Stderr, "no coverage data found")
		os.Exit(1)
	}

	pct := float64(coveredStmts) * 100 / float64(totalStmts)
	fmt.Printf("%s: %.1f%%\n", *label, pct)

	if *threshold > 0 && pct < *threshold {
		fmt.Printf("FAIL: below %.0f%% threshold\n", *threshold)
		os.Exit(1)
	}
}

func parseProfile(filename, module, filter string) (map[string]*pkgStats, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	prefix := module + "/"
	stats := map[string]*pkgStats{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}

		// fields[0] = "me/internal/todos/handler.go:25.34,27.2"
		fileAndPos := strings.SplitN(fields[0], ":", 2)
		if len(fileAndPos) != 2 {
			continue
		}
		filePath := fileAndPos[0]

		if !strings.HasPrefix(filePath, prefix) {
			continue
		}
		// Strip module prefix and filename to get package path.
		// "me/internal/todos/handler.go" → "internal/todos"
		rel := filePath[len(prefix):]
		lastSlash := strings.LastIndex(rel, "/")
		if lastSlash < 0 {
			continue // root package, no directory
		}
		pkg := rel[:lastSlash]

		if filter != "" && !strings.Contains(pkg, filter) {
			continue
		}

		numStmts, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		if stats[pkg] == nil {
			stats[pkg] = &pkgStats{}
		}
		stats[pkg].total += numStmts
		if count > 0 {
			stats[pkg].covered += numStmts
		}
	}

	return stats, scanner.Err()
}

func readModule() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func sortedKeys(m map[string]*pkgStats) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
