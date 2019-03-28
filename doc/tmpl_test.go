package doc

import (
	"bytes"
	"fmt"
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
	"os"
	"strings"
	"testing"
	"text/template"
)

var testTmpl = template.New("testTmpl")

func TestMain(m *testing.M) {
	testTmpl.Funcs(map[string]interface{}{
		"add":     func(a, b int) int { return a + b },
		"Title":   strings.Title,
		"ToLower": strings.ToLower,
		"ToMembers": func(t ast.Expr) (mems []string) {
			unt := t.(*ast.UnionType)
			for _, mem := range unt.Members {
				mems = append(mems, mem.Name)
			}
			return
		},
		"ToFieldData": ToFieldData,
		"ToObjData":   ToObjData,
		"Trim":        func(s string) string { return strings.Trim(strings.TrimSpace(s), "\"") },
		"PrintDir": func(d *ast.DirectiveLit) string {
			var s strings.Builder
			s.WriteRune('@')
			s.WriteString(d.Name)

			if d.Args == nil {
				return s.String()
			}

			s.WriteRune('(')
			aLen := len(d.Args.Args)
			for i, arg := range d.Args.Args {
				s.WriteString(arg.Name.Name)
				s.WriteString(": ")
				switch v := arg.Value.(type) {
				case *ast.BasicLit:
					s.WriteString(v.Value)
				case *ast.ListLit:
				case *ast.ObjLit:
				}

				if i < aLen-1 {
					s.WriteString(", ")
				}
			}
			s.WriteRune(')')
			return s.String()
		},
	})
	template.Must(testTmpl.Parse(mdTmpl))
	template.Must(testTmpl.New("objTmpl").Parse(objTmpl))
	template.Must(testTmpl.New("fieldListTmpl").Parse(fieldListTmpl))

	os.Exit(m.Run())
}

func TestFieldListTmpl(t *testing.T) {
	listTestData := []struct {
		Name string
		In   string
		Out  string
	}{
		{
			Name: "plain",
			In: `type Test {
	a: A
	b: B
	c: Int
}`,
			Out: `- a

	*Type*: **[A](#a)**

- b

	*Type*: **[B](#b)**

- c

	*Type*: **Int**`,
		},
		{
			Name: "withListAndNonNull",
			In: `type Test {
	a: [A]
	b: B!
	c: [Int!]!
}`,
			Out: `- a

	*Type*: **[[A](#a)]**

- b

	*Type*: **[B](#b)!**

- c

	*Type*: **[Int!]!**`,
		},
		{
			Name: "withDirectives",
			In: `type Test {
	a: A @a
	b: B @a @b(x: 1)
	c: Int! @a @b @c(x: 1, y: "2")
}`,
			Out: `- a

	*Type*: **[A](#a)**

	*Directives*: @a,

- b

	*Type*: **[B](#b)**

	*Directives*: @a, @b(x: 1),

- c

	*Type*: **Int!**

	*Directives*: @a, @b, @c(x: 1, y: "2"),`,
		},
		{
			Name: "withArgs",
			In: `type Test {
	a(): A
	b(id: ID): B!
	c(
		"Arg Description"
		w: Int = 100 @a @b(x: 3),

		"Arg Description"
		x: Int,

		y: [String],

		z: Int @a @b(x: 3),
	): Int
}`,
			Out: `- a

	*Type*: **[A](#a)**

- b

	*Type*: **[B](#b)!**

	*Args*:
	- id

		*Type*: **ID**

- c

	*Type*: **Int**

	*Args*:
	- w

		*Type*: **Int** = 100
		*Directives*: @a, @b(x: 3),

		Arg Description

	- x

		*Type*: **Int**

		Arg Description

	- y

		*Type*: **[String]**

	- z

		*Type*: **Int**
		*Directives*: @a, @b(x: 3),`,
		},
	}

	fieldListTmpl := testTmpl.Lookup("fieldListTmpl")
	for _, testCase := range listTestData {
		t.Run(testCase.Name, func(subT *testing.T) {
			doc, err := parser.ParseDoc(token.NewDocSet(), testCase.Name, strings.NewReader(testCase.In), 0)
			if err != nil {
				subT.Errorf("unexpected error while parsing: %s", err)
				return
			}

			fieldList := doc.Types[0].Specs[0].(*ast.TypeSpec).Type.(*ast.ObjectType).Fields

			var b bytes.Buffer
			err = fieldListTmpl.Execute(&b, fieldList)
			if err != nil {
				subT.Errorf("unexpected error when executing fieldListTmpl: %s", err)
				return
			}

			var line, col int
			oBytes := b.Bytes()
			exBytes := []byte(testCase.Out)
			for i := 0; i < len(exBytes); i++ {
				if exBytes[i] != oBytes[i] {
					fmt.Printf("%d:%d :%s:%s:\n", line+1, col+1, string(exBytes[i]), string(oBytes[i]))
					fmt.Println(b.String())
					subT.Fail()
					break
				}
				if r := rune(exBytes[i]); r != '\n' && r != '\r' {
					continue
				}

				line++
				col++
			}
		})
	}
}
