//go:build js && wasm

package main

import (
	"bytes"
	"crypto"
	_ "crypto/sha512"
	_ "embed"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"syscall/js"

	"github.com/regexb/whcypher"
)

//go:embed source.txt
var sourceData []byte

var (
	nonAlphaRegex = regexp.MustCompile(`[^a-zA-Z]`)
)

func loadSource() [][][]byte {
	var out [][][]byte
	groups := bytes.Split(sourceData, []byte("\n\n"))
	for _, g := range groups {
		out = append(out, bytes.Split(g, []byte("\n")))
	}
	return out
}

// rowFromSource returns the string of characters walking in the x and y direction
func rowFromSource(source [][]byte, startX, startY, moveX, moveY int) []byte {
	row := make([]byte, 0)
	x, y := startX, startY

	for x >= 0 && x < len(source) && y >= 0 && y < len(source[x]) {
		row = append(row, source[x][y])
		x += moveX
		y += moveY
	}

	return row
}

func cypherTreeFromSource(source [][][]byte) *whcypher.Trie {
	trie := whcypher.NewTrie()
	for pi, page := range source {
		for ri, row := range page {
			for bi := range row {
				if err := trie.InsertPagePart(whcypher.DirectionRight, pi, ri, bi, string(rowFromSource(page, ri, bi, 0, 1))); err != nil {
					panic(err)
				}
				if err := trie.InsertPagePart(whcypher.DirectionLeft, pi, ri, bi, string(rowFromSource(page, ri, bi, 0, -1))); err != nil {
					panic(err)
				}
				if err := trie.InsertPagePart(whcypher.DirectionUp, pi, ri, bi, string(rowFromSource(page, ri, bi, -1, 0))); err != nil {
					panic(err)
				}
				if err := trie.InsertPagePart(whcypher.DirectionDown, pi, ri, bi, string(rowFromSource(page, ri, bi, 1, 0))); err != nil {
					panic(err)
				}
				if err := trie.InsertPagePart(whcypher.DirectionRightDown, pi, ri, bi, string(rowFromSource(page, ri, bi, 1, 1))); err != nil {
					panic(err)
				}
				if err := trie.InsertPagePart(whcypher.DirectionLeftDown, pi, ri, bi, string(rowFromSource(page, ri, bi, 1, -1))); err != nil {
					panic(err)
				}
				if err := trie.InsertPagePart(whcypher.DirectionRightUp, pi, ri, bi, string(rowFromSource(page, ri, bi, -1, 1))); err != nil {
					panic(err)
				}
				if err := trie.InsertPagePart(whcypher.DirectionLeftUp, pi, ri, bi, string(rowFromSource(page, ri, bi, -1, -1))); err != nil {
					panic(err)
				}
			}
		}
	}
	return trie
}

type cypherTree struct {
	source [][][]byte
	trie   *whcypher.Trie
}

func (c *cypherTree) hash(this js.Value, args []js.Value) any {
	h := crypto.SHA512.New()
	h.Write([]byte(args[0].String()))

	return hex.EncodeToString(h.Sum(nil))
}

func (c *cypherTree) generate(this js.Value, args []js.Value) any {
	if len(args) != 3 {
		panic("bad args")
	}

	// Remove non-alpha characters
	in := nonAlphaRegex.ReplaceAllString(args[0].String(), "")

	algo := args[2].String()
	direction := whcypher.Direction(args[1].Int())

	fmt.Printf("Query: %q w/ algo (%s) and options (%s)\n", in, algo, direction)

	var rawCode [][5]int
	if algo == "longest" {
		var err error
		rawCode, err = c.trie.ConstructPhraseLongest(in, direction)
		if err != nil {
			return "error"
		}
	} else {
		var err error
		rawCode, err = c.trie.ConstructPhraseLTR(in, direction)
		if err != nil {
			return "error"
		}
	}

	if len(rawCode) == 0 {
		return "not found"
	}

	out := &strings.Builder{}
	for i, part := range rawCode {
		fmt.Fprintf(out, "[%d %d %d %d (%s)]", part[0]+3, part[1]+1, part[2]+1, part[3], dirMap[whcypher.Direction(part[4])])
		if i == len(rawCode)-1 {
			fmt.Fprintln(out)
		} else {
			fmt.Fprint(out, " ")
		}
	}

	return out.String()
}

var dirMap = map[whcypher.Direction]string{
	whcypher.DirectionRight:     "➡️",
	whcypher.DirectionLeft:      "⬅️",
	whcypher.DirectionUp:        "⬆️",
	whcypher.DirectionDown:      "⬇️",
	whcypher.DirectionRightDown: "↘️",
	whcypher.DirectionRightUp:   "↗️",
	whcypher.DirectionLeftDown:  "↙️",
	whcypher.DirectionLeftUp:    "↖️",
}

func main() {

	// Read the file
	source := loadSource()
	fmt.Printf("loaded %d pages\n", len(source))

	cypherGenerator := &cypherTree{
		trie:   cypherTreeFromSource(source),
		source: source,
	}

	js.Global().Set("generateCypher", js.FuncOf(cypherGenerator.generate))

	select {}
}
