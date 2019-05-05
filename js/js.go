// Package js contains a Javascript generator for GraphQL Documents.
package js

import (
	"context"
	"github.com/gqlc/graphql/ast"
)

// Generator generates Javascript code for a GraphQL schema.
type Generator struct{}

// Generate generates Javascript code for all schemas found within the given document.
func (gen *Generator) Generate(ctx context.Context, doc *ast.Document, opts string) error {
	return nil
}
