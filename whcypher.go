package whcypher

import (
	"fmt"
	"math/rand"
	"strings"
)

type Node struct {
	Children [26]*Node
	KnownLoc [][3]int // int[[page, row, col], [page, row, col]]
}

func (n *Node) AddLoc(page, row, col int) {
	n.KnownLoc = append(n.KnownLoc, [3]int{page, row, col})
}

func NewNode(char string) *Node {
	return &Node{}
}

type Trie struct {
	RootNode *Node
}

func (t *Trie) InsertPageRow(page, rowNum int, letters []string) error {
	for i := range letters {
		next := letters[i:]
		if err := t.InsertPagePart(page, rowNum, i, next); err != nil {
			return err
		}
	}
	return nil
}

func (t *Trie) InsertPagePart(page, rowNum, colStart int, letters []string) error {
	current := t.RootNode
	for i, l := range letters {
		l := strings.ToLower(string(l[0])) // ensure only 1 character
		index := l[0] - 'a'                // 99 - lower ascii table decimal number
		if index < 0 || index > 25 {
			return fmt.Errorf("invalid characters in source: %q", l)
		}
		if current.Children[index] == nil {
			current.Children[index] = NewNode(string(l))
		}
		current = current.Children[index]

		// then add loc
		if current != t.RootNode {
			current.AddLoc(page, rowNum, colStart+i)
		}
	}
	return nil
}

// SearchLetters returns the index of the term found up until and all the known locations.
// If the whole term was found, the index will be len(term)
func (t *Trie) SearchLetters(term string) (int, [][3]int) {
	current := t.RootNode
	strippedTerm := strings.ToLower(term)
	for i := 0; i < len(strippedTerm); i++ {
		index := strippedTerm[i] - 'a'

		// next letter not found
		if current == nil || current.Children[index] == nil {
			return i, current.KnownLoc
		}
		current = current.Children[index]
	}
	return len(strippedTerm), current.KnownLoc
}

// ConstructPhraseLTR uses a left to right search to find the longest runs of
// letters it can until the phrase is complete.
func (t *Trie) ConstructPhraseLTR(phrase string) ([][4]int, error) {

	phraseLocations := [][4]int{}

	// Strip down phrase for searching.
	strippedPhrase := strings.ToLower(strings.ReplaceAll(phrase, " ", ""))

	// Use search until the phrase is complete.
	remaining := strippedPhrase[0:]
	for len(remaining) > 0 {
		index, locations := t.SearchLetters(remaining)

		if index < 1 || len(locations) < 1 {
			return nil, fmt.Errorf("Letter %s not found", string(remaining[index]))
		}
		ri := rand.Intn(len(locations))
		loc := [4]int{locations[ri][0], locations[ri][1], locations[ri][2] - index + 1, index}
		phraseLocations = append(phraseLocations, loc)
		remaining = remaining[index:]
	}

	return phraseLocations, nil
}

func (t *Trie) ConstructPhraseLongest(phrase string) ([][4]int, error) {
	strippedPhrase := strings.ToLower(strings.ReplaceAll(phrase, " ", ""))
	return t.findAllLongest(strippedPhrase)
}

func (t *Trie) findAllLongest(phrase string) (res [][4]int, err error) {
	if len(phrase) == 0 {
		return nil, fmt.Errorf("invalid phrase %q", phrase)
	}

	li, ls, lloc := t.FindLongest(phrase)
	if len(lloc) == 0 {
		return nil, fmt.Errorf("unable to complete phrase %q", phrase)
	}

	ri := rand.Intn(len(lloc))
	lpart := [4]int{lloc[ri][0], lloc[ri][1], lloc[ri][2] - ls + 1, ls}

	if len(phrase) == ls {
		return append(res, lpart), nil
	}

	pre := phrase[0:li]
	post := phrase[li+ls:]

	// Prefix remaining
	if len(pre) > 0 {
		prer, prerr := t.findAllLongest(pre)
		if prerr != nil {
			return nil, prerr
		}
		res = append(res, prer...)
	}

	// Main
	res = append(res, lpart)

	// Postfix remaining
	if len(post) > 0 {
		posr, poerr := t.findAllLongest(post)
		if poerr != nil {
			return nil, poerr
		}
		res = append(res, posr...)
	}

	return
}

func (t *Trie) FindLongest(phrase string) (longestIndex int, longestSize int, longestLoc [][3]int) {
	for i := 0; i < len(phrase); i++ {
		check := phrase[i:]
		s, loc := t.SearchLetters(check)
		if s > longestSize {
			longestIndex = i
			longestLoc = loc
			longestSize = s
		}
		if s > len(phrase)/2 {
			return
		}
	}
	return
}
