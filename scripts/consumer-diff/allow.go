package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// allowEntry explains an expected new finding. Used counts the new findings
// it explained; an entry that explains nothing is reported as unused.
type allowEntry struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Gate     string `json:"gate"`
	Severity string `json:"severity"`
	Pattern  string `json:"pattern"`
	Reason   string `json:"reason"`
	Used     int    `json:"used"`
	re       *regexp.Regexp
}

func (e *allowEntry) String() string {
	return fmt.Sprintf("%s:%d", e.File, e.Line)
}

// matches reports whether e explains f: same gate (full id, short id before
// the dash, or *), same severity, and the pattern found in the normalized
// message.
func (e *allowEntry) matches(f finding) bool {
	if e.Severity != f.Severity {
		return false
	}
	short, _, _ := strings.Cut(f.Gate, "-")
	if e.Gate != "*" && !strings.EqualFold(e.Gate, f.Gate) && !strings.EqualFold(e.Gate, short) {
		return false
	}
	return e.re.MatchString(f.Message)
}

var allowSeverities = map[string]string{
	"error": sevError, "drift": sevDrift, "warn": sevWarn, "warning": sevWarn,
}

func parseAllowFile(path string) ([]*allowEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("allow file: %w", err)
	}
	defer func() { _ = f.Close() }()
	var entries []*allowEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		n++
		e, err := parseAllowLine(sc.Text())
		if err != nil {
			return nil, fmt.Errorf("allow file %s:%d: %w", path, n, err)
		}
		if e != nil {
			e.File, e.Line = path, n
			entries = append(entries, e)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("allow file %s: %w", path, err)
	}
	return entries, nil
}

// parseAllowLine reads `<gate> <severity> /<regexp>/ <reason>`; nil for a
// blank or comment line.
func parseAllowLine(line string) (*allowEntry, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}
	gate, rest := cutField(line)
	sevWord, rest := cutField(rest)
	if rest == "" {
		return nil, fmt.Errorf("want `<gate> <severity> /<regexp>/ <reason>`, got %q", line)
	}
	sev, ok := allowSeverities[strings.ToLower(sevWord)]
	if !ok {
		return nil, fmt.Errorf("severity %q is not error, drift or warn (notes never gate, so they need no entry)", sevWord)
	}
	if !strings.HasPrefix(rest, "/") {
		return nil, fmt.Errorf("the pattern must be written /<regexp>/, got %q", rest)
	}
	end := closingSlash(rest)
	if end < 0 {
		return nil, fmt.Errorf("the pattern %q has no closing slash", rest)
	}
	pattern := rest[1:end]
	if pattern == "" {
		return nil, fmt.Errorf("the pattern is empty; an entry must name what it explains")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("pattern /%s/: %w", pattern, err)
	}
	reason := strings.TrimSpace(rest[end+1:])
	if reason == "" {
		return nil, fmt.Errorf("the reason is mandatory: say why the candidate is right to report this")
	}
	return &allowEntry{Gate: gate, Severity: sev, Pattern: pattern, Reason: reason, re: re}, nil
}

// cutField splits off the first whitespace-delimited field.
func cutField(s string) (field, rest string) {
	s = strings.TrimSpace(s)
	i := strings.IndexAny(s, " \t")
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimSpace(s[i:])
}

// closingSlash finds the slash ending a /pattern/ that starts at index 0; a
// slash preceded by an odd number of backslashes is part of the pattern.
func closingSlash(s string) int {
	for i := 1; i < len(s); i++ {
		if s[i] != '/' {
			continue
		}
		bs := 0
		for j := i - 1; j > 0 && s[j] == '\\'; j-- {
			bs++
		}
		if bs%2 == 0 {
			return i
		}
	}
	return -1
}
