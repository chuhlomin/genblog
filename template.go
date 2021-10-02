package main

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const templatePost = "post.html"

var fm = template.FuncMap{
	"back": back,
}

func writeFile(filename string, data interface{}, t *template.Template) error {
	// create directories for file
	dir := filepath.Dir(filename)
	if err := os.MkdirAll("./"+dir, permDir); err != nil {
		return errors.Wrapf(err, "create directories %s for %s", dir, filename)
	}

	// open file
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
	if err != nil {
		return errors.Wrapf(err, "creating %s", filename)
	}
	defer f.Close()

	// execute template
	if err = t.Execute(f, data); err != nil {
		return errors.Wrap(err, "template execution")
	}

	return nil
}

func getPostTemplate(t *template.Template) *template.Template {
	t = t.Lookup(templatePost)
	if t == nil {
		return template.Must(
			template.New(templatePost).Funcs(fm).Parse(`<!DOCTYPE html>
<html>
<head>
<title>{{.Metadata.Title}}</title>
</head>
<body>
{{.Body}}
</body>
</html>`),
		)
	}

	return t
}

func back(path string) string {
	return strings.Repeat("../", len(strings.Split(path, "/"))-1)
}

func parseFiles(funcs template.FuncMap, filenames ...string) (*template.Template, error) {
	return template.New(filepath.Base(filenames[0])).Funcs(funcs).ParseFiles(filenames...)
}
