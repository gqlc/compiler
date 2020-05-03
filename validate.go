package compiler

import (
	"fmt"
	"sort"

	"github.com/gqlc/graphql/ast"
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

var builtins = &ast.Document{Name: "gqlc.compiler.types"}

// CheckTypes is a helper function for running a suite of
// type checking on several GraphQL Documents. Any types given
// to RegisterTypes will included as their very own document.
//
func CheckTypes(docs IR, checkers ...TypeChecker) (errs []*TypeError) {
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

// ImportValidator validates that all types referenced are correctly imported.
var ImportValidator = TypeCheckerFn(validateImports)

func validateImports(docs IR) (errs []error) {
	return
}
