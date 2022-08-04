# GenBlog

[![main](https://github.com/chuhlomin/genblog/actions/workflows/main.yml/badge.svg)](https://github.com/chuhlomin/genblog/actions/workflows/main.yml)
[![release](https://github.com/chuhlomin/genblog/actions/workflows/release.yml/badge.svg)](https://github.com/chuhlomin/genblog/actions/workflows/release.yml)
[![Dockerhub](https://img.shields.io/badge/docker-hub-4988CC)](https://hub.docker.com/repository/docker/chuhlomin/genblog)

Generate a static blog from Markdown files.

## Inputs

| Name                      | Description                                                                     | Default                    |
|---------------------------|---------------------------------------------------------------------------------|----------------------------|
| `base_path`               | Base path for all generated URLs                                                | "/"                        |
| `source_directory`        | Path to directory with Markdown files                                           | "."                        |
| `output_directory`        | Path to output directory                                                        | "output"                   |
| `allowed_file_extensions` | Comma-separated list of allowed file extensions that will be copied as is       | "jpeg,.jpg,.png,.mp4,.pdf" |
| `templates_directory`     | Path to templates directory                                                     | "_templates"               |
| `default_template`        | Filename of the default template                                                | "_post.html"               |
| `default_language`        | Default language of the blog                                                    | "en"                       |
| `comments_enabled`        | Enable comments                                                                 | "false"                    |
| `comments_site_id`        | Site ID for Remark42 comments                                                   | ""                         |
| `show_drafts`             | Show drafts                                                                     | "false"                    |
| `thumb_path`              | Path to thumbnails directory                                                    | "thumb"                    |
| `thumb_max_width`         | Max width of thumbnails                                                         | "140"                      |
| `thumb_max_height`        | Max height of thumbnails                                                        | "140"                      |
| `search_enabled`          | Create `bleve` index directory                                                  | "false"                    |
| `search_url`              | Search URL prefix                                                               | ""                         |
| `search_path`             | Path to `bleve` index directory                                                 | "index.bleve"              |

Genblog scans files in the `source_directory`.

For every Markdown file it renders HTML page in the `output_directory`
using `default_template` in the `templates_directory`,
keeping the same directory structure.

For any file that have `allowed_file_extensions` it just copies it to the
`output_directory`, keeping the same directory structure.

## Post metadata

```md
---
date: 2022-01-01
---

# Header

Body

#tag1 #tag2
```

## Templates

Genblog uses Go [html/template](https://pkg.go.dev/html/template) to render pages.

### `Data`

Genblog passes the following structure to `default_template` to render
individual post pages:

| Field       | Type             | Description                                                     |
|-------------|------------------|-----------------------------------------------------------------|
| `Current`   | `MarkdownFile`   | Current post, corresponds to a single Markdown file (see below) |
| `All`       | `[]MarkdownFile` | Array of all available posts                                    |
| `Timestamp` | `int64`          | Unix timestamp when Genblog was started                         |

### `MarkdownFile`

`MarkdownFile` structure has these fields:

| Field             | Type       | Description                                                                    |
|-------------------|------------|--------------------------------------------------------------------------------|
| `Source`          | `string`   | Relative path to the source Markdown file                                      |
| `Path`            | `string`   | Relative path to the generated HTML file                                       |
| `Canonical`       | `string`   | Canonical URL of the post                                                      |
| `ID`              | `string`   | Same post in different languages will have the same ID value                   |
| `Markdown`        | `string`   | Markdown file content                                                          |
| `Title`           | `string`   | By default equals to `H1` in Markdown file                                     |
| `Body`            | `string`   | Rendered HTML body                                                             |
| `Date`            | `string`   | date when post was published, in format "2006-01-02"                           |
| `Tags`            | `[]string` | Post tags, by default parsed from the post                                     |
| `Language`        | `string`   | Language ("en", "ru", ...), parsed from filename, overrides `default_language` |
| `Description`     | `string`   | Used in the `meta` description tag                                             |
| `Author`          | `string`   | Used in the `meta` author tag, overrides `author` input                        |
| `Keywords`        | `string`   | Used in the `meta` keywords tag                                                |
| `Draft`           | `bool`     | Marks post as draft, `false` by default                                        |
| `Template`        | `string`   | Template to use, overrides the default "`post.html`"                           |
| `CommentsEnabled` | `bool`     | Overrides `comments_enabled` input                                             |
| `Image`           | `string`   | Image associated with the post; it's used to generate the thumbnail            |
| `Images`          | `[]image`  | All images associated with the post                                            |

### `image`

`image` structure has these fields:

| Field       | Type     | Description                                |
|-------------|----------|--------------------------------------------|
| `Path`      | `string` | Relative path to the original image        |
| `Alt`       | `string` | Image alt text                             |
| `Title`     | `string` | Image title text                           |
| `ThumbPath` | `string` | Relative path to generated thumbnail image |

### Template functions

The following functions are defined and can be used in templates:

| Function                | Description                                       | Return type  | Example usage                                                                   |
|-------------------------|---------------------------------------------------|--------------|---------------------------------------------------------------------------------|
| `debugJSON`             | Prints JSON of the given object                   | `string`     | `{{ debugJSON . }}`                                                             |
| `stripTags`             | Strips HTML tags from the given string            | `string`     | `<title>{{ stripTags .Title }}</title>`                                         |
| `config`                | Returns config value                              | `string`     | `{{ config "SearchURL" }}`                                                      |
| `join`                  | Joins the given list of strings                   | `string`     | `{{ join .Metadata.Tags "," }}`                                                 |
| `prevPage`              | Returns previous page                             | `pageData`   | `{{ $prev := prevPage . }}{{ $prev.Path }}`                                     |
| `nextPage`              | Returns next page                                 | `pageData`   | `{{ $next := nextPage . }}{{ $next.Path }}`                                     |
| `allLanguageVariations` | Returns all language variations of the given post | `[]pageData` | `{{ $langs := allLanguageVariations . }}{{ range $langs }}{{ .Path }}{{ end }}` |
| `i18n`                  | Returns translated string                         | `string`     | `{{ i18n "edit" }}`                                                             |
