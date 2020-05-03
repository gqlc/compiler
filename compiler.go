// Package compiler provides utilities for working with GraphQL documents.
package compiler

import (
	"github.com/gqlc/graphql/ast"
)

// IR is a representation used by the compiler APIs.
type IR map[*ast.Document]map[string][]*ast.TypeDecl

// Lookup returns the Type and the Document it belongs to, or nil.
func Lookup(name string, ir IR) (*ast.Document, []*ast.TypeDecl) {
	for doc, decls := range ir {
		if decl, ok := decls[name]; ok {
			return doc, decl
		}
	}

	return nil, nil
}

// ToIR converts a GraphQL Document to a intermediate
// representation for the compiler internals.
//
func ToIR(docs []*ast.Document) IR {
	ir := make(map[*ast.Document]map[string][]*ast.TypeDecl, len(docs))

	var ts *ast.TypeSpec
	for _, doc := range docs {
		types := make(map[string][]*ast.TypeDecl, len(doc.Types))
		ir[doc] = types

		for _, decl := range doc.Types {
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

			l := types[name]
			l = append(l, decl)
			types[name] = l
		}
	}

	return IR(ir)
}

// FromIR converts the compiler intermediate representation
// back to a simple slice of GraphQL type declarations.
//
func FromIR(ir IR) []*ast.Document {
	docs := make([]*ast.Document, len(ir))

	for doc, mdecls := range ir {
		doc.Types = doc.Types[:]

		for _, decls := range mdecls {
			doc.Types = append(doc.Types, decls...)
		}
	}

	return docs
}
