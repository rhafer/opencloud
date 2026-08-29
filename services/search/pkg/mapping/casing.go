package mapping

import "strings"

// addSearchSiblings writes the _lowercase and _words siblings the overrides ask
// for next to their base values.
func addSearchSiblings(m map[string]any, overrides map[string]FieldOpts) {
	for key, opts := range overrides {
		if !isCasedType(opts) || (!opts.caseInsensitive() && !opts.wordBroken()) {
			continue
		}
		parent, leaf, ok := resolveLeaf(m, key)
		if !ok {
			continue
		}
		if opts.caseInsensitive() {
			addLowercaseSibling(parent, leaf)
		}
		if opts.wordBroken() {
			addWordsSibling(parent, leaf)
		}
	}
}

// addWordsSibling copies the value to a <leaf>_words sibling; the words
// analyzer does the splitting and lowercasing. No-op for non-strings.
func addWordsSibling(parent map[string]any, leaf string) {
	switch v := parent[leaf].(type) {
	case string, []any, []string:
		parent[leaf+WordsSuffix] = v
	}
}

func isCasedType(opts FieldOpts) bool {
	return opts.Type == "" || opts.Type == TypeKeyword || opts.Type == TypePath
}

func resolveLeaf(m map[string]any, dottedPath string) (map[string]any, string, bool) {
	parts := strings.Split(dottedPath, ".")
	parent := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := parent[p].(map[string]any)
		if !ok {
			return nil, "", false
		}
		parent = next
	}
	return parent, parts[len(parts)-1], true
}

// addLowercaseSibling writes a <leaf>_lowercase sibling; no-op for non-strings.
func addLowercaseSibling(parent map[string]any, leaf string) {
	switch v := parent[leaf].(type) {
	case string:
		parent[leaf+LowercaseSuffix] = strings.ToLower(v)
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, strings.ToLower(s))
			}
		}
		parent[leaf+LowercaseSuffix] = out
	case []string:
		out := make([]string, len(v))
		for i, s := range v {
			out[i] = strings.ToLower(s)
		}
		parent[leaf+LowercaseSuffix] = out
	}
}
