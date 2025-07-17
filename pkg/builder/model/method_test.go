package model_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/reedom/convergen/v8/pkg/builder/model"
	"github.com/reedom/convergen/v8/pkg/option"
	"github.com/stretchr/testify/require"
)

func loadSrc(t *testing.T, src string) (*ast.File, *token.FileSet, *types.Package) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", src, parser.ParseComments)
	require.NoError(t, err, "failed to parse test src:\n%v", src)

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("example.go", fset, []*ast.File{file}, nil)
	require.Nil(t, err)
	return file, fset, pkg
}

//func getCodeText(t *testing.T, fset *token.FileSet, file *ast.File) string {
//	buf := bytes.Buffer{}
//	require.Nil(t, printer.Fprint(&buf, fset, file))
//	return buf.String()
//}

func TestMethodEntry(t *testing.T) {
	src := `package main

type MyType int

func (m *MyType) Test1(a, b string) (int, error) {
	return 0, nil
}

func (m *MyType) Test2() {}

func main() {}`

	_, _, pkg := loadSrc(t, src)

	scope := pkg.Scope()
	//imports := util.NewImportNames(scope)

	obj := scope.Lookup("MyType")
	method1, _, _ := types.LookupFieldOrMethod(obj.Type(), true, pkg, "Test1")
	method2, _, _ := types.LookupFieldOrMethod(obj.Type(), true, pkg, "Test2")

	info := &model.MethodsInfo{
		Marker: "test_marker",
		Methods: []*model.MethodEntry{
			{
				Method:     method1,
				Opts:       option.Options{Receiver: "m"},
				DocComment: &ast.CommentGroup{},
			},
			{
				Method:     method2,
				Opts:       option.Options{Receiver: "m"},
				DocComment: &ast.CommentGroup{},
			},
		},
	}

	for _, method := range info.Methods {
		if method.Name() == "" {
			t.Errorf("Method name is empty")
		}
		//if method.Recv() == nil {
		//	t.Errorf("Receiver type is nil")
		//}
		//if len(method.Args()) == 0 {
		//	t.Errorf("Argument types are empty")
		//}
		//if len(method.Results()) == 0 {
		//	t.Errorf("Result types are empty")
		//}
		//if method.SrcVar() == nil {
		//	t.Errorf("SrcVar is nil")
		//}
		//if method.DstVar() == nil {
		//	t.Errorf("DstVar is nil")
		//}
	}
}
