package compiler

import "github.com/gqlc/graphql/ast"

// SimplifyImports simplifies a set of documents by including imported
// type defs into the Documents that they're imported into.
//
func SimplifyImports(docs []*ast.Document) ([]*ast.Document, error) {
	// TODO
	return nil, nil
}
