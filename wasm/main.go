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

func cypherTreeFromSource(source [][][]byte, opts int) *whcypher.Trie {
	trie := whcypher.NewTrie()
	for pi, page := range source {
		for ri, row := range page {
			for bi := range row {
				if opts&FlagRight != 0 {
					if err := trie.InsertPagePart(pi, ri, bi, string(rowFromSource(page, ri, bi, 0, 1))); err != nil {
						panic(err)
					}
				}

				if opts&FlagLeft != 0 {
					if err := trie.InsertPagePart(pi, ri, bi, string(rowFromSource(page, ri, bi, 0, -1))); err != nil {
						panic(err)
					}
				}

				if opts&FlagUp != 0 {
					if err := trie.InsertPagePart(pi, ri, bi, string(rowFromSource(page, ri, bi, -1, 0))); err != nil {
						panic(err)
					}
				}

				if opts&FlagDown != 0 {
					if err := trie.InsertPagePart(pi, ri, bi, string(rowFromSource(page, ri, bi, 1, 0))); err != nil {
						panic(err)
					}
				}

				if opts&FlagDiagonal != 0 {
					if err := trie.InsertPagePart(pi, ri, bi, string(rowFromSource(page, ri, bi, 1, 1))); err != nil {
						panic(err)
					}
					if err := trie.InsertPagePart(pi, ri, bi, string(rowFromSource(page, ri, bi, 1, -1))); err != nil {
						panic(err)
					}
					if err := trie.InsertPagePart(pi, ri, bi, string(rowFromSource(page, ri, bi, -1, 1))); err != nil {
						panic(err)
					}
					if err := trie.InsertPagePart(pi, ri, bi, string(rowFromSource(page, ri, bi, -1, -1))); err != nil {
						panic(err)
					}
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

const (
	FlagRight    = 1 << iota // 1
	FlagLeft     = 1 << iota // 2
	FlagUp       = 1 << iota // 4
	FlagDown     = 1 << iota // 8
	FlagDiagonal = 1 << iota // 16
)

func (c *cypherTree) generate(this js.Value, args []js.Value) any {
	if len(args) != 1 {
		panic("bad args")
	}

	// Remove non-alpha characters
	in := nonAlphaRegex.ReplaceAllString(args[0].String(), "")

	fmt.Printf("Query: %q\n", in)

	rawCode, err := c.trie.ConstructPhraseLTR(in)
	if err != nil {
		return "error"
	}

	if len(rawCode) == 0 {
		return "not found"
	}

	out := &strings.Builder{}
	for i, part := range rawCode {
		fmt.Fprintf(out, "%d %d %d %d", part[0]+3, part[1]+1, part[2]+1, part[3])
		if i == len(rawCode)-1 {
			fmt.Fprintln(out)
		} else {
			fmt.Fprint(out, " ")
		}
	}

	return out.String()
}

func (c *cypherTree) reloadTrie(this js.Value, args []js.Value) any {
	if len(args) != 1 {
		panic("bad args")
	}
	fmt.Printf("reloading trie with: %08b\n", args[0].Int())
	c.trie = cypherTreeFromSource(c.source, args[0].Int())
	return nil
}

func main() {

	// Read the file
	source := loadSource()
	fmt.Printf("loaded %d pages\n", len(source))

	cypherGenerator := &cypherTree{
		trie:   cypherTreeFromSource(source, FlagRight),
		source: source,
	}

	js.Global().Set("generateCypher", js.FuncOf(cypherGenerator.generate))
	js.Global().Set("reload", js.FuncOf(cypherGenerator.reloadTrie))

	select {}
}
