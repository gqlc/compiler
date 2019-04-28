package compiler

import (
	"container/list"
	"fmt"
	"github.com/gqlc/graphql/ast"
)

// IsImported reports whether or not a type declaration
// was imported from another Document i.e. it was copied
// from another AST.
//
func IsImported(gd *ast.GenDecl) (ok bool) {
	switch v := gd.Spec.(type) {
	case *ast.TypeSpec:
		_, ok = v.Name.(*ast.SelectorExpr)
	case *ast.TypeExtensionSpec:
		_, ok = v.Type.Name.(*ast.SelectorExpr)
	}
	return
}

// node extends a ast.Document with its imports as children, thus
// creating a tree structure which can be walked.
type node struct {
	Imported bool
	Childs   []*node
	*ast.Document
}

// ReduceImports reduces a set of Documents by including imported
// type defs into the Documents that they're imported into.
//
func ReduceImports(docs []*ast.Document) ([]*ast.Document, error) {
	// Map docs to nodes
	dMap := make(map[string]*node, len(docs))
	nodes := make([]*node, len(docs))
	for i, doc := range docs {
		n := &node{Document: doc}
		dMap[doc.Name] = n
		nodes[i] = n
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
	rDocs := make([]*ast.Document, len(forest))
	for i, trie := range forest {
		rDocs[i] = trie.Document
	}
	return rDocs, nil
}

func createImportTries(nodes []*node, dMap map[string]*node) ([]*node, error) {
	for i := 0; i < len(nodes); i++ {
		n := nodes[i]
		for _, ig := range n.Document.Imports {
			for _, imps := range ig.Specs {
				id := dMap[imps.Name.Name]
				if id == nil {
					return nil, Error{
						DocName: n.Name,
						GenName: "ReduceImports",
						Msg:     fmt.Sprintf("unknown import: %s", imps.Name.Name),
					}
				}
				if isCircular(n, id) {
					return nil, Error{
						DocName: n.Name,
						GenName: "ReduceImports",
						Msg:     fmt.Sprintf("circular imports: %s <-> %s", n.Name, id.Name),
					}
				}

				id.Imported = true
				n.Childs = append(n.Childs, id)
			}
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

func isCircular(a, b *node) bool {
	for _, c := range b.Childs {
		if a == c {
			return true
		}
	}
	return false
}

func resolveImports(root *node) (err error) {
	typeMap := make(map[string]*ast.GenDecl)

	// Add root types to typeMap
	err = addTypes(root, typeMap, func(name string, decl *ast.GenDecl, decls map[string]*ast.GenDecl) { decls[name] = decl })
	if err != nil {
		return
	}

	// Create queue and populate with children of root node
	q := list.New()
	for _, c := range root.Childs {
		q.PushBack(c)
	}

	// Walk import graph
	err = walk(q, typeMap, func(n *node, decls map[string]*ast.GenDecl) error {
		return addTypes(n, decls, func(name string, decl *ast.GenDecl, types map[string]*ast.GenDecl) {
			if decl == nil {
				types[name] = decl
				return
			}

			v, exists := types[name]
			switch {
			case v == nil && exists:
				types[name] = decl
			case v != nil && v != decl:
				types[name] = mergeTypes(v, decl)
			}
		})
	})

	// Convert type map to type slice
	i := 0
	root.Types = make([]*ast.GenDecl, len(typeMap))
	for _, v := range typeMap {
		root.Types[i] = v
		i++
	}
	return
}

// walk preforms a breadth-first walk of the import graph
func walk(q *list.List, typeMap map[string]*ast.GenDecl, f func(*node, map[string]*ast.GenDecl) error) (err error) {
	for q.Len() > 0 {
		v := q.Front()
		q.Remove(v)

		vn := v.Value.(*node)
		err = f(vn, typeMap)
		if err != nil {
			return
		}

		for _, c := range vn.Childs {
			q.PushBack(c)
		}
	}
	return
}

// mergeTypes handles merging TypeSpecExts with TypeSpecs or TypeSpecExts w/ TypeSpecExts
func mergeTypes(o, n *ast.GenDecl) *ast.GenDecl {
	decl := &ast.GenDecl{Doc: &ast.DocGroup{}}

	// Try asserting to TypeSpecs
	ots, otsOK := o.Spec.(*ast.TypeSpec)
	nts, ntsOK := n.Spec.(*ast.TypeSpec)

	// Convert both to TypeSpecs if they aren't already
	ts := &ast.TypeSpec{Doc: decl.Doc}
	switch {
	case otsOK && !ntsOK: // Old: Spec, New: Ext
		panic("compiler: circular import")
	case !otsOK && ntsOK: // Old: Ext, New: Spec
		// Set new spec
		decl.Spec = ts

		// Assert old to ext
		ext := o.Spec.(*ast.TypeExtensionSpec)
		ots = ext.Type
		ots.Doc = ext.Doc
	case !otsOK && !ntsOK: // Old: Ext, New: Ext
		// Set new spec
		decl.Spec = &ast.TypeExtensionSpec{Type: ts}

		// Assert to old and new ext
		oext := o.Spec.(*ast.TypeExtensionSpec)
		next := n.Spec.(*ast.TypeExtensionSpec)

		ots, nts = oext.Type, next.Type
		ots.Doc, nts.Doc = oext.Doc, next.Doc
	default:
		panic("compiler: unexpected merging of TypeSpec and TypeSpec")
	}

	// Merge doc groups
	ts.Doc.List = append(ts.Doc.List, ots.Doc.List...)
	ts.Doc.List = append(ts.Doc.List, nts.Doc.List...)

	// Merge type
	ts.Type = mergeExprs(ots.Type, nts.Type)

	// Add name and directives
	ts.Name = ots.Name
	ts.Dirs = append(ts.Dirs, ots.Dirs...)
	ts.Dirs = append(ts.Dirs, nts.Dirs...)

	return decl
}

func mergeExprs(o, n ast.Expr) (e ast.Expr) {
	switch u := o.(type) {
	case *ast.SchemaType:
		v, ok := n.(*ast.SchemaType)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.SchemaType{
			Fields: &ast.FieldList{
				List: append(u.Fields.List, v.Fields.List...),
			},
		}
	case *ast.ScalarType:
		_, ok := n.(*ast.ScalarType)
		if !ok {
			panic("mismatched types")
		}

		e = u
	case *ast.ObjectType:
		v, ok := n.(*ast.ObjectType)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.ObjectType{
			Impls: append(u.Impls, v.Impls...),
			Fields: &ast.FieldList{
				List: append(u.Fields.List, v.Fields.List...),
			},
		}
	case *ast.InterfaceType:
		v, ok := n.(*ast.InterfaceType)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.InterfaceType{
			Fields: &ast.FieldList{
				List: append(u.Fields.List, v.Fields.List...),
			},
		}
	case *ast.EnumType:
		v, ok := n.(*ast.EnumType)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.EnumType{
			Fields: &ast.FieldList{
				List: append(u.Fields.List, v.Fields.List...),
			},
		}
	case *ast.UnionType:
		v, ok := n.(*ast.UnionType)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.UnionType{
			Members: append(u.Members, v.Members...),
		}
	case *ast.InputType:
		v, ok := n.(*ast.InputType)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.InputType{
			Fields: &ast.FieldList{
				List: append(u.Fields.List, v.Fields.List...),
			},
		}
	}
	return
}

func addTypes(n *node, typeMap map[string]*ast.GenDecl, add func(string, *ast.GenDecl, map[string]*ast.GenDecl)) (err error) {
	for _, tg := range n.Types {
		var ts *ast.TypeSpec
		switch v := tg.Spec.(type) {
		case *ast.TypeSpec:
			ts = v
		case *ast.TypeExtensionSpec:
			ts = v.Type
		default:
			panic("unknown spec")
		}

		var name string
		switch v := ts.Name.(type) {
		case *ast.Ident:
			name = fmt.Sprintf("%s.%s", n.Name, v.Name)
		case *ast.SelectorExpr:
			name = fmt.Sprintf("%s.%s", v.Sel.Name, v.X.(*ast.Ident).Name)
		}

		add(name, tg, typeMap)

		switch v := ts.Type.(type) {
		case *ast.ScalarType:
			// Scalar doesn't need anything done to it
		case *ast.ObjectType:
			for _, impl := range v.Impls {
				se, ok := impl.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				name = fmt.Sprintf("%s.%s", se.Sel.Name, se.X.(*ast.Ident).Name)
				add(name, nil, typeMap)
			}

			err = resolveFieldList(v.Fields, typeMap, add)
			if err != nil {
				return
			}
		case *ast.InterfaceType:
			err = resolveFieldList(v.Fields, typeMap, add)
			if err != nil {
				return
			}
		case *ast.UnionType:
			for _, impl := range v.Members {
				se, ok := impl.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				name = fmt.Sprintf("%s.%s", se.Sel.Name, se.X.(*ast.Ident).Name)
				add(name, nil, typeMap)
			}
		case *ast.EnumType:
			err = resolveFieldList(v.Fields, typeMap, add)
			if err != nil {
				return
			}
		case *ast.InputType:
			err = resolveFieldList(v.Fields, typeMap, add)
			if err != nil {
				return err
			}
		case *ast.DirectiveType:
			err = resolveFieldList(v.Args, typeMap, add)
			if err != nil {
				return err
			}
		}
	}

	return
}

func resolveFieldList(fields *ast.FieldList, typeMap map[string]*ast.GenDecl, add func(string, *ast.GenDecl, map[string]*ast.GenDecl)) (err error) {
	if fields == nil {
		return
	}

	for _, f := range fields.List {
		err = resolveFieldList(f.Args, typeMap, add)
		if err != nil {
			return
		}

		t := unwrapType(f.Type)
		if se, ok := t.(*ast.SelectorExpr); ok {
			name := fmt.Sprintf("%s.%s", se.Sel.Name, se.X.(*ast.Ident).Name)
			add(name, nil, typeMap)
		}
	}
	return
}

func unwrapType(t ast.Expr) ast.Expr {
	switch v := t.(type) {
	case *ast.SelectorExpr:
		return v
	case *ast.Ident:
		return v
	case *ast.List:
		return unwrapType(v.Type)
	case *ast.NonNull:
		return unwrapType(v.Type)
	}

	return nil
}
