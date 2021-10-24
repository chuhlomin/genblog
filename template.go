package main

import (
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

const templatePost = "post.html"

var fm = template.FuncMap{
	"bool":                  boolean, // Go teamplates doesn't check value of a pointer
	"back":                  back,
	"prevPage":              prevPage,
	"nextPage":              nextPage,
	"allLanguageVariations": allLanguageVariations,
	"langToGetParameter":    langToGetParameter,
}

var langSuffix = regexp.MustCompile(`_([a-z]{2}).html$`)

func renderTemplate(filename string, data interface{}, t *template.Template) error {
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

func parseFiles(funcs template.FuncMap, filenames ...string) (*template.Template, error) {
	return template.New(filepath.Base(filenames[0])).Funcs(funcs).ParseFiles(filenames...)
}

func boolean(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func back(path string) string {
	return strings.Repeat("../", len(strings.Split(path, "/"))-1)
}

func prevPage(page page) (prev *pageData) {
	// technically, it get's the NEXT page
	// from the list of all pages SORTED by created date (descending)
	// but chronologically, it's the PREVIOUS page
	prev = nil

	var (
		i int
		p *pageData
	)

	for i, p = range page.AllPages {
		if p.Path == page.CurrentPage.Path { // searching for the current page
			break
		}
	}

	for _, p := range page.AllPages[i+1:] {
		if p.ID == page.CurrentPage.ID { // skipping same pages in different languages
			continue
		}
		if p.Metadata.Language == page.CurrentPage.Metadata.Language { // first page in the same language
			prev = p
			break
		}
	}

	return
}

func nextPage(page page) (next *pageData) {
	// technically, it get's the PREVIOUS page
	// from the list of all pages SORTED by created date (descending)
	// but chronologically, it's the NEXT page
	next = nil

	for _, p := range page.AllPages {
		if p.Path == page.CurrentPage.Path { // searching for the current page
			break // this is the most recent page, so there's no next page
		}
		if p.Metadata.Language == page.CurrentPage.Metadata.Language {
			// last seen page in the same language
			next = p
		}
	}

	return
}

func allLanguageVariations(page page) []*pageData {
	if len(page.AllLanguageVariations) > 0 {
		return page.AllLanguageVariations
	}

	var result []*pageData

	for _, p := range page.AllPages {
		if p.ID == page.CurrentPage.ID {
			result = append(result, p)
		}
	}

	sort.Sort(ByLanguage(result))

	return result
}

func langToGetParameter(url string) string {
	// if url matches langSuffix regex then replace it with .html and append ?lang=ru
	if langSuffix.MatchString(url) {
		suffix := langSuffix.FindStringSubmatch(url)
		return url[:len(url)-len(suffix[0])] + ".html?lang=" + suffix[1]
	}

	return url
}
