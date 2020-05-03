package compiler

import (
	"fmt"
	"sort"

	"github.com/gqlc/graphql/ast"
)

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
	// Check analyzes the types in a GraphQL Document and returns any
	// problems it has detected.
	//
	Check(directives []*ast.DirectiveLit, types map[string][]*ast.TypeDecl) []error
}

// TypeCheckerFn represents a single function behaving as a TypeChecker.
type TypeCheckerFn func([]*ast.DirectiveLit, map[string][]*ast.TypeDecl) []error

// Check calls the TypeCheckerFn given the GraphQL Document.
func (f TypeCheckerFn) Check(directives []*ast.DirectiveLit, types map[string][]*ast.TypeDecl) []error {
	return f(directives, types)
}

// CheckTypes is a helper function for running a suite of
// type checking on several GraphQL Documents. Any TypeDecls
// passed to RegisterTypes will be appended to each Documents' Type list.
//
// All errors encountered will be appended into the return slice: errs
//
func CheckTypes(docs map[*ast.Document]map[string][]*ast.TypeDecl, checkers ...TypeChecker) (errs []*TypeError) {
	builtins := ToIR(Types)

	for d, doc := range docs {
		types := merge(builtins, doc) // this order ensures user types can override builtin types

		sortTypes(types)

		for _, checker := range checkers {
			cerrs := checker.Check(d.Directives, types)
			for _, err := range cerrs {
				errs = append(errs, &TypeError{
					Doc: d,
					Msg: err.Error(),
				})
			}
		}
	}
	return
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

// Types contains the builtin types provided by the compiler
// and any custom types given to RegisterTypes.
//
var Types []*ast.TypeDecl

// RegisterTypes registers pre-defined types with the compiler.
func RegisterTypes(decls ...*ast.TypeDecl) { Types = append(Types, decls...) }
