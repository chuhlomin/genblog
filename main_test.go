package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func boolPtr(val bool) *bool {
	return &val
}

func TestProcess(t *testing.T) {
	tests := []struct {
		description string
		b           []byte
		c           config
		source      string
		pageData    pageData
	}{
		{
			description: "Default behavior",
			b:           []byte("---\ndate: 2006-01-02\n---\n# Blogpost\nPost body"),
			c:           config{},
			source:      "2006/blogpost.md",
			pageData: pageData{
				Source: "2006/blogpost.md",
				Path:   "2006/blogpost.html",
				ID:     "2006/blogpost.md",
				Metadata: &metadata{
					Title:             "Blogpost",
					Date:              "2006-01-02",
					Language:          "",
					Tags:              []string{},
					CommentsEnabled:   boolPtr(false),
					TypographyEnabled: boolPtr(false),
				},
				Body: "<p>Post body</p>\n",
			},
		},
		{
			description: "Set default language",
			b:           []byte("---\ndate: 2006-01-02\n---\n# Blogpost\nPost body"),
			c: config{
				DefaultLanguage: "en",
			},
			source: "2006/blogpost_ru.md",
			pageData: pageData{
				Source: "2006/blogpost_ru.md",
				Path:   "2006/blogpost_ru.html",
				ID:     "2006/blogpost.md",
				Metadata: &metadata{
					Title:             "Blogpost",
					Date:              "2006-01-02",
					Language:          "ru",
					Tags:              []string{},
					CommentsEnabled:   boolPtr(false),
					TypographyEnabled: boolPtr(false),
				},
				Body: "<p>Post body</p>\n",
			},
		},
		{
			description: "Empty file",
			b:           []byte(""),
			c:           config{},
			source:      "2006/blogpost.md",
			pageData: pageData{
				Source: "2006/blogpost.md",
				Path:   "2006/blogpost.html",
				ID:     "2006/blogpost.md",
				Metadata: &metadata{
					Title:             "",
					Date:              "",
					Language:          "",
					Tags:              []string{},
					CommentsEnabled:   boolPtr(false),
					TypographyEnabled: boolPtr(false),
				},
				Body: "",
			},
		},
	}

	for _, test := range tests {
		pageData, err := process(test.b, test.c, test.source)

		require.NoError(t, err, test.description)
		require.Equal(t, test.pageData, *pageData, test.description)
	}
}

func TestProcessCommentsEnabled(t *testing.T) {
	tests := []struct {
		description     string
		b               []byte
		c               config
		commentsEnabled bool
	}{
		{
			description:     "Comments disabled by default",
			b:               []byte("---\ndate: 2006-01-02\n---\n"),
			c:               config{},
			commentsEnabled: false,
		},
		{
			description: `Comments can be enabled by passing INPUT_COMMENTS_ENABLED="true" environment variable`,
			b:           []byte("---\ndate: 2006-01-02\n---\n"),
			c: config{
				CommentsEnabled: true,
			},
			commentsEnabled: true,
		},
		{
			description: `comments_enabled metadata in source file overrides INPUT_COMMENTS_ENABLED="true" value`,
			b:           []byte("---\ndate: 2006-01-02\ncomments_enabled: false\n---\n"),
			c: config{
				CommentsEnabled: true,
			},
			commentsEnabled: false,
		},
		{
			description: `comments_enabled metadata in source file overrides INPUT_COMMENTS_ENABLED="false" value`,
			b:           []byte("---\ndate: 2006-01-02\ncomments_enabled: true\n---\n"),
			c: config{
				CommentsEnabled: false,
			},
			commentsEnabled: true,
		},
	}

	for _, test := range tests {
		pageData, err := process(test.b, test.c, "")

		require.NoError(t, err, test.description)
		require.Equal(
			t,
			boolPtr(test.commentsEnabled),
			pageData.Metadata.CommentsEnabled,
			test.description,
		)
	}
}

func TestProcessTags(t *testing.T) {
	tests := []struct {
		description string
		b           []byte
		c           config
		tags        tags
	}{
		{
			description: "There are no default tags",
			b:           []byte("---\ndate: 2006-01-02\n---\n"),
			c:           config{},
			tags:        tags([]string{}),
		},
		{
			description: "Tags can be passed in metadata blog",
			b:           []byte("---\ndate: 2006-01-02\ntags: tagA, tagB\n---\n"),
			c:           config{},
			tags:        tags([]string{"tagA", "tagB"}),
		},
		{
			description: "Tags can be at the bottom of the file",
			b:           []byte("---\ndate: 2006-01-02\n---\n#tagA #tagB"),
			c:           config{},
			tags:        tags([]string{"tagA", "tagB"}),
		},
		{
			description: "Tags can be at the bottom of the file with newline",
			b:           []byte("---\ndate: 2006-01-02\n---\n# Header\n#tvshow\n\n"),
			c:           config{},
			tags:        tags([]string{"tvshow"}),
		},
	}

	for _, test := range tests {
		pageData, err := process(test.b, test.c, "")

		require.NoError(t, err, test.description)
		require.Equal(t, test.tags, pageData.Metadata.Tags, test.description)
	}
}
