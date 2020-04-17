package template

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_flow_control(t *testing.T) {
	t.Parallel()

	var (
		code = `
			{{- "List: " -}}
			{{- range $i, $value := seq 10 -}}
				{{- if $i }}-{{ end -}}
				{{- if %s $value %d }}{{ %s }}{{- end -}}
				{{ $value }}
			{{- end -}}
			{{- "!" -}}
		`

		sequence = func(n int) []int {
			result := make([]int, n)
			for i := range result {
				result[i] = i + 1
			}
			return result
		}

		tests = []struct {
			statement string
			condition string
			value     int
			wanted    string
		}{
			{"break", "gt", 5, "List: 1-2-3-4-5-!"},
			{"continue", "le", 8, "List: --------9-10!"},
			{"return", "eq", 7, "List: 1-2-3-4-5-6-"},
			{"return 1 2", "eq", 4, "[1 2]"},
			{`return "Under" 20`, "ge", 20, "List: 1-2-3-4-5-6-7-8-9-10!"},
			{`return "Over" 2`, "le", 2, "[Over 2]"},
		}
	)

	for _, tc := range tests {
		t.Run(tc.statement, func(t *testing.T) {
			var (
				buffer = new(bytes.Buffer)
				code   = fmt.Sprintf(code, tc.condition, tc.value, tc.statement)
				err    error
			)

			tmpl, err := New("test").Option(FlowControl).Funcs(FuncMap{"seq": sequence}).Parse(code)
			if err == nil {
				err = tmpl.Execute(buffer, nil)
				assert.EqualValues(t, tc.wanted, buffer.String(), code)
			}
			assert.NoError(t, err, code)
		})
	}
}
