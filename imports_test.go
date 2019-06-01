package compiler

import (
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
	"io"
	"strings"
	"testing"
)

func TestCreateImportTries(t *testing.T) {
	// Create test docs
	testData := []struct {
		Name  string
		Value *ast.Arg_CompositeLit
	}{
		{
			Name: "a",
			Value: &ast.Arg_CompositeLit{
				CompositeLit: &ast.CompositeLit{
					Value: &ast.CompositeLit_ListLit{
						ListLit: &ast.ListLit{
							List: &ast.ListLit_BasicList{
								BasicList: &ast.ListLit_Basic{
									Values: []*ast.BasicLit{
										{Value: "e"},
										{Value: "f"},
									},
								},
							},
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
			Value: &ast.Arg_CompositeLit{
				CompositeLit: &ast.CompositeLit{
					Value: &ast.CompositeLit_ListLit{
						ListLit: &ast.ListLit{
							List: &ast.ListLit_BasicList{
								BasicList: &ast.ListLit_Basic{
									Values: []*ast.BasicLit{
										{Value: "g"},
										{Value: "h"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "d",
			Value: &ast.Arg_CompositeLit{
				CompositeLit: &ast.CompositeLit{
					Value: &ast.CompositeLit_ListLit{
						ListLit: &ast.ListLit{
							List: &ast.ListLit_BasicList{
								BasicList: &ast.ListLit_Basic{
									Values: []*ast.BasicLit{
										{Value: "i"},
										{Value: "j"},
									},
								},
							},
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
			Value: &ast.Arg_CompositeLit{
				CompositeLit: &ast.CompositeLit{
					Value: &ast.CompositeLit_ListLit{
						ListLit: &ast.ListLit{
							List: &ast.ListLit_BasicList{
								BasicList: &ast.ListLit_Basic{
									Values: []*ast.BasicLit{
										{Value: "e"},
										{Value: "f"},
									},
								},
							},
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
			Value: &ast.Arg_CompositeLit{
				CompositeLit: &ast.CompositeLit{
					Value: &ast.CompositeLit_ListLit{
						ListLit: &ast.ListLit{
							List: &ast.ListLit_BasicList{
								BasicList: &ast.ListLit_Basic{
									Values: []*ast.BasicLit{
										{Value: "e"},
										{Value: "h"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "j",
			Value: &ast.Arg_CompositeLit{
				CompositeLit: &ast.CompositeLit{
					Value: &ast.CompositeLit_ListLit{
						ListLit: &ast.ListLit{
							List: &ast.ListLit_BasicList{
								BasicList: &ast.ListLit_Basic{
									Values: []*ast.BasicLit{
										{Value: "h"},
										{Value: "f"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	docs := make([]*ast.Document, len(testData))
	for i, data := range testData {
		docs[i] = &ast.Document{
			Name: data.Name,
		}
		if data.Value == nil {
			continue
		}

		docs[i].Directives = []*ast.DirectiveLit{
			{
				Name: "import",
				Args: &ast.CallExpr{
					Args: []*ast.Arg{{Value: data.Value}},
				},
			},
		}
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

	for _, d := range forest {
		if len(d.Directives) > 0 {
			t.Fail()
			return
		}
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
	// Create test docs
	testData := []struct {
		Name  string
		Value *ast.Arg_CompositeLit
	}{
		{
			Name: "a",
			Value: &ast.Arg_CompositeLit{
				CompositeLit: &ast.CompositeLit{
					Value: &ast.CompositeLit_ListLit{
						ListLit: &ast.ListLit{
							List: &ast.ListLit_BasicList{
								BasicList: &ast.ListLit_Basic{
									Values: []*ast.BasicLit{
										{Value: "b"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "b",
			Value: &ast.Arg_CompositeLit{
				CompositeLit: &ast.CompositeLit{
					Value: &ast.CompositeLit_ListLit{
						ListLit: &ast.ListLit{
							List: &ast.ListLit_BasicList{
								BasicList: &ast.ListLit_Basic{
									Values: []*ast.BasicLit{
										{Value: "a"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	docs := make([]*ast.Document, len(testData))
	for i, data := range testData {
		docs[i] = &ast.Document{
			Name: data.Name,
			Directives: []*ast.DirectiveLit{
				{
					Name: "import",
					Args: &ast.CallExpr{
						Args: []*ast.Arg{{Value: data.Value}},
					},
				},
			},
		}
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
		Name     string
		Old, New *ast.TypeDecl
		check    func(*ast.TypeDecl) bool
	}{
		// Ext -> Spec
		{
			Name: "Ext->Spec:schema",
			New: &ast.TypeDecl{
				Spec: &ast.TypeDecl_TypeSpec{
					TypeSpec: &ast.TypeSpec{
						Name: &ast.Ident{Name: "test"},
						Type: &ast.TypeSpec_Schema{Schema: &ast.SchemaType{RootOps: &ast.FieldList{List: make([]*ast.Field, 10)}}},
						Doc:  new(ast.DocGroup),
					},
				},
			},
			Old: &ast.TypeDecl{
				Spec: &ast.TypeDecl_TypeExtSpec{
					TypeExtSpec: &ast.TypeExtensionSpec{
						Type: &ast.TypeSpec{
							Type:       &ast.TypeSpec_Schema{Schema: &ast.SchemaType{RootOps: &ast.FieldList{List: make([]*ast.Field, 20)}}},
							Directives: make([]*ast.DirectiveLit, 10),
						},
						Doc: new(ast.DocGroup),
					},
				},
			},
			check: func(d *ast.TypeDecl) bool {
				ts, ok := d.Spec.(*ast.TypeDecl_TypeSpec)
				if !ok {
					return false
				}

				if len(ts.TypeSpec.Directives) != 10 {
					return false
				}

				if len(ts.TypeSpec.Type.(*ast.TypeSpec_Schema).Schema.RootOps.List) != 30 {
					return false
				}

				return true
			},
		},
		{
			Name: "Ext->Spec:scalar",
			New: &ast.TypeDecl{
				Spec: &ast.TypeDecl_TypeSpec{
					TypeSpec: &ast.TypeSpec{
						Name: &ast.Ident{Name: "Test"},
						Type: &ast.TypeSpec_Scalar{Scalar: &ast.ScalarType{Name: &ast.Ident{Name: "Test"}}},
						Doc:  new(ast.DocGroup),
					},
				},
			},
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Type: &ast.TypeSpec{
					Type:       &ast.TypeSpec_Scalar{Scalar: &ast.ScalarType{}},
					Directives: make([]*ast.DirectiveLit, 10),
				},
				Doc: new(ast.DocGroup),
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				ts, ok := d.Spec.(*ast.TypeDecl_TypeSpec)
				if !ok {
					return false
				}

				return len(ts.TypeSpec.Directives) == 10
			},
		},
		{
			Name: "Ext->Spec:object",
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeSpec{TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.TypeSpec_Object{Object: &ast.ObjectType{
					Interfaces: make([]*ast.Ident, 10),
					Fields:     &ast.FieldList{List: make([]*ast.Field, 10)},
				},
				},
				Directives: make([]*ast.DirectiveLit, 5),
			},
			},
			},
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Type: &ast.TypeSpec_Object{Object: &ast.ObjectType{
						Interfaces: make([]*ast.Ident, 10),
						Fields:     &ast.FieldList{List: make([]*ast.Field, 10)},
					},
					},
					Directives: make([]*ast.DirectiveLit, 5),
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				ts, ok := d.Spec.(*ast.TypeDecl_TypeSpec)
				if !ok {
					return false
				}

				if len(ts.TypeSpec.Directives) != 10 {
					return false
				}

				obj := ts.TypeSpec.Type.(*ast.TypeSpec_Object)
				if len(obj.Object.Interfaces) != 20 {
					return false
				}
				if len(obj.Object.Fields.List) != 20 {
					return false
				}
				return true
			},
		},
		{
			Name: "Ext->Spec:interface",
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeSpec{TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.TypeSpec_Interface{Interface: &ast.InterfaceType{
					Fields: &ast.FieldList{List: make([]*ast.Field, 10)},
				}},
				Directives: make([]*ast.DirectiveLit, 5),
			},
			},
			},
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Interface{Interface: &ast.InterfaceType{
						Fields: &ast.FieldList{List: make([]*ast.Field, 10)},
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				ts, ok := d.Spec.(*ast.TypeDecl_TypeSpec)
				if !ok {
					return false
				}

				if len(ts.TypeSpec.Directives) != 10 {
					return false
				}

				obj := ts.TypeSpec.Type.(*ast.TypeSpec_Interface)
				return len(obj.Interface.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Spec:enum",
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeSpec{TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.TypeSpec_Enum{Enum: &ast.EnumType{
					Values: &ast.FieldList{List: make([]*ast.Field, 10)},
				}},
				Directives: make([]*ast.DirectiveLit, 5),
			},
			},
			},
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Enum{Enum: &ast.EnumType{
						Values: &ast.FieldList{List: make([]*ast.Field, 10)},
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				ts, ok := d.Spec.(*ast.TypeDecl_TypeSpec)
				if !ok {
					return false
				}

				if len(ts.TypeSpec.Directives) != 10 {
					return false
				}

				obj := ts.TypeSpec.Type.(*ast.TypeSpec_Enum)
				return len(obj.Enum.Values.List) == 20
			},
		},
		{
			Name: "Ext->Spec:union",
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeSpec{TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.TypeSpec_Union{Union: &ast.UnionType{
					Members: make([]*ast.Ident, 10),
				}},
				Directives: make([]*ast.DirectiveLit, 5),
			},
			},
			},
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Union{Union: &ast.UnionType{
						Members: make([]*ast.Ident, 10),
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				ts, ok := d.Spec.(*ast.TypeDecl_TypeSpec)
				if !ok {
					return false
				}

				if len(ts.TypeSpec.Directives) != 10 {
					return false
				}

				obj := ts.TypeSpec.Type.(*ast.TypeSpec_Union)
				return len(obj.Union.Members) == 20
			},
		},
		{
			Name: "Ext->Spec:input",
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeSpec{TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Test"},
				Doc:  new(ast.DocGroup),
				Type: &ast.TypeSpec_Input{Input: &ast.InputType{
					Fields: &ast.InputValueList{List: make([]*ast.InputValue, 10)},
				}},
				Directives: make([]*ast.DirectiveLit, 5),
			},
			},
			},
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Input{Input: &ast.InputType{
						Fields: &ast.InputValueList{List: make([]*ast.InputValue, 10)},
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				ts, ok := d.Spec.(*ast.TypeDecl_TypeSpec)
				if !ok {
					return false
				}

				if len(ts.TypeSpec.Directives) != 10 {
					return false
				}

				obj := ts.TypeSpec.Type.(*ast.TypeSpec_Input)
				return len(obj.Input.Fields.List) == 20
			},
		},

		// Ext -> Ext
		{
			Name: "Ext->Ext:schema",
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Schema{Schema: &ast.SchemaType{
						RootOps: &ast.FieldList{List: make([]*ast.Field, 10)},
					}},
				},
			},
			},
			},
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Schema{Schema: &ast.SchemaType{
						RootOps: &ast.FieldList{List: make([]*ast.Field, 10)},
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				text, ok := d.Spec.(*ast.TypeDecl_TypeExtSpec)
				if !ok {
					return false
				}

				ts := text.TypeExtSpec.Type
				if len(ts.Directives) != 10 {
					return false
				}

				v := ts.Type.(*ast.TypeSpec_Schema)
				return len(v.Schema.RootOps.List) == 20
			},
		},
		{
			Name: "Ext->Ext:scalar",
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type:       &ast.TypeSpec_Scalar{Scalar: &ast.ScalarType{}},
				},
			},
			},
			},
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type:       &ast.TypeSpec_Scalar{Scalar: &ast.ScalarType{}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				text, ok := d.Spec.(*ast.TypeDecl_TypeExtSpec)
				if !ok {
					return false
				}

				ts := text.TypeExtSpec.Type
				return len(ts.Directives) == 10
			},
		},
		{
			Name: "Ext->Ext:object",
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Object{Object: &ast.ObjectType{
						Interfaces: make([]*ast.Ident, 10),
						Fields:     &ast.FieldList{List: make([]*ast.Field, 10)},
					},
					},
				},
			},
			},
			},
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Object{Object: &ast.ObjectType{
						Interfaces: make([]*ast.Ident, 10),
						Fields:     &ast.FieldList{List: make([]*ast.Field, 10)},
					},
					},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				text, ok := d.Spec.(*ast.TypeDecl_TypeExtSpec)
				if !ok {
					return false
				}

				ts := text.TypeExtSpec.Type
				if len(ts.Directives) != 10 {
					return false
				}

				v := ts.Type.(*ast.TypeSpec_Object)
				if len(v.Object.Interfaces) != 20 {
					return false
				}
				return len(v.Object.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Ext:interface",
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Interface{Interface: &ast.InterfaceType{
						Fields: &ast.FieldList{List: make([]*ast.Field, 10)},
					}},
				},
			},
			},
			},
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Interface{Interface: &ast.InterfaceType{
						Fields: &ast.FieldList{List: make([]*ast.Field, 10)},
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				text, ok := d.Spec.(*ast.TypeDecl_TypeExtSpec)
				if !ok {
					return false
				}

				ts := text.TypeExtSpec.Type
				if len(ts.Directives) != 10 {
					return false
				}

				v := ts.Type.(*ast.TypeSpec_Interface)
				return len(v.Interface.Fields.List) == 20
			},
		},
		{
			Name: "Ext->Ext:enum",
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Enum{Enum: &ast.EnumType{
						Values: &ast.FieldList{List: make([]*ast.Field, 10)},
					}},
				},
			},
			},
			},
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Enum{Enum: &ast.EnumType{
						Values: &ast.FieldList{List: make([]*ast.Field, 10)},
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				text, ok := d.Spec.(*ast.TypeDecl_TypeExtSpec)
				if !ok {
					return false
				}

				ts := text.TypeExtSpec.Type
				if len(ts.Directives) != 10 {
					return false
				}

				v := ts.Type.(*ast.TypeSpec_Enum)
				return len(v.Enum.Values.List) == 20
			},
		},
		{
			Name: "Ext->Ext:union",
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Union{Union: &ast.UnionType{
						Members: make([]*ast.Ident, 10),
					}},
				},
			},
			},
			},
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Union{Union: &ast.UnionType{
						Members: make([]*ast.Ident, 10),
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				text, ok := d.Spec.(*ast.TypeDecl_TypeExtSpec)
				if !ok {
					return false
				}

				ts := text.TypeExtSpec.Type
				if len(ts.Directives) != 10 {
					return false
				}

				v := ts.Type.(*ast.TypeSpec_Union)
				return len(v.Union.Members) == 20
			},
		},
		{
			Name: "Ext->Ext:input",
			Old: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Input{Input: &ast.InputType{
						Fields: &ast.InputValueList{List: make([]*ast.InputValue, 10)},
					}},
				},
			},
			},
			},
			New: &ast.TypeDecl{Spec: &ast.TypeDecl_TypeExtSpec{TypeExtSpec: &ast.TypeExtensionSpec{
				Doc: new(ast.DocGroup),
				Type: &ast.TypeSpec{
					Directives: make([]*ast.DirectiveLit, 5),
					Type: &ast.TypeSpec_Input{Input: &ast.InputType{
						Fields: &ast.InputValueList{List: make([]*ast.InputValue, 10)},
					}},
				},
			},
			},
			},
			check: func(d *ast.TypeDecl) bool {
				text, ok := d.Spec.(*ast.TypeDecl_TypeExtSpec)
				if !ok {
					return false
				}

				ts := text.TypeExtSpec.Type
				if len(ts.Directives) != 10 {
					return false
				}

				v := ts.Type.(*ast.TypeSpec_Input)
				return len(v.Input.Fields.List) == 20
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(subT *testing.T) {
			if testCase.check == nil {
				subT.Skip()
				return
			}

			t := mergeTypes(testCase.Old, testCase.New)
			if !testCase.check(t) {
				subT.Fail()
				return
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

	typeMap := make(map[string]*ast.TypeDecl)
	err = addTypes(&node{Document: doc}, typeMap, func(name string, decl *ast.TypeDecl, decls map[string]*ast.TypeDecl) bool {
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
		t.Error(err)
		return
	}

	if len(typeMap) != 5 {
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
	if len(forest[0].Types) != 5 {
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

	apiGQL = `@import(paths: ["graph"])

interface TestNode {
	id: ID!
}

type User implements Node & TestNode {
	id: ID!
	name: String
}

type UserConnection implements Connection {
	total: Int
	edges: [Node]
	hasNextPage: Boolean
}`

	oneGql = `@import(paths: ["two", "six", "four"])

type Service implements Doc {
	t: Time
	obj: Obj
}`
	twoGql = `@import(paths: ["thr"])

interface Doc {
	v: Version
}`
	thrGql  = `scalar Version`
	fourGql = `scalar Time`
	fiveGql = `@import(paths: ["six", "two"])

type T implements Doc {
	v: Version
	obj: Obj
}`
	sixGql = `@import(paths: ["thr"])

type Obj {
	v: Version
}`
)

func TestReduceImports(t *testing.T) {
	testCases := []struct {
		Name     string
		DocsLen  int
		TypesLen map[string]int
		Docs     map[string]io.Reader
	}{
		{
			Name:     "Graphy",
			DocsLen:  1,
			TypesLen: map[string]int{"test": 5},
			Docs:     map[string]io.Reader{"graph": strings.NewReader(graphGQL), "test": strings.NewReader(apiGQL)},
		},
		{
			Name:     "NoImports",
			DocsLen:  1,
			TypesLen: map[string]int{"test": 1},
			Docs:     map[string]io.Reader{"test": strings.NewReader(thrGql)},
		},
		{
			Name:     "SingleImport",
			DocsLen:  1,
			TypesLen: map[string]int{"two": 2},
			Docs:     map[string]io.Reader{"two": strings.NewReader(twoGql), "thr": strings.NewReader(thrGql)},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(subT *testing.T) {
			docs, err := parser.ParseDocs(token.NewDocSet(), testCase.Docs, 0)
			if err != nil {
				subT.Error(err)
				return
			}

			docs, err = ReduceImports(docs)
			if err != nil {
				subT.Error(err)
				return
			}

			if len(docs) != testCase.DocsLen {
				t.Fail()
				return
			}

			for _, doc := range docs {
				if len(doc.Types) != testCase.TypesLen[doc.Name] {
					t.Fail()
					return
				}

				for _, tg := range doc.Types {
					if tg == nil {
						t.Fail()
						return
					}
				}
			}
		})
	}
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

	baseGQL = `@import(paths: ["graph"])

type User {
	id: ID!
	name: String
}

type UserConnection implements Connection {}`
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
