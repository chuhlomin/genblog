package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextPage(t *testing.T) {
	tests := []struct {
		data            Data
		defaultLanguage string
		nextPage        *MarkdownFile
		prevPage        *MarkdownFile
	}{
		{
			data: Data{
				Current: &MarkdownFile{
					Path:     "2021/post-1.html",
					ID:       "2021/post-1.md",
					Language: "en",
				},
				All: []*MarkdownFile{
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Language: "en",
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Language: "en",
					},
				},
			},
			defaultLanguage: "en",
			nextPage: &MarkdownFile{
				Path:     "2021/post-2.html",
				ID:       "2021/post-2.md",
				Language: "en",
			},
			prevPage: nil,
		},
		{
			data: Data{
				Current: &MarkdownFile{
					Path:     "2021/post-2.html",
					ID:       "2021/post-2.md",
					Language: "en",
				},
				All: []*MarkdownFile{
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Language: "en",
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Language: "en",
					},
				},
			},
			defaultLanguage: "en",
			nextPage:        nil,
			prevPage: &MarkdownFile{
				Path:     "2021/post-1.html",
				ID:       "2021/post-1.md",
				Language: "en",
			},
		},
		{
			data: Data{
				Current: &MarkdownFile{
					Path:     "2021/post-1.html",
					ID:       "2021/post-1.md",
					Language: "en",
				},
				All: []*MarkdownFile{
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Language: "en",
					},
					{
						Path:     "2021/post-1_ru.html",
						ID:       "2021/post-1.md",
						Language: "ru",
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Language: "en",
					},
				},
			},
			defaultLanguage: "en",
			nextPage: &MarkdownFile{
				Path:     "2021/post-2.html",
				ID:       "2021/post-2.md",
				Language: "en",
			},
			prevPage: nil,
		},
		{
			data: Data{
				Current: &MarkdownFile{
					Path:     "2021/post-1_ru.html",
					ID:       "2021/post-1.md",
					Language: "ru",
				},
				All: []*MarkdownFile{
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Language: "en",
					},
					{
						Path:     "2021/post-1_ru.html",
						ID:       "2021/post-1.md",
						Language: "ru",
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Language: "en",
					},
				},
			},
			defaultLanguage: "en",
			nextPage:        nil,
			prevPage:        nil,
		},
		{
			data: Data{
				Current: &MarkdownFile{
					Path:     "2021/post-1_ru.html",
					ID:       "2021/post-1.md",
					Language: "ru",
				},
				All: []*MarkdownFile{
					{
						Path:     "2021/post-3_ru.html",
						ID:       "2021/post-3.md",
						Language: "ru",
					},
					{
						Path:     "2021/post-2_ru.html",
						ID:       "2021/post-2.md",
						Language: "ru",
					},
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Language: "en",
					},
					{
						Path:     "2021/post-1_ru.html",
						ID:       "2021/post-1.md",
						Language: "ru",
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Language: "en",
					},
				},
			},
			defaultLanguage: "en",
			nextPage: &MarkdownFile{
				Path:     "2021/post-2_ru.html",
				ID:       "2021/post-2.md",
				Language: "ru",
			},
			prevPage: nil,
		},
		{
			data: Data{
				Current: &MarkdownFile{
					Path:     "2021/post-2_ru.html",
					ID:       "2021/post-2.md",
					Language: "ru",
				},
				All: []*MarkdownFile{
					{
						Path:     "2021/post-2_ru.html",
						ID:       "2021/post-2.md",
						Language: "ru",
					},
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Language: "en",
					},
					{
						Path:     "2021/post-1_ru.html",
						ID:       "2021/post-1.md",
						Language: "ru",
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Language: "en",
					},
				},
			},
			defaultLanguage: "en",
			nextPage:        nil,
			prevPage: &MarkdownFile{
				Path:     "2021/post-1_ru.html",
				ID:       "2021/post-1.md",
				Language: "ru",
			},
		},
	}

	for _, test := range tests {
		np := nextPage(test.data)
		require.Equal(t, test.nextPage, np, "next page")

		pp := prevPage(test.data)
		require.Equal(t, test.prevPage, pp, "prev page")
	}
}

func TestGetIDAndLangFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		id       string
		lang     string
	}{
		{
			filename: "index.html",
			id:       "index.html",
			lang:     "",
		},
		{
			filename: "index_en.html",
			id:       "index.html",
			lang:     "en",
		},
		{
			filename: "index_ru.html",
			id:       "index.html",
			lang:     "ru",
		},
		{
			filename: "index",
			id:       "index",
			lang:     "",
		},
	}

	for _, test := range tests {
		id, lang := getIDAndLangFromFilename(test.filename)
		require.Equal(t, test.id, id, "id")
		require.Equal(t, test.lang, lang, "lang")
	}
}

func TestLangToGetParameter(t *testing.T) {
	tests := []struct {
		url             string
		defaultLanguage string
		expectedResult  string
	}{
		{
			url:            "/2021/post_ru.html",
			expectedResult: "/2021/post.html?lang=ru",
		},
		{
			url:            "/2021/post.html",
			expectedResult: "/2021/post.html",
		},
		{
			url:            "/2021/post_en.html",
			expectedResult: "/2021/post.html?lang=en",
		},
	}

	for _, test := range tests {
		result := langToGetParameter(test.url)
		require.Equal(t, test.expectedResult, result)
	}
}

func TestLangGetParameter(t *testing.T) {
	tests := []struct {
		url             string
		defaultLanguage string
		getParameter    string
	}{
		{
			url:             "2006/blogpost.html",
			defaultLanguage: "en",
			getParameter:    "",
		},
		{
			url:             "2006/blogpost_ru.html",
			defaultLanguage: "en",
			getParameter:    "?lang=ru",
		},
		{
			url:             "2006/blogpost_ru.md", // works with both pageData.Path and pageData.ID
			defaultLanguage: "en",
			getParameter:    "?lang=ru",
		},
		{
			url:             "2006/blogpost.html",
			defaultLanguage: "", /* unlikely to happen: func run sets DefaultLanguage="en" */
			getParameter:    "",
		},
		{
			url:             "2006/blogpost_en.html", // why would you do this?
			defaultLanguage: "en",
			getParameter:    "",
		},
	}

	for _, test := range tests {
		cfg = config{DefaultLanguage: test.defaultLanguage}
		getParameter := langGetParameter(test.url)

		require.Equal(
			t,
			test.getParameter,
			getParameter,
			fmt.Sprintf("url %q, defaultLanguage %q", test.url, test.defaultLanguage),
		)
	}
}

func TestYear(t *testing.T) {
	tests := []struct {
		date string
		year string
	}{
		{
			date: "2006-01-02",
			year: "2006",
		},
		{
			date: "",
			year: "",
		},
	}

	for _, test := range tests {
		year := year(test.date)
		require.Equal(t, test.year, year)
	}
}

func TestStripTags(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{
			in:  "Text <tag>123</tag>",
			out: "Text 123",
		},
		{
			in:  "Text",
			out: "Text",
		},
		{
			in:  "",
			out: "",
		},
	}

	for _, test := range tests {
		require.Equal(t, test.out, stripTags(test.in))
	}
}
