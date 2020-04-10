package template

import (
	"bytes"
	"fmt"
	"reflect"
)

func eval(context *Context, expressions ...string) (result string, err error) {
	t := context.Template().New("eval")
	data := make(Map)

	if context.Dot().IsValid() && context.Dot().Type().ConvertibleTo(reflect.TypeOf(data)) {
		iter := context.Dot().Convert(reflect.TypeOf(data)).MapRange()
		for iter.Next() {
			data[iter.Key().String()] = iter.Value()
		}
	}

	var init string
	for key, value := range context.Variables() {
		data[key] = value
		init += fmt.Sprintf(`{{- %[1]s := index $ "%[1]s" -}}`, key)
	}
	for _, expr := range expressions {
		var buffer bytes.Buffer
		if t, err = t.Parse(init + expr); err == nil {
			err = t.Execute(&buffer, data)
		}
		if err != nil {
			return result, err
		}
		result += buffer.String()
	}
	return result, nil
}
