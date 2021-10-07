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
