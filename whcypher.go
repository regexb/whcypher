package whcypher

import (
	"fmt"
	"math/rand"
	"strings"
)

type Direction int

const (
	DirectionRight Direction = 1 << iota // 1
	DirectionLeft  Direction = 1 << iota // 2
	DirectionUp    Direction = 1 << iota // 4
	DirectionDown  Direction = 1 << iota // 8
	DirectionDiag  Direction = 1 << iota // 16
)

var directionNames map[Direction]string = map[Direction]string{
	DirectionRight: "right",
	DirectionLeft:  "left",
	DirectionUp:    "up",
	DirectionDown:  "down",
	DirectionDiag:  "diagonal",
}

func (d Direction) Directions() []Direction {
	dirs := []Direction{}
	if d&DirectionRight > 0 {
		dirs = append(dirs, DirectionRight)
	}
	if d&DirectionLeft > 0 {
		dirs = append(dirs, DirectionLeft)
	}
	if d&DirectionUp > 0 {
		dirs = append(dirs, DirectionUp)
	}
	if d&DirectionDown > 0 {
		dirs = append(dirs, DirectionDown)
	}
	if d&DirectionDiag > 0 {
		dirs = append(dirs, DirectionDiag)
	}
	return dirs
}

func (d Direction) String() string {
	dirs := []string{}
	for _, n := range d.Directions() {
		if direction, ok := directionNames[n]; ok {
			dirs = append(dirs, direction)
		}
	}
	return strings.Join(dirs, "|")
}

type Node struct {
	Children      [26]*Node
	LocDirections Direction              // 00001 = right, 00010 = left, 00100 = up, 01000 = down, 10000 = diag
	KnownLoc      map[Direction][][4]int // [Direction]int[[page, row, col, len], [page, row, col, len]]
}

func (n *Node) KnownLocationsForDirections(dir Direction) [][5]int {
	directions := dir.Directions()
	locs := [][5]int{}
	for _, d := range directions {
		for _, l := range n.KnownLoc[d] {
			locs = append(locs, [5]int{l[0], l[1], l[2], l[3], int(d)})
		}
	}
	return locs
}

func (n *Node) AddLoc(dir Direction, page, row, colStart, depth int) {
	n.KnownLoc[dir] = append(n.KnownLoc[dir], [4]int{page, row, colStart, depth})
}

func NewNode() *Node {
	return &Node{
		KnownLoc: make(map[Direction][][4]int),
	}
}

type Trie struct {
	RootNode  *Node
	locSelect func(int) int
}

func NewTrie() *Trie {
	return &Trie{
		RootNode: NewNode(),
		locSelect: func(n int) int {
			return 0
		},
	}
}

func (t *Trie) SetLocSelect(f func(int) int) {
	t.locSelect = f
}

func (t *Trie) WithRandomLocSelect() {
	t.locSelect = func(n int) int {
		return rand.Intn(n)
	}
}

func (t *Trie) InsertPageRow(dir Direction, page, rowNum int, letters string) error {
	for i := range letters {
		next := letters[i:]
		if err := t.InsertPagePart(dir, page, rowNum, i, next); err != nil {
			return err
		}
	}
	return nil
}

func (t *Trie) InsertPagePart(dir Direction, page, rowNum, colStart int, letters string) error {
	current := t.RootNode
	for i, l := range strings.ToLower(letters) {
		index := l - 'a' // 99 - lower ascii table decimal number
		if index < 0 || index > 25 {
			return fmt.Errorf("invalid characters in source: %q", l)
		}
		if current.Children[index] == nil {
			current.Children[index] = NewNode()
		}
		current = current.Children[index]

		// then add loc
		if current != t.RootNode {
			current.LocDirections |= dir
			current.AddLoc(dir, page, rowNum, colStart, i+1)
		}
	}
	return nil
}

// SearchLetters returns the index of the term found up until and all the known locations.
// If the whole term was found, the index will be len(term)
func (t *Trie) SearchLetters(term string, direction Direction) (int, [][5]int) {
	current := t.RootNode
	strippedTerm := strings.ToLower(term)
	for i := 0; i < len(strippedTerm); i++ {
		index := strippedTerm[i] - 'a'

		// next letter not found
		if current == nil || current.Children[index] == nil {
			return i, current.KnownLocationsForDirections(direction)
		}

		// next letter in wrong direction
		if current.Children[index].LocDirections&direction == 0 {
			return i, current.KnownLocationsForDirections(direction)
		}
		current = current.Children[index]
	}
	return len(strippedTerm), current.KnownLocationsForDirections(direction)
}

// ConstructPhraseLTR uses a left to right search to find the longest runs of
// letters it can until the phrase is complete.
func (t *Trie) ConstructPhraseLTR(phrase string, dir Direction) ([][5]int, error) {
	if len(phrase) == 0 {
		return nil, fmt.Errorf("invalid phrase %q", phrase)
	}

	phraseLocations := [][5]int{}

	// Strip down phrase for searching.
	strippedPhrase := strings.ToLower(strings.ReplaceAll(phrase, " ", ""))

	// Use search until the phrase is complete.
	remaining := strippedPhrase[0:]
	for len(remaining) > 0 {
		index, locations := t.SearchLetters(remaining, dir)

		if index < 1 || len(locations) < 1 {
			return nil, fmt.Errorf("letter %s not found", string(remaining[index]))
		}
		ri := min(t.locSelect(len(locations)), len(locations)-1)
		phraseLocations = append(phraseLocations, locations[ri])
		remaining = remaining[index:]
	}

	return phraseLocations, nil
}

func (t *Trie) ConstructPhraseLongest(phrase string, dir Direction) ([][5]int, error) {
	strippedPhrase := strings.ToLower(strings.ReplaceAll(phrase, " ", ""))
	return t.findAllLongest(strippedPhrase, dir)
}

func (t *Trie) findAllLongest(phrase string, dir Direction) (res [][5]int, err error) {
	if len(phrase) == 0 {
		return nil, fmt.Errorf("invalid phrase %q", phrase)
	}

	li, ls, lloc := t.FindLongest(phrase, dir)
	if len(lloc) == 0 {
		return nil, fmt.Errorf("unable to complete phrase %q", phrase)
	}

	ri := min(t.locSelect(len(lloc)), len(lloc)-1)
	if len(phrase) == ls {
		return append(res, lloc[ri]), nil
	}

	pre := phrase[0:li]
	post := phrase[li+ls:]

	// Prefix remaining
	if len(pre) > 0 {
		prer, prerr := t.findAllLongest(pre, dir)
		if prerr != nil {
			return nil, prerr
		}
		res = append(res, prer...)
	}

	// Main
	res = append(res, lloc[ri])

	// Postfix remaining
	if len(post) > 0 {
		posr, poerr := t.findAllLongest(post, dir)
		if poerr != nil {
			return nil, poerr
		}
		res = append(res, posr...)
	}

	return
}

func (t *Trie) FindLongest(phrase string, dir Direction) (longestIndex int, longestSize int, longestLoc [][5]int) {
	for i := 0; i < len(phrase); i++ {
		check := phrase[i:]
		s, loc := t.SearchLetters(check, dir)
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
