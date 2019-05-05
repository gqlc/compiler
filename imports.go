package compiler

import (
	"container/list"
	"fmt"
	"github.com/gqlc/graphql/ast"
	"strings"
)

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
		for _, dir := range n.Document.Directives {
			if dir.Name != "import" {
				continue
			}

			imps := dir.Args.Args[0]
			compList := imps.Value.(*ast.Arg_CompositeLit).CompositeLit.Value.(*ast.CompositeLit_ListLit)
			paths := compList.ListLit.List.(*ast.ListLit_BasicList).BasicList.Values
			for _, imp := range paths {
				path := strings.Trim(imp.Value, "\"")
				id := dMap[path]
				if id == nil {
					return nil, Error{
						DocName: n.Name,
						GenName: "ReduceImports",
						Msg:     fmt.Sprintf("unknown import: %s", imp.Value),
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

func resolveImports(root *node) (err error) {
	typeMap := make(map[string]*ast.TypeDecl)
	defer func() {
		// Convert type map to type slice
		i := 0
		root.Types = make([]*ast.TypeDecl, len(typeMap))
		for _, v := range typeMap {
			root.Types[i] = v
			i++
		}
	}()

	// Add root types to typeMap
	err = addTypes(root, typeMap, func(name string, decl *ast.TypeDecl, decls map[string]*ast.TypeDecl) bool {
		if isBuiltinType(name) {
			return true
		}

		if tg, exists := decls[name]; exists && tg != nil {
			return false
		}

		decls[name] = decl
		return false
	})
	if err != nil {
		return
	}

	// Create queue and populate with children of root node
	q := list.New()
	for _, c := range root.Childs {
		q.PushBack(c)
	}

	// Walk import graph
	err = walk(q, typeMap, func(n *node, decls map[string]*ast.TypeDecl) error {
		return addTypes(n, decls, func(name string, decl *ast.TypeDecl, types map[string]*ast.TypeDecl) bool {
			// Check if builtin type
			if isBuiltinType(name) {
				return true
			}

			// Skip if it isn't imported or previously encountered peer type
			_, exists := types[name]
			switch {
			case !exists && decl != nil:
				return false
			case exists && decl == nil:
				return true
			}

			if decl == nil {
				types[name] = decl
				return false
			}

			v, exists := types[name]
			switch {
			case v == nil && exists:
				types[name] = decl
			case v != nil && v != decl:
				types[name] = mergeTypes(v, decl)
			}
			return false
		})
	})
	if err != nil {
		return
	}

	// Check for any un-imported peer types
	peerMap := make(map[string]*ast.TypeDecl)
	for name, tg := range typeMap {
		if tg == nil {
			peerMap[name] = nil
		}
	}
	for len(peerMap) > 0 {
		q = q.Init()
		for _, c := range root.Childs {
			q.PushBack(c)
		}

		// Walk import graph
		err = walk(q, peerMap, func(n *node, decls map[string]*ast.TypeDecl) error {
			return addTypes(n, decls, func(name string, decl *ast.TypeDecl, types map[string]*ast.TypeDecl) bool {
				// Check if builtin type
				if isBuiltinType(name) {
					return true
				}

				// Skip if it isn't imported or previously encountered peer type
				_, exists := types[name]
				switch {
				case !exists && decl != nil:
					return false
				case exists && decl == nil:
					return true
				}

				if decl == nil {
					types[name] = decl
					return false
				}

				v, exists := types[name]
				switch {
				case v == nil && exists:
					types[name] = decl
				case v != nil && v != decl:
					types[name] = mergeTypes(v, decl)
				}
				return false
			})
		})
		if err != nil {
			return
		}

		for name, tg := range peerMap {
			if tg == nil {
				continue
			}

			typeMap[name] = tg
			delete(peerMap, name)
		}
	}

	return
}

// walk preforms a breadth-first walk of the import graph
func walk(q *list.List, typeMap map[string]*ast.TypeDecl, f func(*node, map[string]*ast.TypeDecl) error) (err error) {
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
func mergeTypes(o, n *ast.TypeDecl) *ast.TypeDecl {
	decl := &ast.TypeDecl{Doc: &ast.DocGroup{}}

	// Try asserting to TypeSpecs
	ots, otsOK := o.Spec.(*ast.TypeDecl_TypeSpec)
	nts, ntsOK := n.Spec.(*ast.TypeDecl_TypeSpec)

	// Convert both to TypeSpecs if they aren't already
	ts := &ast.TypeSpec{Doc: decl.Doc}
	dts := &ast.TypeDecl_TypeSpec{TypeSpec: ts}
	switch {
	case otsOK && !ntsOK: // Old: Spec, New: Ext
		panic("compiler: circular import")
	case !otsOK && ntsOK: // Old: Ext, New: Spec
		// Set new spec
		decl.Spec = dts

		// Assert old to ext
		ots = new(ast.TypeDecl_TypeSpec)
		ext := o.Spec.(*ast.TypeDecl_TypeExtSpec).TypeExtSpec
		ots.TypeSpec = ext.Type
		ots.TypeSpec.Doc = ext.Doc
	case !otsOK && !ntsOK: // Old: Ext, New: Ext
		// Set new spec
		decl.Spec = &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{Type: ts}}

		// Assert to old and new ext
		oext := o.Spec.(*ast.TypeDecl_TypeExtSpec)
		next := n.Spec.(*ast.TypeDecl_TypeExtSpec)

		ots, nts = new(ast.TypeDecl_TypeSpec), new(ast.TypeDecl_TypeSpec)
		ots.TypeSpec, nts.TypeSpec = oext.TypeExtSpec.Type, next.TypeExtSpec.Type
		ots.TypeSpec.Doc, nts.TypeSpec.Doc = oext.TypeExtSpec.Doc, next.TypeExtSpec.Doc
	default:
		panic("compiler: unexpected merging of TypeSpec and TypeSpec")
	}

	// Merge doc groups
	ts.Doc.List = append(ts.Doc.List, ots.TypeSpec.Doc.List...)
	ts.Doc.List = append(ts.Doc.List, nts.TypeSpec.Doc.List...)

	// Merge type
	t := mergeExprs(ots.TypeSpec.Type, nts.TypeSpec.Type)
	switch v := t.(type) {
	case *ast.TypeSpec_Schema:
		ts.Type = v
	case *ast.TypeSpec_Scalar:
		ts.Type = v
	case *ast.TypeSpec_Object:
		ts.Type = v
	case *ast.TypeSpec_Interface:
		ts.Type = v
	case *ast.TypeSpec_Union:
		ts.Type = v
	case *ast.TypeSpec_Enum:
		ts.Type = v
	case *ast.TypeSpec_Input:
		ts.Type = v
	case *ast.TypeSpec_Directive:
		ts.Type = v
	}

	// Add name and directives
	ts.Name = ots.TypeSpec.Name
	ts.Directives = append(ts.Directives, ots.TypeSpec.Directives...)
	ts.Directives = append(ts.Directives, nts.TypeSpec.Directives...)

	return decl
}

func mergeExprs(o, n interface{}) (e interface{}) {
	switch u := o.(type) {
	case *ast.TypeSpec_Schema:
		v, ok := n.(*ast.TypeSpec_Schema)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.TypeSpec_Schema{
			Schema: &ast.SchemaType{
				RootOps: &ast.FieldList{
					List: append(u.Schema.RootOps.List, v.Schema.RootOps.List...),
				},
			},
		}
	case *ast.TypeSpec_Scalar:
		_, ok := n.(*ast.TypeSpec_Scalar)
		if !ok {
			panic("mismatched types")
		}

		e = u
	case *ast.TypeSpec_Object:
		v, ok := n.(*ast.TypeSpec_Object)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.TypeSpec_Object{
			Object: &ast.ObjectType{
				Interfaces: append(u.Object.Interfaces, v.Object.Interfaces...),
				Fields: &ast.FieldList{
					List: append(u.Object.Fields.List, v.Object.Fields.List...),
				},
			},
		}
	case *ast.TypeSpec_Interface:
		v, ok := n.(*ast.TypeSpec_Interface)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.TypeSpec_Interface{
			Interface: &ast.InterfaceType{
				Fields: &ast.FieldList{
					List: append(u.Interface.Fields.List, v.Interface.Fields.List...),
				},
			},
		}
	case *ast.TypeSpec_Enum:
		v, ok := n.(*ast.TypeSpec_Enum)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.TypeSpec_Enum{
			Enum: &ast.EnumType{
				Values: &ast.FieldList{
					List: append(u.Enum.Values.List, v.Enum.Values.List...),
				},
			},
		}
	case *ast.TypeSpec_Union:
		v, ok := n.(*ast.TypeSpec_Union)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.TypeSpec_Union{
			Union: &ast.UnionType{
				Members: append(u.Union.Members, v.Union.Members...),
			},
		}
	case *ast.TypeSpec_Input:
		v, ok := n.(*ast.TypeSpec_Input)
		if !ok {
			panic("mismatched types")
		}

		e = &ast.TypeSpec_Input{
			Input: &ast.InputType{
				Fields: &ast.FieldList{
					List: append(u.Input.Fields.List, v.Input.Fields.List...),
				},
			},
		}
	}
	return
}

const selExprTmpl = "%s.%s"

func addTypes(n *node, typeMap map[string]*ast.TypeDecl, add func(string, *ast.TypeDecl, map[string]*ast.TypeDecl) bool) (err error) {
	for _, tg := range n.Types {
		var ts *ast.TypeSpec
		switch v := tg.Spec.(type) {
		case *ast.TypeDecl_TypeSpec:
			ts = v.TypeSpec
		case *ast.TypeDecl_TypeExtSpec:
			ts = v.TypeExtSpec.Type
		default:
			panic("unknown spec")
		}

		name := ts.Name.Name
		if ts.Name == nil {
			name = "schema"
		}

		// Add type decl
		skip := add(name, tg, typeMap)
		if skip {
			continue
		}

		// Find any imported types contained within decl
		switch v := ts.Type.(type) {
		case *ast.TypeSpec_Scalar:
			// Scalar doesn't need anything done to it
		case *ast.TypeSpec_Object:
			for i := range v.Object.Interfaces {
				impl := v.Object.Interfaces[i]
				add(impl.Name, nil, typeMap)
			}

			err = resolveFieldList(n.Name, v.Object.Fields, typeMap, add)
			if err != nil {
				return
			}
		case *ast.TypeSpec_Interface:
			err = resolveFieldList(n.Name, v.Interface.Fields, typeMap, add)
			if err != nil {
				return
			}
		case *ast.TypeSpec_Union:
			for i := range v.Union.Members {
				mem := v.Union.Members[i]
				add(mem.Name, nil, typeMap)
			}
		case *ast.TypeSpec_Enum:
			// TODO: probably should resolve directives here
		case *ast.TypeSpec_Input:
			err = resolveFieldList(n.Name, v.Input.Fields, typeMap, add)
			if err != nil {
				return err
			}
		case *ast.TypeSpec_Directive:
			err = resolveFieldList(n.Name, v.Directive.Args, typeMap, add)
			if err != nil {
				return err
			}
		}
	}

	return
}

func resolveFieldList(name string, fields *ast.FieldList, typeMap map[string]*ast.TypeDecl, add func(string, *ast.TypeDecl, map[string]*ast.TypeDecl) bool) (err error) {
	if fields == nil {
		return
	}

	for _, f := range fields.List {
		err = resolveFieldList(name, f.Args, typeMap, add)
		if err != nil {
			return
		}

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
		add(t.Name, nil, typeMap)
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

func isBuiltinType(name string) bool {
	tname := strings.ToLower(name)
	return tname == "id" || tname == "boolean" || tname == "int" || tname == "string" || tname == "float"
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
			return unwrapType(v.Type)
		case *ast.List_NonNull:
			return unwrapType(v.Type)
		}
	case *ast.NonNull:
		switch u := v.Type.(type) {
		case *ast.NonNull_Ident:
			return u.Ident
		case *ast.NonNull_List:
			return unwrapType(v.Type)
		}
	}

	return nil
}
