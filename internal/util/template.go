package util

import (
	"bytes"
	"fmt"
	"text/template"

	templatefs "whatsupbot/internal/template"
)

var tmpl *template.Template

func InitTemplate() {
	tmpl = template.Must(template.ParseFS(templatefs.FS, "*.tmpl"))
}

func Render(templatePath string, data any) (string, error) {
	if tmpl == nil {
		return "", fmt.Errorf("render: failed to initialize")
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, templatePath, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
