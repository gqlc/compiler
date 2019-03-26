package compiler

import (
	"context"
	"github.com/gqlc/graphql/ast"
	"io"
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

// GenCtx represents request scoped data
// for each CodeGenerator.Generate(All) call.
type GenCtx struct {
	// Out is where the generator should output
	// all generated text.
	Out io.Writer
}

const genOut = "output"

// WithOutput returns a prepared context.Context with the given
// output source.
func WithOutput(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, genOut, w)
}

// Output returns the output source for the generator to use.
func Output(ctx context.Context) io.Writer {
	return ctx.Value(genOut).(io.Writer)
}