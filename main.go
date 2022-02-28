package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	goimage "image"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/caarlos0/env/v6"
	"github.com/chuhlomin/typograph"
	"github.com/disintegration/imaging"
	"github.com/gomarkdown/markdown"
	i "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

const (
	permFile = 0644
	permDir  = 0755
)

type config struct {
	Title                    string   `env:"INPUT_TITLE,required"`
	ShortDescription         string   `env:"INPUT_SHORT_DESCRIPTION,required"`
	Author                   string   `env:"INPUT_AUTHOR,required"`
	SourceDirectory          string   `env:"INPUT_SOURCE_DIRECTORY" envDefault:"."`
	OutputDirectory          string   `env:"INPUT_OUTPUT_DIRECTORY" envDefault:"./output"`
	TemplatesDirectory       string   `env:"INPUT_TEMPLATES_DIRECTORY" envDefault:"templates"`
	TemplatePost             string   `env:"INPUT_TEMPLATE_POST" envDefault:"post.html"`
	AllowedFileExtensions    []string `env:"INPUT_ALLOWED_FILE_EXTENSIONS" envDefault:".jpeg,.jpg,.png,.mp4,.pdf" envSeparator:","`
	DefaultLanguage          string   `env:"INPUT_DEFAULT_LANGUAGE" envDefault:"en"`
	TypographyEnabled        bool     `env:"INPUT_TYPOGRAPHY_ENABLED" envDefault:"false"`
	CommentsEnabled          bool     `env:"INPUT_COMMENTS_ENABLED" envDefault:"true"`
	CommentsSiteID           string   `env:"INPUT_COMMENTS_SITE_ID" envDefault:""`
	ShowSocialSharingButtons bool     `env:"INPUT_SHOW_SOCIAL_SHARING_BUTTONS" envDefault:"false"`
	ShowDrafts               bool     `env:"INPUT_SHOW_DRAFTS" envDefault:"false"`
	ThumbPath                string   `env:"INPUT_THUMB_PATH" envDefault:"thumb"`
	ThumbMaxWidth            int      `env:"INPUT_THUMB_MAX_WIDTH" envDefault:"140"`
	ThumbMaxHeight           int      `env:"INPUT_THUMB_MAX_HEIGHT" envDefault:"140"`
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

// image is a struct that contains metadata of image from the post
type image struct {
	Path      string `yaml:"path"`
	Alt       string `yaml:"alt"`
	Title     string `yaml:"title"`
	ThumbPath string `yaml:"thumb_path"`
}

// metadata is a struct that contains metadata for a post
// it is used to render the template.
// Each markdown file may have a header wrapped by `---\n`, eg:
//    ---
//    title: "Title"
//    slug: "title"
//    date: "2021-10-24"
//    ---
//    Page content
type metadata struct {
	Type                     string  `yaml:"type"`                        // page type, by default "post"
	Title                    string  `yaml:"title"`                       // by default equals to H1 in Markdown file
	Date                     string  `yaml:"date"`                        // date when post was published, in format "2006-01-02"
	Tags                     tags    `yaml:"tags"`                        // post tags, by default parsed from the post
	Language                 string  `yaml:"language"`                    // language ("en", "ru", ...), parsed from filename, overrides config.DefaultLanguage
	Slug                     string  `yaml:"slug"`                        // slug is used for the URL, by default it's the same as the file path
	Description              string  `yaml:"description"`                 // description is used for the meta description
	Author                   string  `yaml:"author"`                      // author is used for the meta author, overrides config.Author
	Keywords                 string  `yaml:"keywords"`                    // keywords is used for the meta keywords
	Draft                    bool    `yaml:"draft"`                       // draft is used to mark post as draft
	Template                 string  `yaml:"template"`                    // template to use in config.TemplatesDirectory, overrides default "post.html"
	TypographyEnabled        *bool   `yaml:"typography_enabled"`          // typography_enabled overrides config.TypographyEnabled
	CommentsEnabled          *bool   `yaml:"comments_enabled"`            // comments_enabled overrides config.CommentsEnabled
	ShowSocialSharingButtons *bool   `yaml:"show_social_sharing_buttons"` // show_social_sharing_buttons is used to show social sharing buttons, overrides config.ShowSocialSharingButtons
	Images                   []image `yaml:"images"`                      // images in the post
}

type pageData struct {
	Source   string // path to the source markdown file
	Path     string // path to the generated HTML file
	ID       string // same post in different language will have the same ID value
	Metadata *metadata
	Body     string
}

type page struct {
	CurrentPage           *pageData
	AllPages              []*pageData
	AllLanguageVariations []*pageData // used only for index.html
	DefaultLanguage       string
	CommentsSiteID        string
}

type ByCreated []*pageData

func (c ByCreated) Len() int           { return len(c) }
func (c ByCreated) Less(i, j int) bool { return c[i].Metadata.Date > c[j].Metadata.Date }
func (c ByCreated) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type ByLanguage []*pageData

func (c ByLanguage) Len() int           { return len(c) }
func (c ByLanguage) Less(i, j int) bool { return c[i].Metadata.Language > c[j].Metadata.Language }
func (c ByLanguage) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }

type TagsCounterList map[string]int

func (tcl TagsCounterList) Add(tags []string) {
	for _, tag := range tags {
		tcl[tag]++
	}
}

var errSkipDraft = errors.New("skip draft")

var bundle *i.Bundle

func main() {
	log.Println("Starting")
	t := time.Now()

	if err := run(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	log.Printf("Finished in %dms", time.Now().Sub(t).Milliseconds())
}

func run() error {
	var c config
	err := env.Parse(&c)
	if err != nil {
		return errors.Wrap(err, "environment variables parsing")
	}
	if c.DefaultLanguage == "" {
		c.DefaultLanguage = "en"
	}

	lang, err := language.Parse(c.DefaultLanguage)
	if err != nil {
		return errors.Wrapf(err, "parse language %q", c.DefaultLanguage)
	}
	bundle = i.NewBundle(lang) // used in templates/i18n to get translated strings
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	if err = createDirectory(c.OutputDirectory); err != nil {
		return errors.Wrapf(err, "output directory creation %q", c.OutputDirectory)
	}

	t, err := template.New("").Funcs(fm).ParseGlob(c.TemplatesDirectory + "/*")
	if err != nil {
		return errors.Wrap(err, "templates parsing")
	}

	tp := t.Lookup(c.TemplatePost)
	if tp == nil {
		return errors.Errorf("template %q not found", c.TemplatePost)
	}

	// scan source directory
	var pagesData []*pageData
	tagsCounter := TagsCounterList{}

	channelFiles := make(chan string)
	channelImages := make(chan image, 100)
	doneFiles := make(chan bool)
	doneImages := make(chan bool)

	go func() {
		for {
			path, more := <-channelFiles
			if more {
				switch filepath.Ext(path) {
				case ".md":
					b, err := ioutil.ReadFile(c.SourceDirectory + "/" + path)
					if err != nil {
						log.Printf("ERROR read file %q: %v", c.SourceDirectory+"/"+path, err)
						continue
					}

					p, err := process(b, c, path)
					if p == nil {
						log.Printf("ERROR failed to process file: %q", c.SourceDirectory+"/"+path)
						continue
					}

					for _, image := range p.Metadata.Images {
						channelImages <- image
					}

					if p.Metadata.Language == c.DefaultLanguage {
						tagsCounter.Add(p.Metadata.Tags)
					}

					if err != nil {
						if err == errSkipDraft {
							log.Printf("DEBUG skipping draft %v", path)
							continue
						}
						log.Printf("ERROR processing markdown file %s: %v", path, err)
						continue
					}
					pagesData = append(pagesData, p)
				case ".toml":
					_, err := bundle.LoadMessageFile(c.SourceDirectory + "/" + path)
					if err != nil {
						log.Printf("ERROR load message file %q: %v", c.SourceDirectory+"/"+path, err)
					}
				default:
					copyFile(
						c.SourceDirectory+"/"+path,
						c.OutputDirectory+"/"+path,
					)
				}
			} else {
				doneFiles <- true
				return
			}
		}
	}()

	go func() {
		for {
			img, more := <-channelImages
			if more {
				if err := resizeImage(
					c.SourceDirectory,
					img.Path,
					img.ThumbPath,
					c.ThumbMaxWidth,
					c.ThumbMaxHeight,
				); err != nil {
					log.Printf("ERROR resize image %q: %v", img.Path, err)
				}
			} else {
				doneImages <- true
				return
			}
		}
	}()

	if err := readSourceDirectory(
		c.SourceDirectory,
		c.AllowedFileExtensions,
		channelFiles,
	); err != nil {
		return errors.Wrap(err, "read posts directory")
	}

	log.Println("DEBUG closing channelFiles")
	close(channelFiles)

	<-doneFiles
	close(channelImages)

	sort.Sort(ByCreated(pagesData))

	if err = renderPages(pagesData, c, tp); err != nil {
		return errors.Wrap(err, "rendering pages")
	}

	printTagsStags(tagsCounter)

	if err := renderTemplates(t, c, pagesData); err != nil {
		return errors.Wrap(err, "rendering templates")
	}

	<-doneImages
	return nil
}

func getImageFromURL(url string) (goimage.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "get image from url")
	}
	defer resp.Body.Close()

	img, _, err := goimage.Decode(resp.Body)
	return img, err
}

func resizeImage(srcDir, path, thumbPath string, maxWidth, maxHeight int) error {
	var (
		img goimage.Image
		err error
	)

	// read image
	if isValidURL(path) {
		img, err = getImageFromURL(path)
		if err != nil {
			return errors.Wrapf(err, "get image from url %q", path)
		}
	} else {
		img, err = imaging.Open(srcDir+"/"+path, imaging.AutoOrientation(true))
		if err != nil {
			return errors.Wrapf(err, "read image %q", path)
		}
	}

	// resize image
	img = imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)

	// get directory path from thumbPath
	dirPath := filepath.Dir(thumbPath)
	if err := createDirectory(dirPath); err != nil {
		return errors.Wrapf(err, "create directory %q", dirPath)
	}

	// save image
	if err := imaging.Save(img, thumbPath); err != nil {
		return errors.Wrapf(err, "save image %q", thumbPath)
	}

	return nil
}

func renderPages(pagesData []*pageData, c config, defaultTmpl *template.Template) error {
	for _, p := range pagesData {
		tmpl := defaultTmpl
		if p.Metadata.Template != "" {
			tmpl = defaultTmpl.Lookup(p.Metadata.Template)
			if tmpl == nil {
				return errors.Errorf("template %q not found", p.Metadata.Template)
			}
		}

		if err := renderTemplate(
			c.OutputDirectory+"/"+p.Path,
			page{
				CurrentPage:     p,
				AllPages:        pagesData,
				DefaultLanguage: c.DefaultLanguage,
				CommentsSiteID:  c.CommentsSiteID,
			},
			tmpl,
		); err != nil {
			return errors.Wrapf(err, "rendering page %q", p.Path)
		}
	}
	return nil
}

func renderTemplates(t *template.Template, c config, pagesData []*pageData) error {
	pagesIDMap := make(map[string][]*pageData)

	// group pages by IP
	for _, tpl := range t.Templates() {
		// if template starts with underscore
		// or if template name is empty (root)
		// or it is a "post" template – skip it
		if strings.HasPrefix(tpl.Name(), "_") || tpl.Name() == "" || tpl.Name() == c.TemplatePost {
			continue
		}

		id, lang := getLanguageFromFilename(tpl.Name())
		if lang == "" {
			lang = c.DefaultLanguage
		}

		pagesIDMap[id] = append(pagesIDMap[id], &pageData{
			ID:   id,
			Path: tpl.Name(),
			Metadata: &metadata{
				Language: lang,
			},
		})
	}

	for _, pages := range pagesIDMap {
		sort.Sort(ByLanguage(pages))

		for _, p := range pages {

			tmpl := t.Lookup(p.Path)
			if tmpl == nil {
				log.Printf("WARNING: template %q not found", p.Path)
				continue
			}

			err := renderTemplate(
				c.OutputDirectory+"/"+p.Path,
				page{
					CurrentPage:           p,
					AllPages:              pagesData,
					AllLanguageVariations: pages,
				},
				tmpl,
			)
			if err != nil {
				return errors.Wrapf(err, "write template %q", p.Path)
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, permDir); err != nil {
		return errors.Wrapf(err, "create directories for file %s", dst)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

func createDirectory(name string) error {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return os.Mkdir(name, permDir)
	}

	return nil
}

func readSourceDirectory(path string, allowedExtensions []string, filesChannel chan string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(path, "output") {
			return nil
		}

		ext := filepath.Ext(path)

		if (ext == ".md" && path != "README.md") || inArray(allowedExtensions, ext) {
			filesChannel <- path
		}
		if ext == ".toml" {
			filesChannel <- path
		}
		return nil
	})
}

func inArray(s []string, needle string) bool {
	for _, s := range s {
		if s == needle {
			return true
		}
	}
	return false
}

// process parses markdown file and returns pageData
func process(b []byte, c config, source string) (*pageData, error) {
	path := strings.Replace(source, ".md", ".html", 1)
	baseDir := filepath.Dir(path)
	metadataBytes, bodyBytes := getMetadataAndBody(b)

	m, bodyBytes, err := buildMetadata(metadataBytes, bodyBytes, baseDir, c.ThumbPath)
	if err != nil {
		return nil, errors.Wrapf(err, "build metadata %s", source)
	}

	if m.Draft && !c.ShowDrafts {
		return nil, errSkipDraft
	}

	id := source

	if m.Language == "" {
		// if file ends with _en.md, use en as language
		id, m.Language = getLanguageFromFilename(source)
	}
	if m.Language == "" {
		m.Language = c.DefaultLanguage
	}
	m.Language = strings.ToLower(m.Language)

	if m.CommentsEnabled == nil {
		m.CommentsEnabled = &c.CommentsEnabled
	}

	bodyBytes = markdown.ToHTML(bodyBytes, nil, nil)

	if m.TypographyEnabled == nil {
		m.TypographyEnabled = &c.TypographyEnabled
	}
	if m.TypographyEnabled != nil && *m.TypographyEnabled {
		bodyBytes = typograph.NewTypograph().Process(bodyBytes)
	}

	return &pageData{
		ID:       id,
		Path:     path,
		Source:   source,
		Metadata: m,
		Body:     string(bodyBytes),
	}, nil
}

func getMetadataAndBody(b []byte) ([]byte, []byte) {
	if bytes.HasPrefix(b, []byte("---")) {
		if parts := bytes.SplitN(b, []byte("---"), 3); len(parts) == 3 {
			return parts[1], parts[2]
		}
	}

	return []byte{}, b
}

func buildMetadata(
	metadataBytes []byte,
	bodyBytes []byte,
	relativePath string,
	thumbPath string,
) (*metadata, []byte, error) {
	m := metadata{
		Tags: tags([]string{}), // setting default value, so that there is no need to check for nil in templates
	}
	if len(metadataBytes) != 0 {
		err := yaml.Unmarshal(metadataBytes, &m)
		if err != nil {
			return nil, bodyBytes, errors.Wrapf(err, "reading metadata")
		}
	}

	return grabMetadata(m, bodyBytes, relativePath, thumbPath)
}

var (
	imageMarkdown  = regexp.MustCompile(`!\[(.*?)\]\(([^\s)]*)\s*"?([^"]*?)?"?\)`)
	imageHTML      = regexp.MustCompile(`<img(.*?)>`)
	htmlAttributes = regexp.MustCompile(`(\S+)\s*=\s*\"?(.*?)\"`)
)

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

func fixPath(path, relativePath, thumbPath string) (string, string) {
	if !isValidURL(path) {
		return relativePath + "/" + path,
			thumbPath + "/" + path
	}
	// sha1 hash of the url
	h := sha1.New()
	h.Write([]byte(path))
	hash := hex.EncodeToString(h.Sum(nil))

	// get path file extension
	ext := filepath.Ext(path)

	return path, thumbPath + "/" + hash + "." + ext
}

func grabMetadata(
	m metadata,
	b []byte,
	relativePath string,
	thumbPath string,
) (*metadata, []byte, error) {
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

			m.Title = htmlTitle
			hasHeader = true
			continue // so that we don't leave header in the body
		}

		// parse tags
		if bytes.HasPrefix(scanned, []byte("#")) && !hasTags {
			line := scanner.Text()
			m.Tags = strings.Split(strings.TrimSpace(line), " ")
			for i, tag := range m.Tags {
				m.Tags[i] = strings.Trim(tag, "#,")
			}
			hasTags = true
			continue // so that we don't leave tags in the body
		}

		// parse markdown images
		if matches := imageMarkdown.FindAllStringSubmatch(scanner.Text(), -1); matches != nil {
			for _, match := range matches {
				path, thumbPath := fixPath(match[2], relativePath, thumbPath)
				m.Images = append(m.Images, image{
					Path:      path,
					Alt:       match[1],
					Title:     match[3],
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

				m.Images = append(m.Images, img)
			}
		}

		buf.Write(scanned)
		buf.WriteString("\n")
	}
	b = buf.Bytes()

	return &m, b, nil
}

func getLanguageFromFilename(filename string) (newFilename, lang string) {
	underscoreIndex := strings.LastIndex(filename, "_")
	if underscoreIndex == -1 {
		return filename, ""
	}

	dotIndex := strings.LastIndex(filename, ".")
	if dotIndex == -1 {
		return filename, ""
	}

	lang = filename[underscoreIndex+1 : dotIndex]
	newFilename = filename[0:underscoreIndex] + filename[dotIndex:]
	return
}

func printTagsStags(tagsCounter TagsCounterList) {
	log.Println("Tags counts:")
	p := make(PairList, len(tagsCounter))
	i := 0
	for k, v := range tagsCounter {
		p[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(p))

	for _, k := range p {
		log.Printf("  %v: %v", k.Key, k.Value)
	}
}
