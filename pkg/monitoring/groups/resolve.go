/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package groups

import (
	"fmt"
	"sort"
	"strings"
)

const (
	tokenAll  = "all"
	tokenNone = "none"
)

// Resolve turns one --*-allowlist value into the set of item names to expose.
//
// The index maps every item of one artifact type to its group, so the same
// grammar serves metrics, recording rules, and alerts. An entry is either a
// group name or an item name:
//
//	""                     the groups with Default: true
//	"all" / "none"         every item / no item
//	"a,b"                  exactly the items of a and b
//	"+a,-b"                the default set, plus a, minus b
//
// Absolute and relative forms cannot be mixed, and an entry that names neither
// a known group nor a known item is an error: a silently ignored typo would
// mean running with less than was asked for.
//
// The second return value lists entries that resolved to nothing. The three
// flags share one group vocabulary, so naming a group with no items of this
// artifact type — inventory has no alerts — is legal rather than an error,
// but it is worth telling the operator about: on its own it produces an empty
// selection that looks like the typo this function otherwise rejects.
func Resolve(index map[string]Name, raw string) (map[string]bool, []string, error) {
	raw = strings.TrimSpace(raw)

	switch raw {
	case "":
		return defaultSet(index), nil, nil
	case tokenAll:
		return allOf(index), nil, nil
	case tokenNone:
		return map[string]bool{}, nil, nil
	}

	entries, err := split(raw)
	if err != nil {
		return nil, nil, err
	}

	relative, err := classify(entries)
	if err != nil {
		return nil, nil, err
	}

	selected := map[string]bool{}
	if relative {
		selected = defaultSet(index)
	}

	var empty []string
	for _, e := range entries {
		// split rejected repeated signs, so this strips exactly one.
		names, err := expand(index, strings.TrimLeft(e, "+-"))
		if err != nil {
			return nil, nil, err
		}
		if len(names) == 0 {
			empty = append(empty, e)
		}
		for _, n := range names {
			if strings.HasPrefix(e, "-") {
				delete(selected, n)
			} else {
				selected[n] = true
			}
		}
	}

	return selected, empty, nil
}

func split(raw string) ([]string, error) {
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		e := strings.TrimSpace(part)
		if e == "" {
			continue
		}
		name := strings.TrimLeft(e, "+-")
		switch signs := len(e) - len(name); {
		case name == "":
			return nil, fmt.Errorf("entry %q has no name after its sign", e)
		case signs > 1:
			return nil, fmt.Errorf("entry %q has more than one leading sign", e)
		case name == tokenAll || name == tokenNone:
			if signs == 1 {
				return nil, fmt.Errorf(
					"%q cannot take a %c prefix: %q and %q are absolute and select everything or nothing",
					e, e[0], tokenAll, tokenNone)
			}
			return nil, fmt.Errorf("%q cannot be combined with other entries", e)
		}

		out = append(out, e)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no entries")
	}
	return out, nil
}

// classify reports whether the entries adjust the default set, and rejects a
// mix of the two forms: "+a,b" reads as though b were also added, when under
// the absolute form it would silently discard every default.
func classify(entries []string) (bool, error) {
	var relative, absolute []string
	for _, e := range entries {
		if strings.HasPrefix(e, "+") || strings.HasPrefix(e, "-") {
			relative = append(relative, e)
			continue
		}
		absolute = append(absolute, e)
	}

	if len(relative) > 0 && len(absolute) > 0 {
		return false, fmt.Errorf(
			"cannot mix %q with %q: prefix every entry with + or -, or none of them",
			relative[0], absolute[0],
		)
	}
	return len(relative) > 0, nil
}

func expand(index map[string]Name, name string) ([]string, error) {
	if _, ok := Lookup(Name(name)); ok {
		var out []string
		for item, g := range index {
			if g == Name(name) {
				out = append(out, item)
			}
		}
		sort.Strings(out)
		return out, nil
	}

	if _, ok := index[name]; ok {
		return []string{name}, nil
	}

	return nil, fmt.Errorf("%q is neither a known group nor a known name here%s", name, hint(index, name))
}

// hint points at the likeliest intent when a name is rejected. A name that is
// valid for another artifact type is the common mistake, and saying so is more
// use than listing every accepted value.
func hint(index map[string]Name, name string) string {
	if _, ok := Lookup(Name(name)); ok {
		return ""
	}
	for item := range index {
		if strings.EqualFold(item, name) {
			return fmt.Sprintf(" (did you mean %q?)", item)
		}
	}
	return ""
}

func defaultSet(index map[string]Name) map[string]bool {
	out := make(map[string]bool)
	for item, g := range index {
		if group, ok := Lookup(g); ok && group.Default {
			out[item] = true
		}
	}
	return out
}

func allOf(index map[string]Name) map[string]bool {
	out := make(map[string]bool, len(index))
	for item := range index {
		out[item] = true
	}
	return out
}
