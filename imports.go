package compiler

import (
	"container/list"
	"fmt"
	"github.com/gqlc/graphql/ast"
	"strings"
)

// ImportError represents an error with the imports in a GraphQL Document
type ImportError struct {
	Doc *ast.Document
	Msg string
}

func (e *ImportError) Error() string {
	return fmt.Sprintf("compiler: import error encountered in %s:%s", e.Doc.Name, e.Msg)
}

// node extends a ast.Document with its imports as children, thus
// creating a tree structure which can be walked.
//
type node struct {
	Imported bool
	Childs   []*node
	Types    map[string][]*ast.TypeDecl
	*ast.Document
}

// ReduceImports reduces a set of Documents by including imported
// type defs into the Documents that they're imported into.
//
// To import a Document the @import directive is used:
// directive @import(paths: [String]) on DOCUMENT
//
func ReduceImports(docs IR) (IR, error) {
	// Map docs to nodes
	dMap := make(map[string]*node, len(docs))
	nodes := make([]*node, 0, len(docs))
	for doc, typeMap := range docs {
		n := &node{Types: typeMap, Document: doc}
		dMap[doc.Name] = n
		nodes = append(nodes, n)
	}

	// Construct import trees from set of documents (technically it could be an import tree forest)
	forest, err := createImportTries(nodes, dMap)
	if err != nil {
		return nil, err
	}

	// Resolve import trees
	for _, trie := range forest {
		err := resolveImports(trie)
		if err != nil {
			return nil, err
		}
	}

	// Unwrap nodes to *ast.Documents
	rDocs := make(map[*ast.Document]map[string][]*ast.TypeDecl, len(forest))
	for _, trie := range forest {
		rDocs[trie.Document] = trie.Types
	}
	return rDocs, nil
}

func createImportTries(nodes []*node, dMap map[string]*node) ([]*node, error) {
	for i := 0; i < len(nodes); i++ {
		n := nodes[i]
		index := -1
		for ind, dir := range n.Document.Directives {
			if dir.Name != "import" {
				continue
			}
			index = ind

			imps := dir.Args.Args[0]
			compList := imps.Value.(*ast.Arg_CompositeLit).CompositeLit.Value.(*ast.CompositeLit_ListLit)

			var paths []*ast.BasicLit
			switch v := compList.ListLit.List.(type) {
			case *ast.ListLit_BasicList:
				paths = append(paths, v.BasicList.Values...)
			case *ast.ListLit_CompositeList:
				cpaths := v.CompositeList.Values
				paths = make([]*ast.BasicLit, len(cpaths))
				for i, c := range cpaths {
					paths[i] = c.Value.(*ast.CompositeLit_BasicLit).BasicLit
				}
			}

			for _, imp := range paths {
				path := strings.Trim(imp.Value, "\"")
				id := dMap[path]
				if id == nil {
					return nil, &ImportError{
						Doc: n.Document,
						Msg: fmt.Sprintf("unknown import: %s", imp.Value),
					}
				}
				if isCircular(n, id) {
					return nil, &ImportError{
						Doc: n.Document,
						Msg: fmt.Sprintf("circular imports: %s <-> %s", n.Name, id.Name),
					}
				}

				id.Imported = true
				n.Childs = append(n.Childs, id)
			}
		}

		if index > -1 {
			copy(n.Document.Directives[index:], n.Document.Directives[index+1:])
			n.Document.Directives[len(n.Document.Directives)-1] = nil // or the zero value of T
			n.Document.Directives = n.Document.Directives[:len(n.Document.Directives)-1]
		}
	}

	// Remove any imported nodes which are not root nodes
	for i := 0; i < len(nodes); i++ {
		if !nodes[i].Imported {
			continue
		}

		copy(nodes[i:], nodes[i+1:])
		nodes[len(nodes)-1] = nil // or the zero value of T
		nodes = nodes[:len(nodes)-1]
		i--
	}

	return nodes, nil
}

func resolveImports(root *node) error {
	typeMap := make(map[string][]*ast.TypeDecl)
	directives := make(map[string]*ast.DirectiveLit)
	defer func() {
		removeBuiltins(typeMap)

		root.Types = typeMap

		if root.Directives != nil {
			root.Directives = root.Directives[:0]
		}
		for _, d := range directives {
			root.Directives = append(root.Directives, d)
		}
	}()

	// Collect root directives
	for _, d := range root.Directives {
		directives[d.Name] = d
	}

	// Add root types to typeMap
	for name, decls := range root.Types {
		typeMap[name] = decls

		for _, decl := range decls {
			addDeps(decl, typeMap, root.Types)
		}
	}

	// Create queue and populate with children of root node
	q := list.New()
	for _, c := range root.Childs {
		q.PushBack(c)
	}

	// Walk import graph
	return walk(q, typeMap, func(n *node, decls map[string][]*ast.TypeDecl) {
		// Collect directives
		for _, d := range n.Directives {
			if _, exists := directives[d.Name]; !exists {
				directives[d.Name] = d
			}
		}

		// Collect types
		addTypes(n, decls)
	})
}

// walk preforms a breadth-first walk of the import graph
func walk(q *list.List, typeMap map[string][]*ast.TypeDecl, f func(*node, map[string][]*ast.TypeDecl)) (err error) {
	for q.Len() > 0 {
		v := q.Front()
		q.Remove(v)

		vn := v.Value.(*node)
		f(vn, typeMap)

		for _, c := range vn.Childs {
			q.PushBack(c)
		}
	}
	return
}

func addTypes(n *node, typeMap map[string][]*ast.TypeDecl) {
	for name, decls := range typeMap {
		if len(decls) > 0 {
			continue
		}

		d, ok := n.Types[name]
		if !ok {
			continue
		}

		for _, decl := range d {
			addDeps(decl, typeMap, n.Types)
		}

		typeMap[name] = append(decls, d...)
	}

	return
}

func addDeps(decl *ast.TypeDecl, typeMap, peers map[string][]*ast.TypeDecl) {
	var ts *ast.TypeSpec
	switch v := decl.Spec.(type) {
	case *ast.TypeDecl_TypeSpec:
		ts = v.TypeSpec
	case *ast.TypeDecl_TypeExtSpec:
		ts = v.TypeExtSpec.Type
	}

	switch v := ts.Type.(type) {
	case *ast.TypeSpec_Scalar:
	case *ast.TypeSpec_Enum:
	case *ast.TypeSpec_Schema:
		resolveFieldList(v.Schema.RootOps, typeMap, peers)
	case *ast.TypeSpec_Object:
		for _, i := range v.Object.Interfaces {
			if _, exists := typeMap[i.Name]; exists {
				continue
			}

			typeMap[i.Name] = peers[i.Name] // either init as peer or nil
		}

		resolveFieldList(v.Object.Fields, typeMap, peers)
	case *ast.TypeSpec_Interface:
		resolveFieldList(v.Interface.Fields, typeMap, peers)
	case *ast.TypeSpec_Union:
		for _, i := range v.Union.Members {
			if _, exists := typeMap[i.Name]; exists {
				continue
			}

			typeMap[i.Name] = peers[i.Name] // either init as peer or nil
		}
	case *ast.TypeSpec_Input:
		resolveArgList(v.Input.Fields, typeMap, peers)
	case *ast.TypeSpec_Directive:
		resolveArgList(v.Directive.Args, typeMap, peers)
	}
}

func resolveFieldList(fields *ast.FieldList, typeMap, peers map[string][]*ast.TypeDecl) {
	if fields == nil {
		return
	}

	for _, f := range fields.List {
		resolveArgList(f.Args, typeMap, peers)

		var t *ast.Ident
		switch v := f.Type.(type) {
		case *ast.Field_Ident:
			t = v.Ident
		case *ast.Field_NonNull:
			t = unwrapType(v.NonNull)
		case *ast.Field_List:
			t = unwrapType(v.List)
		default:
			return
		}
		if isBuiltin(t.Name) {
			continue
		}

		if _, exists := typeMap[t.Name]; exists {
			continue
		}
		typeMap[t.Name] = peers[t.Name] // either init as peer or nil
	}
	return
}

func resolveArgList(fields *ast.InputValueList, typeMap, peers map[string][]*ast.TypeDecl) {
	if fields == nil {
		return
	}

	for _, f := range fields.List {
		var t *ast.Ident
		switch v := f.Type.(type) {
		case *ast.InputValue_Ident:
			t = v.Ident
		case *ast.InputValue_NonNull:
			t = unwrapType(v.NonNull)
		case *ast.InputValue_List:
			t = unwrapType(v.List)
		default:
			return
		}
		if isBuiltin(t.Name) {
			continue
		}

		if _, exists := typeMap[t.Name]; exists {
			continue
		}
		typeMap[t.Name] = peers[t.Name] // either init as peer or nil

	}
	return
}

func isCircular(a, b *node) bool {
	for _, c := range b.Childs {
		if a == c {
			return true
		}
	}
	return false
}

func isBuiltin(name string) bool {
	return name == "ID" || name == "Boolean" || name == "String" || name == "Int" || name == "Float"
}

func removeBuiltins(typeMap map[string][]*ast.TypeDecl) {
	delete(typeMap, "ID")
	delete(typeMap, "Boolean")
	delete(typeMap, "Int")
	delete(typeMap, "String")
	delete(typeMap, "Float")
}

func unwrapType(i interface{}) *ast.Ident {
	switch v := i.(type) {
	case *ast.Ident:
		return v
	case *ast.List:
		switch u := v.Type.(type) {
		case *ast.List_Ident:
			return u.Ident
		case *ast.List_List:
			return unwrapType(u.List)
		case *ast.List_NonNull:
			return unwrapType(u.NonNull)
		}
	case *ast.NonNull:
		switch u := v.Type.(type) {
		case *ast.NonNull_Ident:
			return u.Ident
		case *ast.NonNull_List:
			return unwrapType(u.List)
		}
	}

	return nil
}
