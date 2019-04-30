package compiler

import (
	"fmt"
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
	"io"
	"strings"
	"testing"
)

func TestCreateImportTries(t *testing.T) {
	// Create test docs
	docs := []*ast.Document{
		{
			Name: "a",
			Imports: []*ast.ImportDecl{
				{
					Specs: []*ast.ImportSpec{
						{
							Name: &ast.Ident{Name: "e"},
						},
						{
							Name: &ast.Ident{Name: "f"},
						},
					},
				},
			},
		},
		{
			Name: "b",
		},
		{
			Name: "c",
			Imports: []*ast.ImportDecl{
				{
					Specs: []*ast.ImportSpec{
						{
							Name: &ast.Ident{Name: "g"},
						},
						{
							Name: &ast.Ident{Name: "h"},
						},
					},
				},
			},
		},
		{
			Name: "d",
			Imports: []*ast.ImportDecl{
				{
					Specs: []*ast.ImportSpec{
						{
							Name: &ast.Ident{Name: "i"},
						},
						{
							Name: &ast.Ident{Name: "j"},
						},
					},
				},
			},
		},
		{
			Name: "e",
		},
		{
			Name: "f",
		},
		{
			Name: "g",
			Imports: []*ast.ImportDecl{
				{
					Specs: []*ast.ImportSpec{
						{
							Name: &ast.Ident{Name: "e"},
						},
						{
							Name: &ast.Ident{Name: "f"},
						},
					},
				},
			},
		},
		{
			Name: "h",
		},
		{
			Name: "i",
			Imports: []*ast.ImportDecl{
				{
					Specs: []*ast.ImportSpec{
						{
							Name: &ast.Ident{Name: "e"},
						},
						{
							Name: &ast.Ident{Name: "h"},
						},
					},
				},
			},
		},
		{
			Name: "j",
			Imports: []*ast.ImportDecl{
				{
					Specs: []*ast.ImportSpec{
						{
							Name: &ast.Ident{Name: "h"},
						},
						{
							Name: &ast.Ident{Name: "f"},
						},
					},
				},
			},
		},
	}

	// Initialize nodes and dMap for createImportTries
	dMap := make(map[string]*node, len(docs))
	nodes := make([]*node, len(docs))
	for i, doc := range docs {
		n := &node{Document: doc}
		dMap[doc.Name] = n
		nodes[i] = n
	}

	// Create Import tries
	forest, err := createImportTries(nodes, dMap)
	if err != nil {
		t.Errorf("unexpected error when creating import forest: %s", err)
		return
	}

	// A forest of 4 should be the minimal reduction achieved
	if len(forest) != 4 {
		t.Fail()
		return
	}

	// Now walk trie and verify its structure
	var walk func(trie *node, lvl int, checker func(lvl int, n *node) bool) bool
	walk = func(trie *node, lvl int, checker func(lvl int, n *node) bool) (ok bool) {
		// Top down walk
		ok = checker(lvl, trie)
		if !ok {
			return
		}

		for _, c := range trie.Childs {
			ok = walk(c, lvl+1, checker)
			if !ok {
				break
			}
		}
		return
	}

	// Walk tries
	trieLvls := []map[string]int{
		{"a": 0, "e": 1, "f": 1},
		{"b": 0},
		{"c": 0, "g": 1, "h": 1, "e": 2, "f": 2},
		{"d": 0, "i": 1, "j": 1, "e": 2, "h": 2, "f": 2},
	}
	for i, lvls := range trieLvls {
		fmt.Println(i, lvls)
		ok := walk(forest[i], 0, func(lvl int, n *node) bool {
			nlvl, exists := lvls[n.Name]
			if exists && nlvl != lvl {
				return false
			}
			delete(lvls, n.Name)
			return true
		})

		if !ok {
			t.Fail()
			return
		}
		if len(lvls) != 0 {
			t.Fail()
			return
		}
	}
}

func TestImportCycle(t *testing.T) {
	docs := []*ast.Document{
		{
			Name: "a",
			Imports: []*ast.ImportDecl{
				{Specs: []*ast.ImportSpec{
					{
						Name: &ast.Ident{Name: "b"},
					},
				},
				},
			},
		},
		{
			Name: "b",
			Imports: []*ast.ImportDecl{
				{
					Specs: []*ast.ImportSpec{
						{
							Name: &ast.Ident{Name: "a"},
						},
					},
				},
			},
		},
	}

	// Initialize nodes and dMap for createImportTries
	dMap := make(map[string]*node, len(docs))
	nodes := make([]*node, len(docs))
	for i, doc := range docs {
		n := &node{Document: doc}
		dMap[doc.Name] = n
		nodes[i] = n
	}

	// Create Import tries
	_, err := createImportTries(nodes, dMap)
	if err == nil {
		t.Fail()
		return
	}

}

func TestMergeTypes(t *testing.T) {
	testCases := []struct {
		Name  string
		Old   ast.Spec
		New   ast.Spec
		check func(*ast.GenDecl) bool
	}{
		// Ext -> Spec
		{
			Name: "Ext->Spec:schema",
			New: &ast.TypeSpec{
				Name: &ast.Ident{Name: "test"},
				Type: &ast.SchemaType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				Doc:  new(ast.DocGroup),
			},
			Old: &ast.TypeExtensionSpec{
				Type: &ast.TypeSpec{
					Type: &ast.SchemaType{Fields: &ast.FieldList{List: make([]*ast.Field, 20)}},
					Dirs: make([]*ast.DirectiveLit, 10),
				},
				Doc: new(ast.DocGroup),
			},
			check: func(d *ast.GenDecl) bool {
				ts, ok := d.Spec.(*ast.TypeSpec)
				if !ok {
					return false
				}

				if len(ts.Dirs) != 10 {
					return false
				}

				if len(ts.Type.(*ast.SchemaType).Fields.List) != 30 {
					return false
				}

				return true
			},
		},
		{
			Name: "Ext->Spec:scalar",
			New: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Type: &ast.ScalarType{Name: &ast.Ident{Name: "Test"}},
				Doc:  new(ast.DocGroup),
			},
			Old: &ast.TypeExtensionSpec{
				Type: &ast.TypeSpec{
					Type: &ast.ScalarType{},
					Dirs: make([]*ast.DirectiveLit, 10),
				},
				Doc: new(ast.DocGroup),
			},
			check: func(d *ast.GenDecl) bool {
				ts, ok := d.Spec.(*ast.TypeSpec)
				if !ok {
					return false
				}

				return len(ts.Dirs) == 10
			},
		},
		{
			Name: "Ext->Spec:object",
			New: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.ObjectType{
					Impls:  make([]ast.Expr, 10),
					Fields: &ast.FieldList{List: make([]*ast.Field, 10)},
				},
				Dirs: make([]*ast.DirectiveLit, 5),
			},
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Type: &ast.ObjectType{
						Impls:  make([]ast.Expr, 10),
						Fields: &ast.FieldList{List: make([]*ast.Field, 10)},
					},
					Dirs: make([]*ast.DirectiveLit, 5),
				},
			},
			check: func(d *ast.GenDecl) bool {
				ts, ok := d.Spec.(*ast.TypeSpec)
				if !ok {
					return false
				}

				if len(ts.Dirs) != 10 {
					return false
				}

				obj := ts.Type.(*ast.ObjectType)
				if len(obj.Impls) != 20 {
					return false
				}
				if len(obj.Fields.List) != 20 {
					return false
				}
				return true
			},
		},
		{
			Name: "Ext->Spec:interface",
			New: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.InterfaceType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				Dirs: make([]*ast.DirectiveLit, 5),
			},
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.InterfaceType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			check: func(d *ast.GenDecl) bool {
				ts, ok := d.Spec.(*ast.TypeSpec)
				if !ok {
					return false
				}

				if len(ts.Dirs) != 10 {
					return false
				}

				obj := ts.Type.(*ast.InterfaceType)
				return len(obj.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Spec:enum",
			New: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.EnumType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				Dirs: make([]*ast.DirectiveLit, 5),
			},
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.EnumType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			check: func(d *ast.GenDecl) bool {
				ts, ok := d.Spec.(*ast.TypeSpec)
				if !ok {
					return false
				}

				if len(ts.Dirs) != 10 {
					return false
				}

				obj := ts.Type.(*ast.EnumType)
				return len(obj.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Spec:union",
			New: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.UnionType{Members: make([]ast.Expr, 10)},
				Dirs: make([]*ast.DirectiveLit, 5),
			},
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.UnionType{Members: make([]ast.Expr, 10)},
				},
			},
			check: func(d *ast.GenDecl) bool {
				ts, ok := d.Spec.(*ast.TypeSpec)
				if !ok {
					return false
				}

				if len(ts.Dirs) != 10 {
					return false
				}

				obj := ts.Type.(*ast.UnionType)
				return len(obj.Members) == 20
			},
		},
		{
			Name: "Ext->Spec:input",
			New: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.InputType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				Dirs: make([]*ast.DirectiveLit, 5),
			},
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.InputType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			check: func(d *ast.GenDecl) bool {
				ts, ok := d.Spec.(*ast.TypeSpec)
				if !ok {
					return false
				}

				if len(ts.Dirs) != 10 {
					return false
				}

				obj := ts.Type.(*ast.InputType)
				return len(obj.Fields.List) == 20
			},
		},

		// Ext -> Ext
		{
			Name: "Ext->Ext:schema",
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.SchemaType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			New: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.SchemaType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			check: func(d *ast.GenDecl) bool {
				text, ok := d.Spec.(*ast.TypeExtensionSpec)
				if !ok {
					return false
				}

				ts := text.Type
				if len(ts.Dirs) != 10 {
					return false
				}

				v := ts.Type.(*ast.SchemaType)
				return len(v.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Ext:scalar",
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.ScalarType{},
				},
			},
			New: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.ScalarType{},
				},
			},
			check: func(d *ast.GenDecl) bool {
				text, ok := d.Spec.(*ast.TypeExtensionSpec)
				if !ok {
					return false
				}

				ts := text.Type
				return len(ts.Dirs) == 10
			},
		},
		{
			Name: "Ext->Ext:object",
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.ObjectType{
						Impls:  make([]ast.Expr, 10),
						Fields: &ast.FieldList{List: make([]*ast.Field, 10)},
					},
				},
			},
			New: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.ObjectType{
						Impls:  make([]ast.Expr, 10),
						Fields: &ast.FieldList{List: make([]*ast.Field, 10)},
					},
				},
			},
			check: func(d *ast.GenDecl) bool {
				text, ok := d.Spec.(*ast.TypeExtensionSpec)
				if !ok {
					return false
				}

				ts := text.Type
				if len(ts.Dirs) != 10 {
					return false
				}

				v := ts.Type.(*ast.ObjectType)
				if len(v.Impls) != 20 {
					return false
				}
				return len(v.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Ext:interface",
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.InterfaceType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			New: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.InterfaceType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			check: func(d *ast.GenDecl) bool {
				text, ok := d.Spec.(*ast.TypeExtensionSpec)
				if !ok {
					return false
				}

				ts := text.Type
				if len(ts.Dirs) != 10 {
					return false
				}

				v := ts.Type.(*ast.InterfaceType)
				return len(v.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Ext:enum",
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.EnumType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			New: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.EnumType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			check: func(d *ast.GenDecl) bool {
				text, ok := d.Spec.(*ast.TypeExtensionSpec)
				if !ok {
					return false
				}

				ts := text.Type
				if len(ts.Dirs) != 10 {
					return false
				}

				v := ts.Type.(*ast.EnumType)
				return len(v.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Ext:union",
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.UnionType{Members: make([]ast.Expr, 10)},
				},
			},
			New: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.UnionType{Members: make([]ast.Expr, 10)},
				},
			},
			check: func(d *ast.GenDecl) bool {
				text, ok := d.Spec.(*ast.TypeExtensionSpec)
				if !ok {
					return false
				}

				ts := text.Type
				if len(ts.Dirs) != 10 {
					return false
				}

				v := ts.Type.(*ast.UnionType)
				return len(v.Members) == 20
			},
		},
		{
			Name: "Ext->Ext:input",
			Old: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.InputType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			New: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Dirs: make([]*ast.DirectiveLit, 5),
					Type: &ast.InputType{Fields: &ast.FieldList{List: make([]*ast.Field, 10)}},
				},
			},
			check: func(d *ast.GenDecl) bool {
				text, ok := d.Spec.(*ast.TypeExtensionSpec)
				if !ok {
					return false
				}

				ts := text.Type
				if len(ts.Dirs) != 10 {
					return false
				}

				v := ts.Type.(*ast.InputType)
				return len(v.Fields.List) == 20
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(subT *testing.T) {
			if testCase.check == nil {
				subT.Skip()
				return
			}

			t := mergeTypes(&ast.GenDecl{Spec: testCase.Old}, &ast.GenDecl{Spec: testCase.New})
			if !testCase.check(t) {
				subT.Fail()
			}
		})
	}
}

func TestAddTypes(t *testing.T) {
	doc, err := parser.ParseDoc(token.NewDocSet(), "api", strings.NewReader(apiGQL), 0)
	if err != nil {
		t.Error(err)
		return
	}

	typeMap := make(map[string]*ast.GenDecl)
	err = addTypes(&node{Document: doc}, typeMap, func(name string, decl *ast.GenDecl, decls map[string]*ast.GenDecl) bool {
		if isBuiltinType(name) {
			return true
		}

		decls[name] = decl
		return false
	})
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(typeMap)
	if len(typeMap) != 4 {
		t.Fail()
	}
}

func TestResolveImports(t *testing.T) {
	docs, err := parser.ParseDocs(token.NewDocSet(), map[string]io.Reader{"graph": strings.NewReader(graphGQL), "api": strings.NewReader(apiGQL)}, 0)
	if err != nil {
		t.Error(err)
		return
	}

	// Initialize nodes and dMap for createImportTries
	dMap := make(map[string]*node, len(docs))
	nodes := make([]*node, len(docs))
	for i, doc := range docs {
		n := &node{Document: doc}
		dMap[doc.Name] = n
		nodes[i] = n
	}

	// Create Import tries
	forest, err := createImportTries(nodes, dMap)
	if err != nil {
		t.Errorf("unexpected error when creating import forest: %s", err)
		return
	}

	err = resolveImports(forest[0])
	if err != nil {
		t.Error(err)
		return
	}
	if len(forest[0].Types) != 4 {
		t.Fail()
		return
	}

	for _, tg := range forest[0].Types {
		if tg == nil {
			t.Fail()
			return
		}
	}
}

var (
	graphGQL = `interface Node {
	id: ID!
}

interface Connection {
	total: Int
	edges: [Node]
	hasNextPage: Boolean
}`

	apiGQL = `import "graph"

type User implements graph.Node {
	id: ID!
	name: String
}

type UserConnection implements graph.Connection {
	total: Int
	edges: [graph.Node]
	hasNextPage: Boolean
}`
)

func TestReduceImports(t *testing.T) {
	docs, err := parser.ParseDocs(token.NewDocSet(), map[string]io.Reader{"graph": strings.NewReader(graphGQL), "api": strings.NewReader(apiGQL)}, 0)
	if err != nil {
		t.Error(err)
		return
	}

	docs, err = ReduceImports(docs)
	if err != nil {
		t.Error(err)
		return
	}

	if len(docs) != 1 {
		t.Fail()
		return
	}

	if len(docs[0].Types) != 4 {
		t.Fail()
		return
	}

	for _, tg := range docs[0].Types {
		if tg == nil {
			t.Fail()
			return
		}
	}
}

var (
	impGQL = `interface Node {
	id: ID!
}

interface Connection {
	total: Int
	edges: [Node]
	hasNextPage: Boolean
}`

	baseGQL = `import "graph"

type User {
	id: ID!
	name: String
}

type UserConnection implements graph.Connection {}`
)

func TestPeerTypes(t *testing.T) {
	docs, err := parser.ParseDocs(token.NewDocSet(), map[string]io.Reader{"graph": strings.NewReader(impGQL), "base": strings.NewReader(baseGQL)}, 0)
	if err != nil {
		t.Error(err)
		return
	}

	docs, err = ReduceImports(docs)
	if err != nil {
		t.Error(err)
		return
	}

	if len(docs) != 1 {
		t.Fail()
		return
	}

	if len(docs[0].Types) != 4 {
		t.Fail()
		return
	}

	for _, tg := range docs[0].Types {
		if tg == nil {
			t.Fail()
			return
		}
	}
}
