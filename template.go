package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	i "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
)

var fm = template.FuncMap{
	"debugJSON":             debugJSON,             // JSON debug print
	"join":                  join,                  // alias for strings.Join
	"bool":                  boolean,               // Check the value of a pointer to bool
	"prevPage":              prevPage,              // previous page data
	"nextPage":              nextPage,              // next page data
	"allLanguageVariations": allLanguageVariations, // all pages that has the same ID as current page
	"langGetParameter":      langGetParameter,      // get lang parameter value for page
	"langToGetParameter":    langToGetParameter,    // replace lang suffix with .html and append ?lang=ru, e.g. index_ru.html -> index.html?lang=ru
	"year":                  year,                  // gets the year from date of format "2006-01-02"
	"i18n":                  i18n,                  // translate string
	"stripTags":             stripTags,             // remove html tags
	"config":                getConfigValue,        // get config value
	"sort":                  sortFiles,
}

var langSuffix = regexp.MustCompile(`_([a-z]{2}).(html|md)$`)

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

func debugJSON(v interface{}) string {
	return string(prettyJSON(v))
}

func prettyJSON(v interface{}) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return b
}

func join(elems []string, sep string) string {
	return strings.Join(elems, sep)
}

func boolean(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func prevPage(data Data) (prev *MarkdownFile) {
	// technically, it get's the NEXT page
	// from the list of all pages SORTED by created date (descending)
	// but chronologically, it's the PREVIOUS page
	prev = nil

	var (
		i    int
		file *MarkdownFile
	)

	for i, file = range data.All {
		if file.Path == data.Current.Path { // searching for the current page
			break
		}
	}

	for _, file := range data.All[i+1:] {
		if file.ID == data.Current.ID { // skipping same pages in different languages
			continue
		}
		if file.Language == data.Current.Language { // first page in the same language
			prev = file
			break
		}
	}

	return
}

func nextPage(data Data) (next *MarkdownFile) {
	// technically, it get's the PREVIOUS page
	// from the list of all pages SORTED by created date (descending)
	// but chronologically, it's the NEXT page
	next = nil

	for _, file := range data.All {
		if file.Path == data.Current.Path { // searching for the current page
			break // this is the most recent page, so there's no next page
		}
		if file.Language == data.Current.Language {
			// last seen page in the same language
			next = file
		}
	}

	return
}

func allLanguageVariations(data Data) []*MarkdownFile {
	if len(data.LanguageVariations) > 0 {
		return data.LanguageVariations
	}

	var result []*MarkdownFile

	for _, file := range data.All {
		if file.ID == data.Current.ID {
			result = append(result, file)
		}
	}

	sort.Sort(ByLanguage(result))

	return result
}

func langGetParameter(path string) string {
	if !langSuffix.MatchString(path) {
		return ""
	}

	match := langSuffix.FindStringSubmatch(path)
	if len(match) < 2 { // always false, just some extra caution
		return ""
	}

	lang := match[1]
	if lang == cfg.DefaultLanguage {
		return ""
	}

	return "?lang=" + lang
}

func langToGetParameter(url string) string {
	// if url matches langSuffix regex then replace it with .html and append ?lang=ru
	// That trick useful if you have some reverse proxy that rewrite urls
	// e.g. nginx config like this:
	//
	//   # make requests to _ru.html internal
	//   location ~ "^(.*)_([a-z]){2}.html$" {
	//     internal;
	//   }
	//
	//   location / {
	//     # get "lang" parameter value if presented
	//     set $lang_code "en";
	//     if ($arg_lang != '') {
	//       set $lang_code $arg_lang;
	//     }
	//
	//     # if lang is not "en", replace ".html" with "_$lang.html"
	//     if ($lang_code !~ "en") {
	//       rewrite ^(.*)\.html$ /$1_$lang_code.html last;
	//     }
	//   }
	if langSuffix.MatchString(url) {
		match := langSuffix.FindStringSubmatch(url)
		return url[:len(url)-len(match[0])] + ".html?lang=" + match[1]
	}

	return url
}

func year(date string) string {
	if len(date) < 4 {
		return ""
	}

	return date[:4]
}

func i18n(id string, lang string) string {
	localizer := i.NewLocalizer(bundle, lang)
	str, err := localizer.Localize(&i.LocalizeConfig{
		MessageID: id,
	})

	if err != nil {
		panic(err)
	}

	return str
}

func stripTags(html string) string {
	htmlTagRegexp := regexp.MustCompile("<[^>]*>")
	return htmlTagRegexp.ReplaceAllString(string(html), "")
}

func getConfigValue(key string) string {
	return cfg.GetString(key)
}

func sortFiles(files []*MarkdownFile, field string) []*MarkdownFile {
	switch field {
	case "created":
		sort.Sort(ByCreated(files))
	case "order":
		sort.Sort(ByOrder(files))
	default:
		log.Printf("unknown sort field: %s", field)
	}
	return files
}
