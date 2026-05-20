package main

import "strings"

var knownValueFlags = map[string]bool{
	"target":         true,
	"var":            true,
	"var-file":       true,
	"replace":        true,
	"out":            true,
	"parallelism":    true,
	"lock":           true,
	"lock-timeout":   true,
	"chdir":          true,
	"workspace":      true,
	"input":          true,
	"backend-config": true,
	"plugin-dir":     true,
	"get":            true,
}

var knownBoolFlags = map[string]bool{
	"json":             true,
	"destroy":          true,
	"refresh-only":     true,
	"compact-warnings": true,
	"upgrade":          true,
	"reconfigure":      true,
	"force-copy":       true,
	"backend":          true,
}

func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	out := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			out = append(out, args[i:]...)
			break
		}

		if arg == "-" || !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			out = append(out, arg)
			continue
		}

		name, hasEquals := extractFlagName(arg[1:])

		if knownBoolFlags[name] {
			out = append(out, "-"+arg)
			continue
		}

		if knownValueFlags[name] {
			out = append(out, "-"+arg)
			if !hasEquals && i+1 < len(args) {
				i++
				out = append(out, args[i])
			}
			continue
		}

		if len(name) > 1 {
			out = append(out, "-"+arg)
			continue
		}

		out = append(out, arg)
	}

	return out
}

func extractFlagName(s string) (name string, hasEquals bool) {
	idx := strings.IndexByte(s, '=')
	if idx >= 0 {
		return s[:idx], true
	}
	return s, false
}

func splitPassthrough(args []string) (before, after []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}
