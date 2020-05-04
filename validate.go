package compiler

import (
	"fmt"
	"sort"
	"strings"

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
func CheckTypes(docs IR, checkers ...TypeChecker) (errs []error) {
	docs[builtins] = toDeclMap(Types)

	for _, checker := range checkers {
		cerrs := checker.Check(docs)
		if cerrs == nil {
			continue
		}

		errs = append(errs, cerrs...)
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

// ImportValidator validates that all types are correctly imported.
var ImportValidator = TypeCheckerFn(validateImports)

func validateImports(docs IR) (errs []error) {
	imports := getImports(docs)

	for doc, mdecls := range docs {
		dimports := imports[doc]

		for _, decls := range mdecls {
			rtypes := getUnknownTypes(decls, mdecls)

			for _, rtype := range rtypes {
				d, _ := Lookup(rtype, docs)
				if d == nil {
					errs = append(errs, &TypeError{
						Doc: doc,
						Msg: fmt.Sprintf("undefined type: %s", rtype),
					})
					continue
				}

				if _, ok := dimports[d]; !ok {
					errs = append(errs, &TypeError{
						Doc: doc,
						Msg: fmt.Sprintf("unimported type: %s", rtype),
					})
				}
			}
		}
	}
	return
}

func getImports(docs IR) map[*ast.Document]map[*ast.Document]struct{} {
	imports := make(map[*ast.Document]map[*ast.Document]struct{}, len(docs))
	docMap := make(map[string]*ast.Document, len(docs))

	for doc := range docs {
		docMap[doc.Name] = doc
	}

	paths := make([]*ast.BasicLit, 0, len(docs)-1)
	for doc := range docs {
		imports[doc] = nil

		for _, dir := range doc.Directives {
			if dir.Name != "import" {
				continue
			}

			imps := dir.Args.Args[0]
			compList := imps.Value.(*ast.Arg_CompositeLit).CompositeLit.Value.(*ast.CompositeLit_ListLit)

			switch v := compList.ListLit.List.(type) {
			case *ast.ListLit_BasicList:
				paths = append(paths, v.BasicList.Values...)
			case *ast.ListLit_CompositeList:
				cpaths := v.CompositeList.Values
				for _, c := range cpaths {
					paths = append(paths, c.Value.(*ast.CompositeLit_BasicLit).BasicLit)
				}
			}

			dimports := make(map[*ast.Document]struct{}, len(paths))

			for _, path := range paths {
				p := strings.Trim(path.Value, "\"")
				dimports[docMap[p]] = struct{}{}
			}

			imports[doc] = dimports
		}

		paths = paths[:]
	}

	return imports
}

func getUnknownTypes(decls []*ast.TypeDecl, peers map[string][]*ast.TypeDecl) (unknowns []string) {
	var ts *ast.TypeSpec
	for _, decl := range decls {
		switch x := decl.Spec.(type) {
		case *ast.TypeDecl_TypeSpec:
			ts = x.TypeSpec
		case *ast.TypeDecl_TypeExtSpec:
			ts = x.TypeExtSpec.Type
		}

		switch x := ts.Type.(type) {
		case *ast.TypeSpec_Scalar, *ast.TypeSpec_Enum:
		case *ast.TypeSpec_Schema:
			fromFields(x.Schema.RootOps, peers, &unknowns)
		case *ast.TypeSpec_Object:
			fromIdents(x.Object.Interfaces, peers, &unknowns)
			fromFields(x.Object.Fields, peers, &unknowns)
		case *ast.TypeSpec_Interface:
			fromFields(x.Interface.Fields, peers, &unknowns)
		case *ast.TypeSpec_Union:
			fromIdents(x.Union.Members, peers, &unknowns)
		case *ast.TypeSpec_Input:
			fromArgs(x.Input.Fields, peers, &unknowns)
		case *ast.TypeSpec_Directive:
			fromArgs(x.Directive.Args, peers, &unknowns)
		}
	}
	return
}

func fromArgs(args *ast.InputValueList, peers map[string][]*ast.TypeDecl, unknowns *[]string) {
	for _, arg := range args.List {
		id := unwrapType(arg.Type)
		_, ok := peers[id.Name]
		if ok {
			continue
		}

		*unknowns = append(*unknowns, id.Name)
	}
}

func fromFields(fields *ast.FieldList, peers map[string][]*ast.TypeDecl, unknowns *[]string) {
	for _, field := range fields.List {
		fromArgs(field.Args, peers, unknowns)

		id := unwrapType(field.Type)
		_, ok := peers[id.Name]
		if ok {
			continue
		}

		*unknowns = append(*unknowns, id.Name)
	}
}

func fromIdents(idents []*ast.Ident, peers map[string][]*ast.TypeDecl, unknowns *[]string) {
	for _, id := range idents {
		_, ok := peers[id.Name]
		if ok {
			continue
		}

		*unknowns = append(*unknowns, id.Name)
	}
}
