package compiler

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
)

// Types contains the builtin types provided by the compiler
// and any custom types given to RegisterTypes.
//
var Types []*ast.TypeDecl

// RegisterTypes registers pre-defined types with the compiler.
func RegisterTypes(decls ...*ast.TypeDecl) { Types = append(Types, decls...) }

// TypeError represents a type error.
type TypeError struct {
	// Document where type error was discovered
	Doc *ast.Document

	// Type error message
	Msg string
}

// Error returns a string representation of a TypeError.
func (e *TypeError) Error() string {
	return fmt.Sprintf("compiler: encountered type error in %s:%s", e.Doc.Name, e.Msg)
}

// TypeChecker represents type checking functionality for a GraphQL Document.
type TypeChecker interface {
	// Check performs type checking on the types in the IR.
	Check(ir IR) []error
}

// TypeCheckerFn represents a single function behaving as a TypeChecker.
type TypeCheckerFn func(IR) []error

// Check calls the TypeCheckerFn given the GraphQL Document.
func (f TypeCheckerFn) Check(ir IR) []error {
	return f(ir)
}

// CheckTypes is a helper function for running a suite of
// type checking on several GraphQL Documents. Any types given
// to RegisterTypes will included as their very own document.
//
func CheckTypes(docs IR, checkers ...TypeChecker) (errs []*TypeError) {
	builtins := &ast.Document{Name: "gqlc.compiler.types"}

	docs[builtins] = toDeclMap(Types)

	for _, checker := range checkers {
		cerrs := checker.Check(docs)
		for _, err := range cerrs {
			errs = append(errs, &TypeError{
				Msg: err.Error(),
			})
		}
	}

	return
}

func toDeclMap(decls []*ast.TypeDecl) map[string][]*ast.TypeDecl {
	m := make(map[string][]*ast.TypeDecl, len(decls))

	var ts *ast.TypeSpec
	for _, decl := range decls {
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

		l := m[name]
		l = append(l, decl)
		m[name] = l
	}

	return m
}

func merge(a, b map[string][]*ast.TypeDecl) map[string][]*ast.TypeDecl {
	c := make(map[string][]*ast.TypeDecl, len(a)+len(b))
	for name, l := range a {
		c[name] = l
	}
	for name, l := range b {
		c[name] = l
	}
	return c
}

func sortTypes(types map[string][]*ast.TypeDecl) {
	for name, l := range types {
		sort.Slice(l, func(i, j int) bool {
			_, a := l[i].Spec.(*ast.TypeDecl_TypeSpec)
			_, b := l[j].Spec.(*ast.TypeDecl_TypeExtSpec)
			return a && b
		})

		types[name] = l
	}
}

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
