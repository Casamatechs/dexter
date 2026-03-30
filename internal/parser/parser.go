package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	defmoduleRe = regexp.MustCompile(`^\s*defmodule\s+([A-Za-z0-9_.]+)\s+do`)
	defRe       = regexp.MustCompile(`^\s*(defp?|defmacrop?)\s+([a-z_][a-z0-9_?!]*)\s*[\(|,|do|\s]`)
)

type Definition struct {
	Module   string
	Function string // empty for module definitions
	Line     int
	FilePath string
	Kind     string // "module", "def", "defp", "defmacro", "defmacrop"
}

func ParseFile(path string) ([]Definition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var defs []Definition
	var moduleStack []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	inHeredoc := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track heredoc boundaries (""" toggles in/out)
		quoteCount := strings.Count(line, `"""`)
		if quoteCount > 0 {
			// Heredoc open and close on same line (e.g., @moduledoc """...""") — stay as-is
			if quoteCount >= 2 {
				continue
			}
			inHeredoc = !inHeredoc
			if inHeredoc {
				continue
			}
			// Just closed heredoc — this line is the closing """, skip it
			continue
		}

		if inHeredoc {
			continue
		}

		// Track module nesting via end keywords
		if trimmed == "end" && len(moduleStack) > 1 {
			moduleStack = moduleStack[:len(moduleStack)-1]
		}

		currentModule := ""
		if len(moduleStack) > 0 {
			currentModule = moduleStack[len(moduleStack)-1]
		}

		if m := defmoduleRe.FindStringSubmatch(line); m != nil {
			currentModule = m[1]
			moduleStack = append(moduleStack, currentModule)
			defs = append(defs, Definition{
				Module:   currentModule,
				Line:     lineNum,
				FilePath: path,
				Kind:     "module",
			})
			continue
		}

		if currentModule != "" {
			if m := defRe.FindStringSubmatch(line); m != nil {
				kind := m[1]
				funcName := m[2]
				defs = append(defs, Definition{
					Module:   currentModule,
					Function: funcName,
					Line:     lineNum,
					FilePath: path,
					Kind:     kind,
				})
			}
		}
	}

	return defs, scanner.Err()
}

func IsElixirFile(path string) bool {
	extension := filepath.Ext(path)
	return extension == ".ex" || extension == ".exs"
}
