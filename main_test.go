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
		{
			description: "Code commends are not tags",
			b:           []byte("---\ndate: 2006-01-02\n---\nSome text\n```\n# comment\n```\n---\n\n#cli\n\n"),
			c:           config{},
			tags:        tags([]string{"cli"}),
		},
	}

	for _, test := range tests {
		pageData, err := process(test.b, test.c, "")

		require.NoError(t, err, test.description)
		require.Equal(t, test.tags, pageData.Metadata.Tags, test.description)
	}
}

func TestFixPath(t *testing.T) {
	tests := []struct {
		path              string
		relativePath      string
		expectedPath      string
		expectedThumbPath string
	}{
		{
			"Path",
			"2022",
			"2022/Path",
			"thumb/2022/Path",
		},
		{
			"../2022/image.png",
			"2022",
			"2022/image.png",
			"thumb/2022/image.png",
		},
		{
			"https://example.com/path.png",
			"2022",
			"https://example.com/path.png",
			"thumb/2022/d8b3c394439d1ab84724f824fdad0c876d41395c.png",
		},
	}

	for _, test := range tests {
		path, thumbPath := fixPath(test.path, test.relativePath, "thumb/2022")
		require.Equal(t, test.expectedPath, path)
		require.Equal(t, test.expectedThumbPath, thumbPath)
	}
}

func TestProcessImages(t *testing.T) {
	tests := []struct {
		description string
		b           []byte
		c           config
		images      []image
	}{
		{
			description: "Post with image in Markdown (relative path)",
			b:           []byte("---\ndate: 2006-01-02\n---\n![Alt](Path)\n"),
			c: config{
				ThumbPath: "thumb",
			},
			images: []image{
				{
					Path:      "2022/Path",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/Path",
				},
			},
		},
		{
			description: "Post with image in Markdown with title",
			b:           []byte("---\ndate: 2006-01-02\n---\n![Alt](Path \"Title\")\n"),
			c: config{
				ThumbPath: "thumb",
			},
			images: []image{
				{
					Path:      "2022/Path",
					Alt:       "Alt",
					Title:     "Title",
					ThumbPath: "thumb/2022/Path",
				},
			},
		},
		{
			description: "Post with image in Markdown (absolute URL)",
			b:           []byte("---\ndate: 2006-01-02\n---\n![Alt](https://path.com)\n"),
			c: config{
				ThumbPath: "thumb",
			},
			images: []image{
				{
					Path:      "https://path.com",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/2e5679ee01b14fa7ce7d92a2d349fab44e72d260.com",
				},
			},
		},
		{
			description: "Post with many images in one line in Markdown",
			b:           []byte("---\ndate: 2006-01-02\n---\n![Alt1](Path1) ![Alt2](Path2)\n"),
			c: config{
				ThumbPath: "thumb",
			},
			images: []image{
				{
					Path:      "2022/Path1",
					Alt:       "Alt1",
					ThumbPath: "thumb/2022/Path1",
				},
				{
					Path:      "2022/Path2",
					Alt:       "Alt2",
					ThumbPath: "thumb/2022/Path2",
				},
			},
		},
		{
			description: "Post without images",
			b:           []byte("---\ndate: 2006-01-02\n---\n"),
			c:           config{},
			images:      nil,
		},
		{
			description: "Post with HTML image",
			b:           []byte("---\ndate: 2006-01-02\n---\n<img src=\"Path\" alt=\"Alt\">\n"),
			c: config{
				ThumbPath: "thumb",
			},
			images: []image{
				{
					Path:      "2022/Path",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/Path",
				},
			},
		},
		{
			description: "Post with external image",
			b:           []byte("---\ndate: 2006-01-02\n---\n<img src=\"https://example.com/path.png\" alt=\"Alt\">\n"),
			c: config{
				ThumbPath: "thumb",
			},
			images: []image{
				{
					Path:      "https://example.com/path.png",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/d8b3c394439d1ab84724f824fdad0c876d41395c.png",
				},
			},
		},
		{
			description: "Post with image in metadata",
			b:           []byte("---\ndate: 2006-01-02\nimage: Path\n---\nSome Text\n"),
			c: config{
				ThumbPath: "thumb",
			},
			images: []image{
				{
					Path:      "2022/Path",
					ThumbPath: "thumb/2022/Path",
					Promo:     true,
				},
			},
		},
		{
			description: "Post with image in metadata AND body",
			b:           []byte("---\ndate: 2006-01-02\nimage: Path\n---\n![Alt](Path)\n"),
			c: config{
				ThumbPath: "thumb",
			},
			images: []image{
				{
					Path:      "2022/Path",
					ThumbPath: "thumb/2022/Path",
					Promo:     true,
				},
				{
					Path:      "2022/Path",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/Path",
				},
			},
		},
	}

	for _, test := range tests {
		pageData, err := process(test.b, test.c, "2022/post.md")

		require.NoError(t, err, test.description)
		require.Equal(t, test.images, pageData.Metadata.Images, test.description)
	}
}
