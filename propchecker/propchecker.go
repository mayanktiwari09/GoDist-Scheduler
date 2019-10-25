package propchecker

import (
    "github.com/novalagung/go-eek"
    "go/parser"
    "go/token"
    "go/ast"
    "go/printer"
    "bytes"
    "errors"
//    "log"
)

type Variable struct {
    Name string // Fully qualified package name of the variable
    PropName string // Name Used in the Property
    Type string // Type of the Variable
}

type Property struct {
    Name string // Name of the property
    Variables []Variable // List of all variables used in this property
    PropObj *eek.Eek
}

type Checker struct {
    Properties []Property // List of Properties to be checked
    ImpVariables []string // Names of the Variables need to be tracked for Property Checking
}

func parsePropertyFile(file string) ([]Property, []string, error) {
    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
    if err != nil {
        return []Property{}, []string{}, err
    }
    properties := []Property{}
    impVariables := []string{}
    impVariablesMap := make(map[string]bool)
    for _, f := range node.Decls {
        var name string
        var_names := []string{}
        prop_var_names := []string{}
        types := []string{}
        fn, ok := f.(*ast.FuncDecl)
        if !ok {
            continue
        }
        for i, c := range fn.Doc.List {
            if i == 0 {
                name = c.Text[2:]
            } else {
                var_name := c.Text[2:]
                var_names = append(var_names, var_name)
                impVariablesMap[var_name] = true
            }
        }
        for _, p := range fn.Type.Params.List {
            prop_var_names = append(prop_var_names, p.Names[0].Name)
            var typeName bytes.Buffer
            err := printer.Fprint(&typeName, fset, p.Type)
            if err != nil {
                return properties, impVariables, err
            }
            types = append(types, typeName.String())
        }
        body_statement := ""
        for _, s := range fn.Body.List {
            var statement bytes.Buffer
            err := printer.Fprint(&statement, fset, s)
            if err != nil {
                return properties, impVariables, err
            }
            body_statement += statement.String() + "\n"
        }
        if err != nil {
            return properties, impVariables, err
        }
        prop_obj := eek.New()
        prop_obj.SetName(name)
        vars := []Variable{}
        for i := 0; i < len(var_names); i++ {
            new_var := Variable{Name: var_names[i], PropName: prop_var_names[i], Type: types[i]}
            vars = append(vars, new_var)
            prop_obj.DefineVariable(eek.Var{Name: prop_var_names[i], Type: types[i]})
        }
        prop_obj.PrepareEvaluation(body_statement)
        err := prop_obj.Build()
        if err != nil {
            return properties, impVariables, err
        }
        property := Property{Name: name, Variables: vars, PropObj: prop_obj}
        properties = append(properties, property)
    }
    for k := range impVariablesMap {
        impVariables = append(impVariables, k)
    }
    return properties, impVariables, nil
}

func NewChecker(property_file string) (*Checker, error) {
    properties, impVariables, err := parsePropertyFile(property_file)
    if err != nil {
        return nil, err
    }

    return &Checker{Properties: properties, ImpVariables: impVariables}, nil
}

func (c* Checker) Check(context map[string]interface{}) (bool, error) {
    result := true
    for _, property := range c.Properties {
        execVar := eek.ExecVar{}
        for _, variable := range property.Variables {
            execVar[variable.PropName] = context[variable.Name]
        }
        temp_res, err := property.PropObj.Evaluate(execVar)
        if err != nil {
            return result, err
        }
        temp_res_val, found := temp_res.(bool)
        if !found {
            return result, errors.New("Property doesn't return a bool value")
        }
        result = result && temp_res_val
    }
    return result, nil
}

func (c* Checker) GetImportantVariables() []string {
    return c.ImpVariables
}