package whcypher

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNode_AddLoc(t *testing.T) {
	node := NewNode()
	node.AddLoc(DirectionRight, 1, 2, 3, 4)
	if len(node.KnownLoc) != 1 {
		t.Errorf("Expected 1, got %d", len(node.KnownLoc))
	}
}

func TestNewNode(t *testing.T) {
	node := NewNode()
	if node == nil {
		t.Error("Expected new node, got nil")
	}
}

func TestNewTrie(t *testing.T) {
	trie := NewTrie()
	if trie == nil {
		t.Error("Expected new trie, got nil")
	}
}

func TestTrie_InsertPageRow(t *testing.T) {
	trie := NewTrie()
	err := trie.InsertPageRow(DirectionRight, 0, 0, "abc")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestTrie_InsertPageRow_WithNumbers(t *testing.T) {
	trie := NewTrie()
	err := trie.InsertPageRow(DirectionRight, 0, 0, "abc123")
	if err == nil {
		t.Errorf("Expected invalid character error, got %v", err)
	}
}

func TestTrie_InsertPagePart(t *testing.T) {
	trie := NewTrie()
	err := trie.InsertPagePart(DirectionRight, 0, 0, 0, "abc")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestTrie_SearchLetters(t *testing.T) {
	// Define test cases
	testCases := []struct {
		description        string
		pageRows           []string
		searchLetters      string
		expectedFoundLen   int
		expectedNumMatches int
		expectedLocations  [][5]int
	}{
		{
			description:        "Test case 1",
			pageRows:           []string{"abc"},
			searchLetters:      "abc",
			expectedFoundLen:   3,
			expectedNumMatches: 1,
			expectedLocations:  [][5]int{{0, 0, 0, 3, 1}},
		},
		{
			description:        "Empty search",
			pageRows:           []string{"abc"},
			searchLetters:      "",
			expectedFoundLen:   0,
			expectedNumMatches: 0,
			expectedLocations:  nil,
		},
		{
			description:        "No match",
			pageRows:           []string{"abc"},
			searchLetters:      "def",
			expectedFoundLen:   0,
			expectedNumMatches: 0,
			expectedLocations:  nil,
		},
		{
			description:        "Multi rows",
			pageRows:           []string{"abc", "def", "ghi"},
			searchLetters:      "hi",
			expectedFoundLen:   2,
			expectedNumMatches: 1,
			expectedLocations:  [][5]int{{0, 2, 1, 2, 1}},
		},
		{
			description:        "Partial match",
			pageRows:           []string{"aaaaa", "aaaaa", "aaaaa", "helaa", "aaaaa"},
			searchLetters:      "hello",
			expectedFoundLen:   3,
			expectedNumMatches: 1,
			expectedLocations:  [][5]int{{0, 3, 0, 3, 1}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			trie := NewTrie()
			for i, row := range tc.pageRows {
				trie.InsertPageRow(DirectionRight, 0, i, row)
			}
			index, knownLocations := trie.SearchLetters(tc.searchLetters, DirectionRight)
			if index != tc.expectedFoundLen {
				t.Errorf("Expected length found %d, got %d", tc.expectedFoundLen, index)
			}
			if len(knownLocations) != tc.expectedNumMatches {
				t.Errorf("Expected number of matches %d, got %d", tc.expectedNumMatches, len(knownLocations))
			}
			if diff := cmp.Diff(knownLocations, tc.expectedLocations, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Expected knownLocations to match, got diff %s", diff)
			}
		})
	}
}

func TestTrie_ConstructPhraseLTR(t *testing.T) {
	// Define test cases
	testCases := []struct {
		description string
		pageRows    []string
		searchWord  string
		expected    [][5]int
		expectedErr error
	}{
		{
			description: "Test case 1",
			pageRows:    []string{"abc"},
			searchWord:  "abc",
			expected:    [][5]int{{0, 0, 0, 3, 1}},
			expectedErr: nil,
		},
		{
			description: "Empty search",
			pageRows:    []string{"abc"},
			searchWord:  "",
			expected:    nil,
			expectedErr: errors.ErrUnsupported,
		},
		{
			description: "Missing char",
			pageRows:    []string{"abc"},
			searchWord:  "d",
			expected:    nil,
			expectedErr: errors.ErrUnsupported,
		},
		{
			description: "Multi part match",
			pageRows:    []string{"abcdefg"},
			searchWord:  "bab",
			expected:    [][5]int{{0, 0, 1, 1, 1}, {0, 0, 0, 2, 1}},
			expectedErr: nil,
		},
		{
			description: "Multi row match",
			pageRows:    []string{"fghooo", "ooabco", "oodeoo"},
			searchWord:  "abcdefgh",
			expected:    [][5]int{{0, 1, 2, 3, 1}, {0, 2, 2, 2, 1}, {0, 0, 0, 3, 1}},
			expectedErr: nil,
		},
		{
			description: "Longest at front",
			pageRows:    []string{"abcdea", "aaabca", "aabcda"},
			searchWord:  "abcde",
			expected:    [][5]int{{0, 0, 0, 5, 1}},
			expectedErr: nil,
		},
		{
			description: "Longest in middle",
			pageRows:    []string{"bcdefa", "ooooog", "obcode"},
			searchWord:  "abcdefg",
			expected:    [][5]int{{0, 0, 5, 1, 1}, {0, 0, 0, 5, 1}, {0, 1, 5, 1, 1}},
			expectedErr: nil,
		},
		{
			description: "Longest in middle with shorter ltr match",
			pageRows:    []string{"bcdefa", "abooog", "obcode"},
			searchWord:  "abcdefg",
			expected:    [][5]int{{0, 1, 0, 2, 1}, {0, 0, 1, 4, 1}, {0, 1, 5, 1, 1}},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			trie := NewTrie()
			for i, row := range tc.pageRows {
				trie.InsertPageRow(DirectionRight, 0, i, row)
			}
			result, err := trie.ConstructPhraseLTR(tc.searchWord, DirectionRight)
			if err == nil && tc.expectedErr != nil {
				t.Errorf("Expected error %v, got %v", tc.expectedErr, err)
			}
			if diff := cmp.Diff(result, tc.expected, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Expected result to match, got diff (-got,+want) %s", diff)
			}
		})
	}
}

func TestTrie_ConstructPhraseLongest(t *testing.T) {
	// Define test cases
	testCases := []struct {
		description string
		pageRows    []string
		searchWord  string
		expected    [][5]int
		expectedErr error
	}{
		{
			description: "Test case 1",
			pageRows:    []string{"abc"},
			searchWord:  "abc",
			expected:    [][5]int{{0, 0, 0, 3, 1}},
			expectedErr: nil,
		},
		{
			description: "Empty search",
			pageRows:    []string{"abc"},
			searchWord:  "",
			expected:    nil,
			expectedErr: errors.ErrUnsupported,
		},
		{
			description: "Missing char",
			pageRows:    []string{"abc"},
			searchWord:  "d",
			expected:    nil,
			expectedErr: errors.ErrUnsupported,
		},
		{
			description: "Multi part match",
			pageRows:    []string{"abcdefg"},
			searchWord:  "bab",
			expected:    [][5]int{{0, 0, 1, 1, 1}, {0, 0, 0, 2, 1}},
			expectedErr: nil,
		},
		{
			description: "Multi row match",
			pageRows:    []string{"fghooo", "ooabco", "oodeoo"},
			searchWord:  "abcdefgh",
			expected:    [][5]int{{0, 1, 2, 3, 1}, {0, 2, 2, 2, 1}, {0, 0, 0, 3, 1}},
			expectedErr: nil,
		},
		{
			description: "Longest at front",
			pageRows:    []string{"abcdea", "aaabca", "aabcda"},
			searchWord:  "abcde",
			expected:    [][5]int{{0, 0, 0, 5, 1}},
			expectedErr: nil,
		},
		{
			description: "Longest in middle",
			pageRows:    []string{"bcdefa", "ooooog", "obcode"},
			searchWord:  "abcdefg",
			expected:    [][5]int{{0, 0, 5, 1, 1}, {0, 0, 0, 5, 1}, {0, 1, 5, 1, 1}},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			trie := NewTrie()
			for i, row := range tc.pageRows {
				trie.InsertPageRow(DirectionRight, 0, i, row)
			}
			result, err := trie.ConstructPhraseLongest(tc.searchWord, DirectionRight)
			if err == nil && tc.expectedErr != nil {
				t.Errorf("Expected error %v, got %v", tc.expectedErr, err)
			}
			if diff := cmp.Diff(result, tc.expected); diff != "" {
				t.Errorf("Expected result to match, got diff (-got,+want) %s", diff)
			}
		})
	}
}
