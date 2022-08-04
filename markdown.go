package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// MarkdownFile represents a markdown file, for example
//    ---
//    date: "2021-10-24"
//    ---
//    # Title
//    Page content
type MarkdownFile struct {
	Source          string  // path to the source markdown file
	Path            string  // path to the generated HTML file
	Canonical       string  // canonical URL
	ID              string  // same post in different languages will have the same ID value
	Markdown        string  `indexer:"text"`                    // content of the markdown file
	Title           string  `yaml:"title" indexer:"text"`       // by default equals to H1 in Markdown file
	Body            string  `indexer:"no_store"`                // html body, generated from markdown
	Date            string  `yaml:"date" indexer:"date"`        // date when post was published, in format "2006-01-02"
	ContentType     string  `yaml:"type"`                       // "post" (by default), "page", etc.
	Tags            tags    `yaml:"tags"`                       // post tags, by default parsed from the post
	Language        string  `yaml:"language"`                   // language ("en", "ru", ...), parsed from filename, overrides config.DefaultLanguage
	Description     string  `yaml:"description" indexer:"text"` // description is used for the meta description
	Author          string  `yaml:"author"`                     // author is used for the meta author
	Keywords        string  `yaml:"keywords"`                   // keywords is used for the meta keywords
	Draft           bool    `yaml:"draft"`                      // draft is used to mark post as draft
	Template        string  `yaml:"template"`                   // template to use in config.TemplatesDirectory, overrides default "post.html"
	CommentsEnabled *bool   `yaml:"comments_enabled"`           // comments_enabled overrides config.CommentsEnabled
	Image           string  `yaml:"image"`                      // image associated with the post; it's used to generate the thumbnailPath
	Images          []image // images in the post
}

type ByCreated []*MarkdownFile

func (c ByCreated) Len() int           { return len(c) }
func (c ByCreated) Less(i, j int) bool { return c[i].Date > c[j].Date }
func (c ByCreated) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type ByLanguage []*MarkdownFile

func (c ByLanguage) Len() int           { return len(c) }
func (c ByLanguage) Less(i, j int) bool { return c[i].Language > c[j].Language }
func (c ByLanguage) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

// image is a struct that contains metadata of image from the post
type image struct {
	Path      string `yaml:"path"`
	Alt       string `yaml:"alt"`
	Title     string `yaml:"title"`
	ThumbPath string `yaml:"thumb_path"`
	Promo     bool   `yaml:"promo"`
}

type tags []string

func (t *tags) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	*t = strings.Split(s, ", ")
	return nil
}

func (md MarkdownFile) Type() string {
	suffix := ""
	if md.Language != cfg.DefaultLanguage {
		suffix = "_" + md.Language
	}
	return md.ContentType + suffix
}

func ParseMarkdownFile(path string) (*MarkdownFile, error) {
	content, err := ioutil.ReadFile(cfg.SourceDirectory + "/" + path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read markdown file")
	}

	return processMarkdownFileContent(path, content)
}

func processMarkdownFileContent(path string, content []byte) (*MarkdownFile, error) {
	outputPath := strings.Replace(path, ".md", ".html", 1)

	md := &MarkdownFile{
		Source:      path,
		Path:        outputPath,
		Canonical:   langToGetParameter(outputPath),
		Tags:        tags([]string{}), // setting default value, so that there is no need to check for nil in templates
		ContentType: "post",           // default value, may be overridden by metadata
	}

	md.ID = path
	if md.Language == "" {
		md.ID, md.Language = getIDAndLangFromFilename(path)
	}

	if md.Language == "" {
		md.Language = cfg.DefaultLanguage
	}
	md.Language = strings.ToLower(md.Language)

	if md.CommentsEnabled == nil {
		md.CommentsEnabled = &cfg.CommentsEnabled
	}

	bodyBytes, err := md.processContent(content)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process markdown file")
	}

	md.Markdown = string(bodyBytes)

	bodyBytes = markdown.ToHTML(bodyBytes, nil, nil)

	// todo: add typograph here :typograph.NewTypograph().Process(bodyBytes)

	md.Body = string(bodyBytes)

	return md, nil
}

func (md *MarkdownFile) processContent(content []byte) ([]byte, error) {
	metadataBytes, bodyBytes := splitMetadataAndBody(content)

	if err := md.parseMetadata(metadataBytes); err != nil {
		return nil, errors.Wrapf(err, "parsing metadata")
	}

	baseDir := filepath.Dir(md.Source)
	relativePath := baseDir
	thumbPath := cfg.ThumbPath + "/" + baseDir

	if md.Image != "" {
		// Add image to the list of all images
		path, thumbPath := fixPath(md.Image, relativePath, thumbPath)
		md.Images = append(md.Images, image{
			Path:      path,
			ThumbPath: thumbPath,
		})
	}

	bodyBytes = md.processBody(bodyBytes, relativePath, thumbPath)

	return bodyBytes, nil
}

// parseMetadata parses YAML metadata at the beginning of the Markdown file.
// For convinience, it puts all metadata into MarkdownFile struct.
func (md *MarkdownFile) parseMetadata(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	if err := yaml.Unmarshal(b, &md); err != nil {
		return errors.Wrapf(err, "reading metadata")
	}

	return nil
}

var (
	imageMarkdown  = regexp.MustCompile(`!\[(.*?)\]\(([^\s)]*)\s*"?([^"]*?)?"?\)`)
	imageHTML      = regexp.MustCompile(`<img(.*?)>`)
	htmlAttributes = regexp.MustCompile(`(\S+)\s*=\s*\"?(.*?)\"`)
)

// processBody parses Title, Tags and Images from Markdown file content.
// It removes Title and Tags from the content,
// so that they are not processed by markdown.ToHTML call.
// relativePath and thumbPath are needed for image processing and subject to refactoring.
func (md *MarkdownFile) processBody(b []byte, relativePath, thumbPath string) []byte {
	b = bytes.TrimSpace(b)

	buf := bytes.Buffer{}
	hasHeader := false
	hasTags := false

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		scanned := scanner.Bytes()

		// parse header
		if bytes.HasPrefix(scanned, []byte("# ")) && !hasHeader {
			line := scanner.Text()
			htmlTitle := string(markdown.ToHTML([]byte(strings.TrimSpace(line[2:])), nil, nil))
			htmlTitle = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(htmlTitle), "<p>"), "</p>")

			md.Title = htmlTitle
			hasHeader = true
			continue // so that we don't leave header in the body
		}

		// parse tags
		if bytes.HasPrefix(scanned, []byte("#")) && !hasTags {
			line := scanner.Text()
			md.Tags = strings.Split(strings.TrimSpace(line), " ")
			for i, tag := range md.Tags {
				md.Tags[i] = strings.Trim(tag, "#,")
			}
			hasTags = true
			continue // so that we don't leave tags in the body
		}

		// parse markdown images
		if matches := imageMarkdown.FindAllStringSubmatch(scanner.Text(), -1); matches != nil {
			for _, match := range matches {
				alt, url, title := match[1], match[2], match[3]

				path, thumbPath := fixPath(url, relativePath, thumbPath)
				md.Images = append(md.Images, image{
					Path:      path,
					Alt:       alt,
					Title:     title,
					ThumbPath: thumbPath,
				})
			}
		}

		// parse HTML images
		if matches := imageHTML.FindAllStringSubmatch(scanner.Text(), -1); matches != nil {
			for _, match := range matches {
				img := image{}
				attributes := htmlAttributes.FindAllStringSubmatch(match[1], -1)
				for _, attr := range attributes {
					switch attr[1] {
					case "src":
						path, thumbPath := fixPath(attr[2], relativePath, thumbPath)
						img.Path = path
						img.ThumbPath = thumbPath
					case "alt":
						img.Alt = attr[2]
					case "title":
						img.Title = attr[2]
					}
				}

				md.Images = append(md.Images, img)
			}
		}

		buf.Write(scanned)
		buf.WriteString("\n")
	}

	return buf.Bytes()
}

func splitMetadataAndBody(b []byte) ([]byte, []byte) {
	if bytes.HasPrefix(b, []byte("---")) {
		if parts := bytes.SplitN(b, []byte("---"), 3); len(parts) == 3 {
			return parts[1], parts[2]
		}
	}

	return []byte{}, b
}

func isValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

func fixPath(url, relativePath, thumbPath string) (string, string) {
	if !isValidURL(url) {
		return path.Clean(relativePath + "/" + url),
			path.Clean(thumbPath + "/" + url)
	}
	// sha1 hash of the url
	h := sha1.New()
	h.Write([]byte(url))
	hash := hex.EncodeToString(h.Sum(nil))

	// get path file extension
	ext := filepath.Ext(url)

	return url, thumbPath + "/" + hash + ext
}

func getIDAndLangFromFilename(filename string) (newFilename, lang string) {
	underscoreIndex := strings.LastIndex(filename, "_")
	if underscoreIndex == -1 {
		return filename, ""
	}

	dotIndex := strings.LastIndex(filename, ".")
	if dotIndex == -1 {
		return filename, ""
	}

	lang = filename[underscoreIndex+1 : dotIndex]
	if len(lang) != 2 {
		return filename, ""
	}

	newFilename = filename[0:underscoreIndex] + filename[dotIndex:]
	return
}
