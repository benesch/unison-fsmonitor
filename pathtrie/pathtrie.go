// Package pathtrie provides a specialized data structure for storing filesystem
// paths.
package pathtrie

import (
	"path/filepath"
	"strings"
)

// A PathTrie records filesystem paths that have changed. It stores the minimal
// set of parent paths that must be visited recursively. For example, if a,
// a/b/c, and b/c are all inserted into the trie, only a and b/c will be stored.
type PathTrie struct {
	present  bool
	children map[string]*PathTrie
}

// Insert records that a new path has changed.
//
// Insertion maintains the invariant that only the minimal set of parent paths
// is stored. Inserting a path that is a parent of existing paths in the trie
// will cause the removal of those existing paths. Inserting a path that is
// a child of an existing path in the trie is a no-op.
func (t *PathTrie) Insert(path string) {
	var segments []string
	if len(path) > 0 {
		segments = strings.Split(path, string(filepath.Separator))
	}
	t.insert(segments)
}

func (t *PathTrie) insert(segments []string) {
	if t.present {
		return
	}
	if len(segments) == 0 {
		t.present = true
		t.children = nil
	} else {
		ct, ok := t.children[segments[0]]
		if !ok {
			if t.children == nil {
				t.children = map[string]*PathTrie{}
			}
			ct = &PathTrie{}
			t.children[segments[0]] = ct
		}
		ct.insert(segments[1:])
	}
}

// Walk visits all the paths stored in the trie in some arbitrary order.
func (t *PathTrie) Walk(fn func(string)) {
	t.walk(fn, nil)
}

func (t *PathTrie) walk(fn func(string), segments []string) {
	if t.present {
		fn(filepath.Join(segments...))
	} else {
		for s, c := range t.children {
			c.walk(fn, append(segments, s))
		}
	}
}

// Clear removes all paths from the trie.
func (t *PathTrie) Clear() {
	t.present = false
	t.children = nil
}

// Empty reports whether any paths are stored in the trie.
func (t *PathTrie) Empty() bool {
	return !t.present && len(t.children) == 0
}
