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

func loadSource(file string) ([][]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var out [][]string
	groups := bytes.Split(data, []byte("\n\n"))
	for _, g := range groups {
		var sub []string
		lines := bytes.Split(g, []byte("\n"))
		for _, l := range lines {
			sub = append(sub, string(l))
		}
		out = append(out, sub)
	}
	return out, nil
}

func cypherTreeFromSource(source [][]string) (*whcypher.Trie, error) {
	trie := whcypher.NewTrie()

	for pi, page := range source {
		for ri, row := range page {
			if err := trie.InsertPageRow(pi, ri, row); err != nil {
				return nil, err
			}
		}
	}

	return trie, nil
}

func main() {
	app := &cli.App{
		Name: "whcli",
		Flags: []cli.Flag{
			&cli.PathFlag{Name: "file", Aliases: []string{"f"}},
			&cli.StringFlag{Name: "input", Aliases: []string{"in", "i"}},
			&cli.IntFlag{Name: "page_offset", Aliases: []string{"p"}, Value: 1},
			&cli.IntFlag{Name: "row_offset", Aliases: []string{"r"}, Value: 1},
			&cli.IntFlag{Name: "col_offset", Aliases: []string{"c"}, Value: 1},
			&cli.BoolFlag{Name: "ltr", Value: false},
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

			in := ctx.String("input")
			start = time.Now()
			var out [][4]int
			if ctx.Bool("ltr") {
				out, err = cypher.ConstructPhraseLTR(in)
			} else {
				out, err = cypher.ConstructPhraseLongest(in)
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
