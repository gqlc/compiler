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
	goldImpInFile = "graph.gql"
	goldInFile    = "test.gql"
	goldOutFile   = "test.md"
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
	depGqlFile, err := ioutil.ReadFile(filepath.Join(wd, goldImpInFile))
	if err != nil {
		t.Errorf("unexpected error when reading %s file: %s", goldImpInFile, err)
		return
	}
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

	// Parse input files
	docs, err := parser.ParseDocs(token.NewDocSet(), map[string]io.Reader{"graph.gql": bytes.NewReader(depGqlFile), "test": bytes.NewReader(gqlFile)}, 0)
	if err != nil {
		t.Errorf("unexpected error when parsing %s file: %s", goldInFile, err)
		return
	}
	docs, err = compiler.ReduceImports(docs)
	if err != nil {
		t.Errorf("unexpected error when import reducing %s file: %s", goldInFile, err)
		return
	}
	if len(docs) != 1 {
		t.Fail()
		return
	}
	doc := docs[0]
	for _, tg := range doc.Types {
		if tg == nil {
			t.Fail()
			return
		}
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

func TestComp(t *testing.T) {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("unexpected error when getting wd: %s", err)
		return
	}

	// Read in golden files
	depGqlFile, err := ioutil.ReadFile(filepath.Join(wd, goldImpInFile))
	if err != nil {
		t.Errorf("unexpected error when reading %s file: %s", goldImpInFile, err)
		return
	}
	gqlFile, err := ioutil.ReadFile(filepath.Join(wd, goldInFile))
	if err != nil {
		t.Errorf("unexpected error when reading %s file: %s", goldInFile, err)
		return
	}

	// Parse input files
	docs, err := parser.ParseDocs(token.NewDocSet(), map[string]io.Reader{"test": bytes.NewReader(gqlFile), "graph.gql": bytes.NewReader(depGqlFile)}, 0)
	if err != nil {
		t.Errorf("unexpected error when parsing %s file: %s", goldInFile, err)
		return
	}

	docs, err = compiler.ReduceImports(docs)
	if err != nil {
		t.Errorf("unexpected error when import reducing %s file: %s", goldInFile, err)
		return
	}
	if len(docs) != 1 {
		t.Fail()
		return
	}
}
