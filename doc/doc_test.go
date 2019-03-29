package doc

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gqlc/compiler"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const (
	goldInFile  = "test.gql"
	goldOutFile = "test.md"
)

type testCtx struct {
	io.Writer
}

func (ctx testCtx) Open(filename string) (io.WriteCloser, error) { return ctx, nil }

func (ctx testCtx) Close() error { return nil }

func TestGenerator_Generate(t *testing.T) {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("unexpected error when getting wd: %s", err)
		return
	}

	// Read in golden files
	gqlFile, err := ioutil.ReadFile(filepath.Join(wd, goldInFile))
	if err != nil {
		t.Errorf("unexpected error when reading %s file: %s", goldInFile, err)
		return
	}
	expectedDoc, err := ioutil.ReadFile(filepath.Join(wd, goldOutFile))
	if err != nil {
		t.Errorf("unexpected error when reading %s file: %s", goldOutFile, err)
		return
	}

	// Parse input file
	doc, err := parser.ParseDoc(token.NewDocSet(), goldInFile, bytes.NewReader(gqlFile), 0)
	if err != nil {
		t.Errorf("unexpected error when parsing %s file: %s", goldInFile, err)
		return
	}

	// Run generator
	var b bytes.Buffer
	gen := new(Generator)
	ctx := compiler.WithContext(context.Background(), testCtx{Writer: &b})
	err = gen.Generate(ctx, doc, `{"title": "Test Documentation"}`)
	if err != nil {
		t.Error(err)
		return
	}

	// Compare generated output to golden output
	var line, col int
	oBytes := b.Bytes()
	for i := 0; i < len(expectedDoc); i++ {
		if expectedDoc[i] != oBytes[i] {
			fmt.Printf("%d:%d :%s:%s:\n", line+1, col+1, string(expectedDoc[i]), string(oBytes[i]))
			fmt.Println(b.String())
			t.Fail()
			break
		}
		if r := rune(expectedDoc[i]); r != '\n' && r != '\r' {
			continue
		}

		line++
		col++
	}
}
