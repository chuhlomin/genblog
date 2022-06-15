# GenBlog

[![Dockerhub](https://img.shields.io/badge/docker-hub-4988CC)](https://hub.docker.com/repository/docker/chuhlomin/genblog)

Generate a static blog from Markdown files

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
| `comments_enabled`        | Enable typography                                                                  | "false"                     | false    |
| `comments_site_id`        | Site ID for Remark42 comments                                                      | ""                          | false    |
| `show_drafts`             | Show drafts                                                                        | "false"                     | false    |
| `thumb_path`              | Path to thumbnails directory                                                       | "thumb"                     | false    |
| `thumb_max_width`         | Max width of thumbnails                                                            | "140"                       | false    |
| `thumb_max_height`        | Max height of thumbnails                                                           | "140"                       | false    |

## Local development

```bash
make build
```

```fish
echo 'fish_add_path /path/to/repo' >> ~/.config/fish/config.fish
```
