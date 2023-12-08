//go:build js && wasm

package main

import (
	"bytes"
	"crypto"
	_ "crypto/sha512"
	_ "embed"
	"encoding/hex"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/regexb/whcypher"
)

//go:embed source.txt
var sourceData []byte

func loadSource() [][][]byte {
	var out [][][]byte
	groups := bytes.Split(sourceData, []byte("\n\n"))
	for _, g := range groups {
		out = append(out, bytes.Split(g, []byte("\n")))
	}
	return out
}

func cypherTreeFromSource(source [][][]byte) *whcypher.Trie {
	trie := whcypher.NewTrie()
	for pi, page := range source {
		for ri, row := range page {
			if err := trie.InsertPageRow(pi, ri, string(row)); err != nil {
				panic(err)
			}
		}
	}
	return trie
}

type cypherTree struct {
	trie *whcypher.Trie
}

func (c *cypherTree) hash(this js.Value, args []js.Value) any {
	h := crypto.SHA512.New()
	h.Write([]byte(args[0].String()))

	return hex.EncodeToString(h.Sum(nil))
}

func (c *cypherTree) generate(this js.Value, args []js.Value) any {
	rawCode, err := c.trie.ConstructPhraseLTR(args[0].String())
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

func main() {

	// Read the file
	source := loadSource()
	print("loaded pages: ", len(source), "\n")

	cypherGenerator := &cypherTree{
		trie: cypherTreeFromSource(source),
	}

	js.Global().Set("generateCypher", js.FuncOf(cypherGenerator.generate))

	select {}
}
