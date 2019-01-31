package pathtrie

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathTrie(t *testing.T) {
	testCases := []struct {
		in  []string
		out []string
	}{
		{
			[]string{},
			[]string{},
		},
		{
			[]string{"a", "b", "c"},
			[]string{"a", "b", "c"},
		},
		{
			[]string{"a/b/c", "a/b/d", "a/b/e"},
			[]string{"a/b/c", "a/b/d", "a/b/e"},
		},
		{
			[]string{"a", "a/a", "a/a/a"},
			[]string{"a"},
		},
		{
			[]string{"a/a/a", "a/a", "a"},
			[]string{"a"},
		},
		{
			[]string{"a/a", "b", "a"},
			[]string{"a", "b"},
		},
		{
			[]string{""},
			[]string{""},
		},
		{
			[]string{"", "a"},
			[]string{""},
		},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			var trie PathTrie
			for _, p := range tc.in {
				trie.Insert(p)
			}
			var paths []string
			trie.Walk(func(p string) {
				paths = append(paths, p)
			})
			require.ElementsMatch(t, tc.out, paths)
		})
	}
}
