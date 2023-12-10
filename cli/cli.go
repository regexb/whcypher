package main

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/regexb/whcypher"
	"github.com/urfave/cli/v2"
)

func loadSource(file string) ([][][]byte, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var out [][][]byte
	groups := bytes.Split(data, []byte("\n\n"))
	for _, g := range groups {
		out = append(out, bytes.Split(g, []byte("\n")))
	}
	return out, nil
}

func cypherTreeFromSource(source [][][]byte) (*whcypher.Trie, error) {
	trie := whcypher.NewTrie()
	for pi, page := range source {
		for ri, row := range page {
			for bi := range row {
				// traverse right
				if err := trie.InsertPagePart(whcypher.DirectionRight, pi, ri, bi, string(rowFromSource(page, ri, bi, 0, 1))); err != nil {
					return nil, err
				}

				// traverse left
				if err := trie.InsertPagePart(whcypher.DirectionLeft, pi, ri, bi, string(rowFromSource(page, ri, bi, 0, -1))); err != nil {
					return nil, err
				}

				// traverse up
				if err := trie.InsertPagePart(whcypher.DirectionUp, pi, ri, bi, string(rowFromSource(page, ri, bi, -1, 0))); err != nil {
					return nil, err
				}

				// traverse down
				if err := trie.InsertPagePart(whcypher.DirectionDown, pi, ri, bi, string(rowFromSource(page, ri, bi, 1, 0))); err != nil {
					return nil, err
				}

				// traverse diagonally
				if err := trie.InsertPagePart(whcypher.DirectionDiag, pi, ri, bi, string(rowFromSource(page, ri, bi, 1, 1))); err != nil {
					return nil, err
				}
				if err := trie.InsertPagePart(whcypher.DirectionDiag, pi, ri, bi, string(rowFromSource(page, ri, bi, 1, -1))); err != nil {
					return nil, err
				}
				if err := trie.InsertPagePart(whcypher.DirectionDiag, pi, ri, bi, string(rowFromSource(page, ri, bi, -1, 1))); err != nil {
					return nil, err
				}
				if err := trie.InsertPagePart(whcypher.DirectionDiag, pi, ri, bi, string(rowFromSource(page, ri, bi, -1, -1))); err != nil {
					return nil, err
				}
			}
		}
	}

	return trie, nil
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

func main() {
	app := &cli.App{
		Name:                   "whcli",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.PathFlag{Name: "file", Aliases: []string{"f"}, Required: true},
			&cli.StringFlag{Name: "input", Aliases: []string{"in", "i"}},
			&cli.IntFlag{Name: "page_offset", Aliases: []string{"po"}, Value: 1},
			&cli.IntFlag{Name: "row_offset", Aliases: []string{"ro"}, Value: 1},
			&cli.IntFlag{Name: "col_offset", Aliases: []string{"co"}, Value: 1},
			&cli.BoolFlag{Name: "ltr", Value: false},
			&cli.BoolFlag{Name: "right", Aliases: []string{"r"}, Value: true},
			&cli.BoolFlag{Name: "left", Aliases: []string{"l"}, Value: false},
			&cli.BoolFlag{Name: "up", Aliases: []string{"u"}, Value: false},
			&cli.BoolFlag{Name: "down", Aliases: []string{"d"}, Value: false},
			&cli.BoolFlag{Name: "allDirection", Aliases: []string{"all"}, Value: false},
		},
		Action: func(ctx *cli.Context) error {
			slog.Info("Starting...")

			sourceFile := ctx.Path("file")
			start := time.Now()
			source, err := loadSource(sourceFile)
			if err != nil {
				slog.Error("Failed to load source", "file", sourceFile)
				return err
			}
			slog.Info("Finished loading source", "pages", len(source), "time", time.Since(start))

			slog.Info("Loading source into trie")
			start = time.Now()
			cypher, err := cypherTreeFromSource(source)
			if err != nil {
				slog.Error("Failed to load source into cypher trie", "time", time.Since(start))
				return err
			}
			slog.Info("Finished loading source into cypher trie", "time", time.Since(start))

			dir := whcypher.Direction(0)

			if ctx.Bool("right") {
				dir |= whcypher.DirectionRight
			}
			if ctx.Bool("left") {
				dir |= whcypher.DirectionLeft
			}
			if ctx.Bool("up") {
				dir |= whcypher.DirectionUp
			}
			if ctx.Bool("down") {
				dir |= whcypher.DirectionDown
			}
			if ctx.Bool("allDirection") {
				dir = whcypher.DirectionDiag | whcypher.DirectionRight | whcypher.DirectionLeft | whcypher.DirectionUp | whcypher.DirectionDown
			}

			in := ctx.String("input")
			start = time.Now()
			var out [][5]int
			if ctx.Bool("ltr") {
				out, err = cypher.ConstructPhraseLTR(in, dir)
			} else {
				out, err = cypher.ConstructPhraseLongest(in, dir)
			}
			if err != nil {
				slog.Info("Failed to generate cypher", "phrase", in, "time", time.Since(start))
				return err
			}
			slog.Info("Finished generating cypher", slog.Any("raw", out), slog.Duration("time", time.Since(start)))

			fmt.Fprintln(ctx.App.Writer, "Generated cypher:")
			for i, part := range out {
				fmt.Fprintf(ctx.App.Writer, "%d %d %d %d", part[0]+ctx.Int("page_offset"), part[1]+ctx.Int("row_offset"), part[2]+ctx.Int("col_offset"), part[3])
				if i == len(out)-1 {
					fmt.Fprintln(ctx.App.Writer)
				} else {
					fmt.Fprint(ctx.App.Writer, " ")
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
