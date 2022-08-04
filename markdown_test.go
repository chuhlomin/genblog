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
		desc    string
		content []byte
		cfg     config
		source  string
		md      MarkdownFile
	}{
		{
			desc:    "Default behavior",
			content: []byte("---\ndate: 2006-01-02\n---\n# Blogpost\nPost body"),
			cfg:     config{},
			source:  "2006/blogpost.md",
			md: MarkdownFile{
				Source:          "2006/blogpost.md",
				Path:            "2006/blogpost.html",
				Canonical:       "2006/blogpost.html",
				ID:              "2006/blogpost.md",
				Markdown:        "Post body\n",
				Title:           "Blogpost",
				Body:            "<p>Post body</p>\n",
				Date:            "2006-01-02",
				Language:        "",
				ContentType:     "post",
				Tags:            []string{},
				CommentsEnabled: boolPtr(false),
			},
		},
		{
			desc:    "Set default language",
			content: []byte("---\ndate: 2006-01-02\n---\n# Blogpost\nPost body"),
			cfg: config{
				DefaultLanguage: "en",
			},
			source: "2006/blogpost_ru.md",
			md: MarkdownFile{
				Source:          "2006/blogpost_ru.md",
				Path:            "2006/blogpost_ru.html",
				Canonical:       "2006/blogpost.html?lang=ru",
				ID:              "2006/blogpost.md",
				Markdown:        "Post body\n",
				Title:           "Blogpost",
				Body:            "<p>Post body</p>\n",
				Date:            "2006-01-02",
				Language:        "ru",
				ContentType:     "post",
				Tags:            []string{},
				CommentsEnabled: boolPtr(false),
			},
		},
		{
			desc:    "Empty file",
			content: []byte(""),
			cfg:     config{},
			source:  "2006/blogpost.md",
			md: MarkdownFile{
				Source:          "2006/blogpost.md",
				Path:            "2006/blogpost.html",
				Canonical:       "2006/blogpost.html",
				ID:              "2006/blogpost.md",
				Markdown:        "",
				Title:           "",
				Body:            "",
				Date:            "",
				Language:        "",
				ContentType:     "post",
				Tags:            []string{},
				CommentsEnabled: boolPtr(false),
			},
		},
	}

	for _, test := range tests {
		cfg = test.cfg
		md, err := processMarkdownFileContent(test.source, test.content)
		require.NoError(t, err, test.desc)
		require.Equal(t, test.md, *md, test.desc)
	}
}

func TestProcessCommentsEnabled(t *testing.T) {
	tests := []struct {
		desc            string
		cfg             config
		content         []byte
		commentsEnabled bool
	}{
		{
			desc:            "Comments disabled by default",
			cfg:             config{},
			content:         []byte("---\ndate: 2006-01-02\n---\n"),
			commentsEnabled: false,
		},
		{
			desc:            `Comments can be enabled by passing INPUT_COMMENTS_ENABLED="true" environment variable`,
			cfg:             config{CommentsEnabled: true},
			content:         []byte("---\ndate: 2006-01-02\n---\n"),
			commentsEnabled: true,
		},
		{
			desc:            `comments_enabled metadata in source file overrides INPUT_COMMENTS_ENABLED="true" value`,
			cfg:             config{CommentsEnabled: true},
			content:         []byte("---\ndate: 2006-01-02\ncomments_enabled: false\n---\n"),
			commentsEnabled: false,
		},
		{
			desc:            `comments_enabled metadata in source file overrides INPUT_COMMENTS_ENABLED="false" value`,
			cfg:             config{CommentsEnabled: false},
			content:         []byte("---\ndate: 2006-01-02\ncomments_enabled: true\n---\n"),
			commentsEnabled: true,
		},
	}

	for _, test := range tests {
		cfg = test.cfg
		md, err := processMarkdownFileContent("", test.content)

		require.NoError(t, err, test.desc)
		require.Equal(
			t,
			boolPtr(test.commentsEnabled),
			md.CommentsEnabled,
			test.desc,
		)
	}
}

func TestProcessTags(t *testing.T) {
	tests := []struct {
		desc    string
		cfg     config
		content []byte
		tags    tags
	}{
		{
			desc:    "There are no default tags",
			cfg:     config{},
			content: []byte("---\ndate: 2006-01-02\n---\n"),
			tags:    tags([]string{}),
		},
		{
			desc:    "Tags can be passed in metadata blog",
			cfg:     config{},
			content: []byte("---\ndate: 2006-01-02\ntags: tagA, tagB\n---\n"),
			tags:    tags([]string{"tagA", "tagB"}),
		},
		{
			desc:    "Tags can be at the bottom of the file",
			cfg:     config{},
			content: []byte("---\ndate: 2006-01-02\n---\n#tagA #tagB"),
			tags:    tags([]string{"tagA", "tagB"}),
		},
		{
			desc:    "Tags can be at the bottom of the file with newline",
			cfg:     config{},
			content: []byte("---\ndate: 2006-01-02\n---\n# Header\n#tvshow\n\n"),
			tags:    tags([]string{"tvshow"}),
		},
		{
			desc:    "Code commends are not tags",
			cfg:     config{},
			content: []byte("---\ndate: 2006-01-02\n---\nSome text\n```\n# comment\n```\n---\n\n#cli\n\n"),
			tags:    tags([]string{"cli"}),
		},
	}

	for _, test := range tests {
		cfg = test.cfg
		md, err := processMarkdownFileContent("", test.content)
		require.NoError(t, err, test.desc)
		require.Equal(t, test.tags, md.Tags, test.desc)
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
		desc    string
		cfg     config
		content []byte
		images  []image
	}{
		{
			desc:    "Post with image in Markdown (relative path)",
			cfg:     config{ThumbPath: "thumb"},
			content: []byte("---\ndate: 2006-01-02\n---\n![Alt](Path)\n"),
			images: []image{
				{
					Path:      "2022/Path",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/Path",
				},
			},
		},
		{
			desc:    "Post with image in Markdown with title",
			cfg:     config{ThumbPath: "thumb"},
			content: []byte("---\ndate: 2006-01-02\n---\n![Alt](Path \"Title\")\n"),
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
			desc:    "Post with image in Markdown (absolute URL)",
			cfg:     config{ThumbPath: "thumb"},
			content: []byte("---\ndate: 2006-01-02\n---\n![Alt](https://path.com)\n"),
			images: []image{
				{
					Path:      "https://path.com",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/2e5679ee01b14fa7ce7d92a2d349fab44e72d260.com",
				},
			},
		},
		{
			desc:    "Post with many images in one line in Markdown",
			cfg:     config{ThumbPath: "thumb"},
			content: []byte("---\ndate: 2006-01-02\n---\n![Alt1](Path1) ![Alt2](Path2)\n"),
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
			desc:    "Post without images",
			cfg:     config{},
			content: []byte("---\ndate: 2006-01-02\n---\n"),
			images:  nil,
		},
		{
			desc:    "Post with HTML image",
			cfg:     config{ThumbPath: "thumb"},
			content: []byte("---\ndate: 2006-01-02\n---\n<img src=\"Path\" alt=\"Alt\">\n"),
			images: []image{
				{
					Path:      "2022/Path",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/Path",
				},
			},
		},
		{
			desc:    "Post with external image",
			cfg:     config{ThumbPath: "thumb"},
			content: []byte("---\ndate: 2006-01-02\n---\n<img src=\"https://example.com/path.png\" alt=\"Alt\">\n"),
			images: []image{
				{
					Path:      "https://example.com/path.png",
					Alt:       "Alt",
					ThumbPath: "thumb/2022/d8b3c394439d1ab84724f824fdad0c876d41395c.png",
				},
			},
		},
		{
			desc:    "Post with image in metadata",
			cfg:     config{ThumbPath: "thumb"},
			content: []byte("---\ndate: 2006-01-02\nimage: Path\n---\nSome Text\n"),
			images: []image{
				{
					Path:      "2022/Path",
					ThumbPath: "thumb/2022/Path",
				},
			},
		},
		{
			desc:    "Post with image in metadata AND body",
			cfg:     config{ThumbPath: "thumb"},
			content: []byte("---\ndate: 2006-01-02\nimage: Path\n---\n![Alt](Path)\n"),
			images: []image{
				{
					Path:      "2022/Path",
					ThumbPath: "thumb/2022/Path",
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
		cfg = test.cfg
		md, err := processMarkdownFileContent("2022/post.md", test.content)
		require.NoError(t, err, test.desc)
		require.Equal(t, test.images, md.Images, test.desc)
	}
}
