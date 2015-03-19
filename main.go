package main

import (
	"bufio"
	"fmt"
	"go/build"
	"io"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/tools/cover"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <profile> <file>\n", os.Args[0])
		os.Exit(1)
	}
	profiles, err := cover.ParseProfiles(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot parse cover profile %s: %s\n", os.Args[1], err.Error())
		os.Exit(1)
	}
	f, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open %s: %s\n", os.Args[2], err.Error())
		os.Exit(1)
	}
	defer f.Close()
	filename, err := getPackagePath(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find package for %s: %s\n", os.Args[2], err.Error())
		os.Exit(1)
	}
	var profile *cover.Profile
	for _, p := range profiles {
		if p.FileName == filename {
			profile = p
			break
		}
	}
	var blocks []cover.ProfileBlock
	if profile != nil {
		blocks = profile.Blocks
	}
	w := bufio.NewWriter(os.Stdout)
	if err := annotate(w, f, blocks); err != nil {
		fmt.Fprintf(os.Stderr, "cannot annotate %s: %s\n", os.Args[2], err.Error())
		os.Exit(1)
	}
	w.Flush()
}

func getPackagePath(filename string) (string, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	p, n := filepath.Split(abs)
	pkg, err := build.Default.ImportDir(p, build.FindOnly)
	if err != nil {
		return "", err
	}
	return path.Join(pkg.ImportPath, n), nil
}

func annotate(w io.Writer, r io.Reader, blocks []cover.ProfileBlock) error {
	br := bufio.NewReader(r)
	var linenum int
	var block int
	var keep bool
	var line []byte
	for {
		if !keep {
			linenum++
			var err error
			line, err = br.ReadSlice('\n')
			if err != nil {
				return nil
			}
		}
		keep = false
		if block >= len(blocks) {
			if err := annotateLine(w, ' ', line); err != nil {
				return err
			}
			continue
		}
		if linenum < blocks[block].StartLine {
			if err := annotateLine(w, ' ', line); err != nil {
				return err
			}
			continue
		}
		if linenum == blocks[block].StartLine {
			if isSpace(line[blocks[block].StartCol:]) {
				if err := annotateLine(w, ' ', line); err != nil {
					return err
				}
			} else {
				if err := annotateCodeLine(w, blocks[block].Count > 0, line); err != nil {
					return err
				}
			}
			continue
		}
		if linenum > blocks[block].EndLine || linenum == blocks[block].EndLine && isSpace(line[:blocks[block].EndCol-1]) {
			keep = true
			block++
			continue
		}
		if err := annotateCodeLine(w, blocks[block].Count > 0, line); err != nil {
			return err
		}
	}
}

func annotateLine(w io.Writer, c byte, line []byte) error {
	_, err := w.Write([]byte{c})
	if err != nil {
		return err
	}
	_, err = w.Write(line)
	if err != nil {
		return err
	}
	return nil
}

func annotateCodeLine(w io.Writer, tested bool, line []byte) error {
	if tested {
		return annotateLine(w, '+', line)
	}
	return annotateLine(w, '-', line)
}

func isSpace(b []byte) bool {
	for _, c := range b {
		switch c {
		case ' ', '\r', '\n', '\t':
			continue
		default:
			return false
		}
	}
	return true
}
