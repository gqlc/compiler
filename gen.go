package compiler

import (
	"context"
	"fmt"
	"github.com/gqlc/graphql/ast"
	"io"
	"os"
	"path/filepath"
)

// CodeGenerator provides a simple API for creating a code generator for
// any language desired.
//
type CodeGenerator interface {
	// Generate should handle multiple schemas in a single file.
	Generate(ctx context.Context, doc *ast.Document, opts string) error

	// GenerateAll should handle multiple schemas.
	GenerateAll(ctx context.Context, docs []*ast.Document, opts string) error
}

// GeneratorContext represents the directory to which
// the CodeGenerator is to write and other information
// about the context in which the Generator runs.
type GeneratorContext interface {
	// Open opens a file in the GeneratorContext (i.e. directory).
	Open(filename string) (io.WriteCloser, error)
}

type genContext struct {
	dir string
}

func (ctx *genContext) Open(filename string) (io.WriteCloser, error) {
	return os.Open(filepath.Join(ctx.dir, filename))
}

const genCtx = "genCtx"

// WithDir returns a prepared context.Context
// with a GeneratorContext rooted at the provided directory.
//
func WithDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, genCtx, &genContext{dir: dir})
}

// WithContext returns a prepared context.Context
// with the given GeneratorContext.
//
func WithContext(ctx context.Context, gCtx GeneratorContext) context.Context {
	return context.WithValue(ctx, genCtx, gCtx)
}

// Context returns the generator context.
func Context(ctx context.Context) GeneratorContext {
	return ctx.Value(genCtx).(GeneratorContext)
}

// Error represents an error from a generator.
type Error struct {
	// DocName is the document being worked on when error was encountered.
	DocName string

	// GenName is the generator name which encountered a problem.
	GenName string

	// Msg is any message the generator wants to provide back to the caller.
	Msg string
}

func (e Error) Error() string {
	return fmt.Sprintf("compiler: error occurred in %s:%s %s", e.GenName, e.DocName, e.Msg)
}
