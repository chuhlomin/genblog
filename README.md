# GenBlog

[![Dockerhub](https://img.shields.io/badge/docker-hub-4988CC)](https://hub.docker.com/repository/docker/chuhlomin/genblog)

Generate a static blog from Markdown files.

## Inputs

| Name                      | Description                                                                        | Default                     | Required |
|---------------------------|------------------------------------------------------------------------------------|-----------------------------|----------|
| `title`                   | Title of the blog                                                                  |                             | true     |
| `short_description`       | Short description of the blog                                                      |                             | true     |
| `author`                  | Author of the blog                                                                 |                             | true     |
| `source_directory`        | Path to directory with Markdown files                                              | "."                         | false    |
| `output_directory`        | Path to output directory                                                           | "output"                    | false    |
| `templates_directory`     | Path to templates directory                                                        | "templates"                 | false    |
| `template_post`           | Filename of the template for posts                                                 | "post.html"                 | false    |
| `templates`               | Comma-separated templates filenames that will be processed in addition to posts    | "index.html,404.html"       | false    |
| `allowed_file_extensions` | Comma-separated templates extensions that will be copied as is to output directory | ".jpeg,.jpg,.png,.mp4,.pdf" | false    |
| `default_language`        | Default language of the blog                                                       | "en"                        | false    |
| `comments_enabled`        | Enable comments                                                                    | "false"                     | false    |
| `comments_site_id`        | Site ID for Remark42 comments                                                      | ""                          | false    |
| `show_drafts`             | Show drafts                                                                        | "false"                     | false    |
| `thumb_path`              | Path to thumbnails directory                                                       | "thumb"                     | false    |
| `thumb_max_width`         | Max width of thumbnails                                                            | "140"                       | false    |
| `thumb_max_height`        | Max height of thumbnails                                                           | "140"                       | false    |
| `search_enabled`          | Create `bleve` index directory                                                     | "false"                     | false    |
| `search_path`             | Path to `bleve` index directory                                                    | "index.bleve"               | false    |

Genblog scans files in the `source_directory`.

For every Markdown file it renders HTML page in the `output_directory`
using `template_post` in the `templates_directory`,
keeping the same directory structure.

For any file that have `allowed_file_extensions` it just copies it to the
`output_directory`, keeping the same directory structure.

Then Genblog renders all templates in `templates` list, passing list of all posts there.
That is useful for index page, sitemap and RSS feeds.

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

### `page`

Genblog passes the following structure to `template_post` to render
individual post pages:

| Field             | Type         | Description                                                     |
|-------------------|--------------|-----------------------------------------------------------------|
| `CurrentPage`     | `pageData`   | Current post, corresponds to a single Markdown file (see below) |
| `AllPages`        | `[]pageData` | Array of all available posts                                    |
| `DefaultLanguage` | `string`     | Site language, default to "en"                                  |
| `CommentsSiteID`  | `string`     | Site ID for the comments engine (Remark42)                      |
| `Timestamp`       | `int64`      | Unix timestamp when Genblog was started                         |

### `pageData`

`pageData` structure has these fields:

| Field      | Type       | Description                                                  |
|------------|------------|--------------------------------------------------------------|
| `Source`   | `string`   | Relative path to the source Markdown file                    |
| `Path`     | `string`   | Relative path to the generated HTML file                     |
| `ID`       | `string`   | Same post in different languages will have the same ID value |
| `Metadata` | `metadata` | Post metadata (see below)                                    |
| `Body`     | `string`   | Rendered HTML body                                           |
| `Markdown` | `string`   | Markdown file content                                        |

### `metadata`

`metadata` structure has these fields:

| Field             | Type       | Description                                                                    |
|-------------------|------------|--------------------------------------------------------------------------------|
| `Type`            | `string`   | Page type, "`post`" by default                                                 |
| `Title`           | `string`   | By default equals to `H1` in Markdown file                                     |
| `Date`            | `string`   | date when post was published, in format "2006-01-02"                           |
| `Tags`            | `[]string` | Post tags, by default parsed from the post                                     |
| `Language`        | `string`   | Language ("en", "ru", ...), parsed from filename, overrides `default_language` |
| `Slug`            | `string`   | Slug is used for the URL, by default it's the same as the file path            |
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

## Local development

```bash
make build
```

```fish
echo 'fish_add_path /path/to/repo' >> ~/.config/fish/config.fish
```
