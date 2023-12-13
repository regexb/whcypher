//go:build js && wasm

package main

import (
	"bytes"
	_ "embed"
	"regexp"
	"strconv"
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

func (c *cypherTree) generate(this js.Value, args []js.Value) any {
	if len(args) != 3 {
		panic("bad args")
	}

	// Remove non-alpha characters
	in := nonAlphaRegex.ReplaceAllString(args[0].String(), "")

	algo := args[2].String()
	direction := whcypher.Direction(args[1].Int())

	println("Query ", in)

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

	return js.ValueOf(map[string]interface{}{
		"output":      rawToCode(rawCode),
		"debugOutput": rawToDebugString(rawCode),
		"locations":   rawToJSMap(rawCode),
	})
}

func rawToCode(rawCode [][5]int) string {
	outStr := ""
	for i, part := range rawCode {
		outStr += strconv.FormatInt(int64(part[0]+3), 10) + " "
		outStr += strconv.FormatInt(int64(part[1]+1), 10) + " "
		outStr += strconv.FormatInt(int64(part[2]+1), 10) + " "
		outStr += strconv.FormatInt(int64(part[3]), 10)

		if i != len(rawCode)-1 {
			outStr += " "
		}
	}
	return outStr
}

func rawToDebugString(rawCode [][5]int) string {
	outStr := ""
	for _, part := range rawCode {
		outStr += "[" + strconv.FormatInt(int64(part[0]+3), 10) + " "
		outStr += strconv.FormatInt(int64(part[1]+1), 10) + " "
		outStr += strconv.FormatInt(int64(part[2]+1), 10) + " "
		outStr += strconv.FormatInt(int64(part[3]), 10) + " "
		outStr += dirDebugMap[whcypher.Direction(part[4])] + "]"
	}
	return outStr
}

func rawToJSMap(rawCode [][5]int) []any {
	out := []any{}
	for _, part := range rawCode {
		out = append(out, map[string]any{
			"page": part[0] + 3,
			"row":  part[1] + 1,
			"col":  part[2] + 1,
			"len":  part[3],
			"dir":  whcypher.Direction(part[4]).String(),
		})
	}
	return out
}

var dirDebugMap = map[whcypher.Direction]string{
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
	println("loaded pages: ", len(source))

	cypherGenerator := &cypherTree{
		trie:   cypherTreeFromSource(source),
		source: source,
	}

	js.Global().Set("generateCypher", js.FuncOf(cypherGenerator.generate))

	select {}
}
