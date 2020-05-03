// Package compiler provides utilities for working with GraphQL documents.
package compiler

import (
	"github.com/gqlc/graphql/ast"
)

// ToIR converts a GraphQL Document to a intermediate
// representation for the compiler internals.
//
func ToIR(types []*ast.TypeDecl) map[string][]*ast.TypeDecl {
	ir := make(map[string][]*ast.TypeDecl, len(types))

	var ts *ast.TypeSpec
	for _, decl := range types {
		switch v := decl.Spec.(type) {
		case *ast.TypeDecl_TypeSpec:
			ts = v.TypeSpec
		case *ast.TypeDecl_TypeExtSpec:
			ts = v.TypeExtSpec.Type
		}

		name := "schema"
		if ts.Name != nil {
			name = ts.Name.Name
		}

		l := ir[name]
		l = append(l, decl)
		ir[name] = l
	}

	return ir
}

// FromIR converts the compiler intermediate representation
// back to a simple slice of GraphQL type declarations.
//
func FromIR(ir map[string][]*ast.TypeDecl) []*ast.TypeDecl {
	types := make([]*ast.TypeDecl, 0, len(ir))

	for _, decls := range ir {
		types = append(types, decls...)
	}

	return types
}
