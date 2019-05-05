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
