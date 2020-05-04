// Package compiler provides utilities for working with GraphQL documents.
package compiler

import (
	"sort"

	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/token"
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

		sortTypes(types)
	}

	return IR(ir)
}

// FromIR converts the compiler intermediate representation
// back to a simple slice of GraphQL Documents.
//
func FromIR(ir IR) []*ast.Document {
	docs := make([]*ast.Document, 0, len(ir))

	for doc, mdecls := range ir {
		docs = append(docs, doc)
		doc.Types = doc.Types[:0]

		sortTypes(mdecls)
		for _, decls := range mdecls {
			doc.Types = append(doc.Types, decls...)
		}

		sort.Sort(byTypeAndName{types: &doc.Types})
	}

	return docs
}

type byTypeAndName struct {
	types *[]*ast.TypeDecl
}

type ord uint8

const (
	schema ord = iota
	scalars
	objects
	interfaces
	unions
	enums
	inputs
	directives
)

func (s byTypeAndName) Less(i, j int) bool {
	is := (*s.types)[i]
	js := (*s.types)[j]

	its := is.Spec.(*ast.TypeDecl_TypeSpec).TypeSpec
	jts := js.Spec.(*ast.TypeDecl_TypeSpec).TypeSpec

	if is.Tok == js.Tok {
		return its.Name.Name < jts.Name.Name
	}

	var iOrd, jOrd ord
	switch is.Tok {
	case token.Token_SCHEMA:
		iOrd = schema
	case token.Token_SCALAR:
		iOrd = scalars
	case token.Token_TYPE:
		iOrd = objects
	case token.Token_INTERFACE:
		iOrd = interfaces
	case token.Token_UNION:
		iOrd = unions
	case token.Token_ENUM:
		iOrd = enums
	case token.Token_INPUT:
		iOrd = inputs
	case token.Token_DIRECTIVE:
		iOrd = directives
	}

	switch js.Tok {
	case token.Token_SCHEMA:
		jOrd = schema
	case token.Token_SCALAR:
		jOrd = scalars
	case token.Token_TYPE:
		jOrd = objects
	case token.Token_INTERFACE:
		jOrd = interfaces
	case token.Token_UNION:
		jOrd = unions
	case token.Token_ENUM:
		jOrd = enums
	case token.Token_INPUT:
		jOrd = inputs
	case token.Token_DIRECTIVE:
		jOrd = directives
	}

	return iOrd < jOrd
}

func (s byTypeAndName) Swap(i, j int) {
	jdecl := (*s.types)[j]
	(*s.types)[j] = (*s.types)[i]
	(*s.types)[i] = jdecl
}

func (s byTypeAndName) Len() int { return len(*s.types) }
