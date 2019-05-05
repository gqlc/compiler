// Package dart contains a Dart generator for GraphQL Documents.
package dart

import (
	"context"
	"github.com/gqlc/graphql/ast"
)

// Generator generates Dart code for a GraphQL schema.
type Generator struct{}

// Generate generates Dart code for all schemas found within the given document.
func (gen *Generator) Generate(ctx context.Context, doc *ast.Document, opts string) error {
	return nil
}
