package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/m4gshm/fieldr/struc"
)

func (g *Generator) GenerateAsMapFunc(
	model *struc.Model, pkg, name, keyType string,
	excludedFields map[struc.FieldName]struct{},
	rewriter *CodeRewriter,
	export, snake, returnRefs, noReceiver, allFields, hardcodeValues bool,
) (*ast.FuncDecl, error) {

	receiverVar := "v"
	receiverRef := AsRefIfNeed(receiverVar, returnRefs)

	funcName := renameFuncByConfig(goName("AsMap", export), name)
	typeLink := getTypeName(model.TypeName, pkg)
	// var funcBody string
	// if noReceiver {
	// 	funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ") map[" + keyType + "]interface{}"
	// } else {
	// 	funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "() map[" + keyType + "]interface{}"
	// }
	// funcBody += " {" + "\n" +
	// 	"	return map[" + keyType + "]interface{}{\n"

	// for _, fieldName := range model.FieldNames {
	// 	if _, excluded := excludedFields[fieldName]; isFieldExcluded(fieldName, allFields) || excluded {
	// 		continue
	// 	}
	// 	funcBody += getUsedFieldConstName(model.TypeName, fieldName, hardcodeValues, export, snake) + ": " +
	// 		rewriter.Transform(fieldName, model.FieldsType[fieldName], struc.GetFieldRef(receiverRef, fieldName)) + ",\n"
	// }
	// funcBody += "" +
	// 	"	}\n" +
	// 	"}\n"

	elements := []ast.Expr{}
	var rec ast.Expr = &ast.Ident{Name: receiverVar}
	if returnRefs {
		rec = &ast.UnaryExpr{Op: token.AND, X: rec}
	}

	for _, fieldName := range model.FieldNames {
		if _, excluded := excludedFields[fieldName]; isFieldExcluded(fieldName, allFields) || excluded {
			continue
		}
		var val ast.Expr = &ast.SelectorExpr{
			X:   rec,
			Sel: &ast.Ident{Name: fieldName},
		}
		if rawExpr, ok := rewriter.Transform(fieldName, model.FieldsType[fieldName], struc.GetFieldRef(receiverRef, fieldName)); ok {
			if expr, err := parser.ParseExpr(rawExpr); err != nil {
				return nil, fmt.Errorf("rewrite error: field %s: %w", fieldName, err)
			} else {
				val = expr
			}
		}

		elements = append(elements, &ast.KeyValueExpr{
			Key:   &ast.Ident{Name: getUsedFieldConstName(model.TypeName, fieldName, hardcodeValues, export, snake)},
			Value: val,
		})
	}

	funcDecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: funcName},
		Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: receiverVar}}, Type: &ast.Ident{Name: typeLink}}}},
		Type: &ast.FuncType{
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.MapType{Key: &ast.Ident{Name: keyType}, Value: &ast.InterfaceType{Methods: &ast.FieldList{}}}}}},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CompositeLit{
				Type: &ast.MapType{Key: &ast.Ident{Name: keyType}, Value: &ast.InterfaceType{Methods: &ast.FieldList{}}},
				Elts: elements,
			}}}},
		},
	}

	return funcDecl, nil
}
