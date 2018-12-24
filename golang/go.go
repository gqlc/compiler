// Package golang contains a Go generator for GraphQL Documents.
package golang

import (
	"context"
	"github.com/gqlc/graphql/ast"
)

// Generator generates Go code for a GraphQL schema.
type Generator struct{}

// Generate generates Go code for all schemas found within the given document.
func (gen *Generator) Generate(ctx context.Context, doc *ast.Document, opts string) error {
	return nil
}

// GenerateAll generates Go code for all schemas found within all the given documents.
func (gen *Generator) GenerateAll(ctx context.Context, docs []*ast.Document, opts string) error {
	return nil
}
