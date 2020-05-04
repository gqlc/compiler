package compiler

import (
	"io"
	"strings"

	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
)

type tester interface {
	Fail()
	Logf(format string, args ...interface{})
}

// TestTypeChecker implements a few tests for custom type checkers.
// It is mainly focused around ensuring validation across imports.
//
func TestTypeChecker(t tester, v TypeChecker) {
	// Register builtin type
	RegisterTypes(&ast.TypeDecl{
		Tok: token.Token_SCALAR,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Msg"},
				Type: &ast.TypeSpec_Scalar{
					Scalar: &ast.ScalarType{Name: &ast.Ident{Name: "Msg"}},
				},
			},
		},
	})

	// Create a doc set
	docs, err := parser.ParseDocs(
		token.NewDocSet(),
		map[string]io.Reader{
			"a": strings.NewReader(`
schema {
  query: Query
}`),
			"b": strings.NewReader(`
type Query {
  echo(msg: Msg!): Msg!
}`),
		},
		0,
	)
	if err != nil {
		t.Logf("unexpected error when parsing test docs: %s", err)
		t.Fail()
		return
	}

	ir := ToIR(docs)
	if len(ir) != len(docs) {
		t.Logf("compiler: internal error with ToIR")
		t.Fail()
		return
	}

	errs := CheckTypes(ir, v)
	if len(errs) == 0 {
		return
	}

	for _, err := range errs {
		t.Logf("encountered error while type checking: %s", err)
	}
	t.Fail()
}
