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
							List: &ast.ListLit_CompositeList{
								CompositeList: &ast.ListLit_Composite{
									Values: []*ast.CompositeLit{
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "e"}}},
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "f"}}},
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
							List: &ast.ListLit_CompositeList{
								CompositeList: &ast.ListLit_Composite{
									Values: []*ast.CompositeLit{
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "g"}}},
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "h"}}},
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
							List: &ast.ListLit_CompositeList{
								CompositeList: &ast.ListLit_Composite{
									Values: []*ast.CompositeLit{
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "i"}}},
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "j"}}},
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
							List: &ast.ListLit_CompositeList{
								CompositeList: &ast.ListLit_Composite{
									Values: []*ast.CompositeLit{
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "e"}}},
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "f"}}},
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
							List: &ast.ListLit_CompositeList{
								CompositeList: &ast.ListLit_Composite{
									Values: []*ast.CompositeLit{
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "e"}}},
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "h"}}},
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
							List: &ast.ListLit_CompositeList{
								CompositeList: &ast.ListLit_Composite{
									Values: []*ast.CompositeLit{
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "h"}}},
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "f"}}},
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
							List: &ast.ListLit_CompositeList{
								CompositeList: &ast.ListLit_Composite{
									Values: []*ast.CompositeLit{
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "b"}}},
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
							List: &ast.ListLit_CompositeList{
								CompositeList: &ast.ListLit_Composite{
									Values: []*ast.CompositeLit{
										{Value: &ast.CompositeLit_BasicLit{BasicLit: &ast.BasicLit{Value: "a"}}},
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
		n := &node{Document: doc, Types: toDeclMap(doc.Types)}
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
		{
			Name:     "DiamondImport",
			DocsLen:  1,
			TypesLen: map[string]int{"five": 4},
			Docs:     map[string]io.Reader{"five": strings.NewReader(fiveGql), "two": strings.NewReader(twoGql), "six": strings.NewReader(sixGql), "thr": strings.NewReader(thrGql)},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(subT *testing.T) {
			docs, err := parser.ParseDocs(token.NewDocSet(), testCase.Docs, 0)
			if err != nil {
				subT.Error(err)
				return
			}

			docsIR, err := ReduceImports(docs)
			if err != nil {
				subT.Error(err)
				return
			}

			if len(docsIR) != testCase.DocsLen {
				subT.Fail()
				return
			}

			for doc, docTypes := range docsIR {
				if len(docTypes) != testCase.TypesLen[doc.Name] {
					subT.Fail()
					return
				}

				for _, tg := range doc.Types {
					if tg == nil {
						subT.Fail()
						return
					}
				}

				for _, v := range docTypes {
					if len(v) > 1 {
						subT.Fail()
						return
					}
				}
			}
		})
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

	docsIR, err := ReduceImports(docs)
	if err != nil {
		t.Error(err)
		return
	}

	if len(docsIR) != 1 {
		t.Fail()
		return
	}

	var types map[string][]*ast.TypeDecl
	for _, t := range docsIR {
		if len(t) > 0 {
			types = t
		}
	}

	if len(types) != 4 {
		t.Fail()
		return
	}

	for _, tg := range docsIR[docs[1]] {
		if tg == nil {
			t.Fail()
			return
		}
	}
}
