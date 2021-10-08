package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextPage(t *testing.T) {
	tests := []struct {
		page            page
		defaultLanguage string
		nextPage        *pageData
		prevPage        *pageData
	}{
		{
			page: page{
				CurrentPage: &pageData{
					Path:     "2021/post-1.html",
					ID:       "2021/post-1.md",
					Metadata: &metadata{Language: "en"},
				},
				AllPages: []*pageData{
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Metadata: &metadata{Language: "en"},
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "en"},
					},
				},
			},
			defaultLanguage: "en",
			nextPage: &pageData{
				Path:     "2021/post-2.html",
				ID:       "2021/post-2.md",
				Metadata: &metadata{Language: "en"},
			},
			prevPage: nil,
		},
		{
			page: page{
				CurrentPage: &pageData{
					Path:     "2021/post-2.html",
					ID:       "2021/post-2.md",
					Metadata: &metadata{Language: "en"},
				},
				AllPages: []*pageData{
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Metadata: &metadata{Language: "en"},
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "en"},
					},
				},
			},
			defaultLanguage: "en",
			nextPage:        nil,
			prevPage: &pageData{
				Path:     "2021/post-1.html",
				ID:       "2021/post-1.md",
				Metadata: &metadata{Language: "en"},
			},
		},
		{
			page: page{
				CurrentPage: &pageData{
					Path:     "2021/post-1.html",
					ID:       "2021/post-1.md",
					Metadata: &metadata{Language: "en"},
				},
				AllPages: []*pageData{
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Metadata: &metadata{Language: "en"},
					},
					{
						Path:     "2021/post-1_ru.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "ru"},
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "en"},
					},
				},
			},
			defaultLanguage: "en",
			nextPage: &pageData{
				Path:     "2021/post-2.html",
				ID:       "2021/post-2.md",
				Metadata: &metadata{Language: "en"},
			},
			prevPage: nil,
		},
		{
			page: page{
				CurrentPage: &pageData{
					Path:     "2021/post-1_ru.html",
					ID:       "2021/post-1.md",
					Metadata: &metadata{Language: "ru"},
				},
				AllPages: []*pageData{
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Metadata: &metadata{Language: "en"},
					},
					{
						Path:     "2021/post-1_ru.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "ru"},
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "en"},
					},
				},
			},
			defaultLanguage: "en",
			nextPage:        nil,
			prevPage:        nil,
		},
		{
			page: page{
				CurrentPage: &pageData{
					Path:     "2021/post-1_ru.html",
					ID:       "2021/post-1.md",
					Metadata: &metadata{Language: "ru"},
				},
				AllPages: []*pageData{
					{
						Path:     "2021/post-3_ru.html",
						ID:       "2021/post-3.md",
						Metadata: &metadata{Language: "ru"},
					},
					{
						Path:     "2021/post-2_ru.html",
						ID:       "2021/post-2.md",
						Metadata: &metadata{Language: "ru"},
					},
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Metadata: &metadata{Language: "en"},
					},
					{
						Path:     "2021/post-1_ru.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "ru"},
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "en"},
					},
				},
			},
			defaultLanguage: "en",
			nextPage: &pageData{
				Path:     "2021/post-2_ru.html",
				ID:       "2021/post-2.md",
				Metadata: &metadata{Language: "ru"},
			},
			prevPage: nil,
		},
		{
			page: page{
				CurrentPage: &pageData{
					Path:     "2021/post-2_ru.html",
					ID:       "2021/post-2.md",
					Metadata: &metadata{Language: "ru"},
				},
				AllPages: []*pageData{
					{
						Path:     "2021/post-2_ru.html",
						ID:       "2021/post-2.md",
						Metadata: &metadata{Language: "ru"},
					},
					{
						Path:     "2021/post-2.html",
						ID:       "2021/post-2.md",
						Metadata: &metadata{Language: "en"},
					},
					{
						Path:     "2021/post-1_ru.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "ru"},
					},
					{
						Path:     "2021/post-1.html",
						ID:       "2021/post-1.md",
						Metadata: &metadata{Language: "en"},
					},
				},
			},
			defaultLanguage: "en",
			nextPage:        nil,
			prevPage: &pageData{
				Path:     "2021/post-1_ru.html",
				ID:       "2021/post-1.md",
				Metadata: &metadata{Language: "ru"},
			},
		},
	}

	for _, test := range tests {
		np := nextPage(test.page)
		require.Equal(t, test.nextPage, np, "next page")

		pp := prevPage(test.page)
		require.Equal(t, test.prevPage, pp, "prev page")
	}
}

func TestGetLanguageFromFilename(t *testing.T) {
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
		id, lang := getLanguageFromFilename(test.filename)
		require.Equal(t, test.id, id, "id")
		require.Equal(t, test.lang, lang, "lang")
	}
}
