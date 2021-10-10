package main

import (
	"bufio"
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/gomarkdown/markdown"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	permFile = 0644
	permDir  = 0755
)

type config struct {
	Title                 string   `env:"INPUT_TITLE,required"`
	ShortDescription      string   `env:"INPUT_SHORT_DESCRIPTION,required"`
	Author                string   `env:"INPUT_AUTHOR,required"`
	SourceDirectory       string   `env:"INPUT_SOURCE_DIRECTORY" envDefault:"."`
	OutputDirectory       string   `env:"INPUT_OUTPUT_DIRECTORY" envDefault:"./output"`
	TemplatesDirectory    string   `env:"INPUT_TEMPLATES_DIRECTORY" envDefault:"templates"`
	Templates             []string `env:"INPUT_TEMPLATES" envDefault:"index.html,404.html" envSeparator:","`
	AllowedFileExtensions []string `env:"INPUT_ALLOWED_FILE_EXTENSIONS" envDefault:".jpeg,.jpg,.png,.mp4,.pdf" envSeparator:","`
	DefaultLanguage       string   `env:"INPUT_DEFAULT_LANGUAGE" envDefault:"en"`
	// Template                   string `env:"INPUT_TEMPLATE" envDefault:"acute"`
	// Timezone                   string `env:"INPUT_TIMEZONE" envDefault:"America/New_York"`
	// Encoding                   string `env:"INPUT_ENCODING" envDefault:"utf-8"`
	// Language                   string `env:"INPUT_LANGUAGE" envDefault:"en"`
	// ShowDrafts                 bool   `env:"INPUT_SHOW_DRAFTS"`
	// Future                     bool   `env:"INPUT_FUTURE"`
	// PostsPerPage               int    `env:"INPUT_POSTS_PER_PAGE" envDefault:"10"`
	// DisableTypography          bool   `env:"INPUT_DISABLE_TYPOGRAPHY"`
	// ShowSocialSharingButtons   bool   `env:"INPUT_SHOW_SOCIAL_SHARING_BUTTONS"`
	// CommentsAllowByDefault     bool   `env:"INPUT_COMMENTS_ALLOW_BY_DEFAULT" envDefault:"true"`
	// CommandsOnlyForRecentPosts bool   `env:"INPUT_COMMANDS_ONLY_FOR_RECENT_POSTS"`
	// CommentsSendByEmail        string `env:"INPUT_COMMENTS_SEND_BY_EMAIL"`
	// YandexMetrika              string `env:"INPUT_YANDEX_METRIKA"`
	// GoogleAnalytics            string `env:"INPUT_GOOGLE_ANALYTICS"`
	// RobotsDisallow             bool   `env:"INPUT_ROBOTS_DISALLOW"`
}

type metadata struct {
	Created  string        `yaml:"created"`
	Title    template.HTML `yaml:"title"`
	Tags     []string      `yaml:"tags"`
	Language string        `yaml:"language"`
}

type pageData struct {
	ID       string // same post in different language will have the same ID value
	Path     string
	Metadata *metadata
	Body     template.HTML
}

type page struct {
	CurrentPage           *pageData
	AllPages              []*pageData
	AllLanguageVariations []*pageData // used only for index.html
}

type ByCreated []*pageData

func (c ByCreated) Len() int           { return len(c) }
func (c ByCreated) Less(i, j int) bool { return c[i].Metadata.Created > c[j].Metadata.Created }
func (c ByCreated) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type ByLanguage []*pageData

func (c ByLanguage) Len() int           { return len(c) }
func (c ByLanguage) Less(i, j int) bool { return c[i].Metadata.Language > c[j].Metadata.Language }
func (c ByLanguage) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

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

	if err = createDirectory(c.OutputDirectory); err != nil {
		return errors.Wrapf(err, "output directory creation %q", c.OutputDirectory)
	}

	t, err := template.New("templates").Funcs(fm).ParseGlob(c.TemplatesDirectory + "/*")
	if err != nil {
		return errors.Wrap(err, "templates parsing")
	}

	// scan source directory
	var pagesData []*pageData

	filesChannel := make(chan string)
	done := make(chan bool)

	go func() {
		for {
			path, more := <-filesChannel
			if more {
				if strings.HasSuffix(path, ".md") {
					p, err := convertMarkdownFile(
						path,
						c.SourceDirectory,
						c.OutputDirectory,
						c.DefaultLanguage,
					)
					if err != nil {
						log.Printf("ERROR processing markdown file %s: %v", path, err)
						continue
					}
					pagesData = append(pagesData, p)
				} else {
					copyFile(
						c.SourceDirectory+"/"+path,
						c.OutputDirectory+"/"+path,
					)
				}
			} else {
				done <- true
				return
			}
		}
	}()

	if err := readSourceDirectory(
		c.SourceDirectory,
		c.AllowedFileExtensions,
		filesChannel,
	); err != nil {
		return errors.Wrap(err, "read posts directory")
	}

	close(filesChannel)
	<-done

	sort.Sort(ByCreated(pagesData))

	if err = renderPages(pagesData, c.OutputDirectory, t.Lookup("post.html")); err != nil {
		return errors.Wrap(err, "rendering pages")
	}

	return renderTemplates(t, c.Templates, c.OutputDirectory, pagesData, c.DefaultLanguage)
}

func renderPages(pagesData []*pageData, outputDirectory string, tmpl *template.Template) error {
	for _, p := range pagesData {
		if err := renderTemplate(
			outputDirectory+"/"+p.Path,
			page{
				CurrentPage: p,
				AllPages:    pagesData,
			},
			tmpl,
		); err != nil {
			return errors.Wrapf(err, "rendering page %q", p.Path)
		}
	}
	return nil
}

func renderTemplates(
	t *template.Template,
	templates []string,
	outputDir string,
	pagesData []*pageData,
	defaultLanguage string,
) error {
	customPages := make(map[string][]*pageData)

	for _, tmplPath := range templates {
		id, lang := getLanguageFromFilename(tmplPath)
		if lang == "" {
			lang = defaultLanguage
		}

		customPages[id] = append(customPages[id], &pageData{
			ID:   id,
			Path: tmplPath,
			Metadata: &metadata{
				Language: lang,
			},
		})
	}

	for _, pages := range customPages {
		sort.Sort(ByLanguage(pages))

		for _, p := range pages {

			tmpl := t.Lookup(p.Path)
			if tmpl == nil {
				log.Printf("WARNING: template %q not found", p.Path)
				continue
			}

			err := renderTemplate(
				outputDir+"/"+p.Path,
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

func convertMarkdownFile(path, source, output, defaultLanguage string) (*pageData, error) {
	b, err := ioutil.ReadFile(source + "/" + path)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", path)
	}

	metadataBytes, bodyBytes := getMetadataAndBody(b)

	m, bodyBytes, err := buildMetadata(metadataBytes, bodyBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "build metadata %s", path)
	}

	id := path

	if m.Language == "" {
		// if file ends with _en.md, use en as language
		id, m.Language = getLanguageFromFilename(path)
	}
	if m.Language == "" {
		m.Language = defaultLanguage
	}
	m.Language = strings.ToLower(m.Language)

	path = strings.Replace(path, ".md", ".html", 1)

	htmlBody := markdown.ToHTML(bodyBytes, nil, nil)

	return &pageData{
		ID:       id,
		Path:     path,
		Metadata: m,
		Body:     template.HTML(string(htmlBody)),
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

func buildMetadata(metadataBytes []byte, bodyBytes []byte) (*metadata, []byte, error) {
	m := metadata{}
	if len(metadataBytes) != 0 {
		err := yaml.Unmarshal(metadataBytes, &m)
		if err != nil {
			return nil, bodyBytes, errors.Wrapf(err, "reading metadata")
		}
	}

	return grabMetadata(m, bodyBytes)
}

func grabMetadata(m metadata, b []byte) (*metadata, []byte, error) {
	b = bytes.TrimSpace(b)

	if bytes.HasPrefix(b, []byte("#")) {
		buf := bytes.Buffer{}
		seenHeader := false

		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			if !seenHeader {
				line := scanner.Text()
				if strings.HasPrefix(line, "# ") {
					htmlTitle := string(markdown.ToHTML([]byte(strings.TrimSpace(line[2:])), nil, nil))
					htmlTitle = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(htmlTitle), "<p>"), "</p>")
					m.Title = template.HTML(htmlTitle)
					seenHeader = true
					continue
				}
			}

			buf.Write(scanner.Bytes())
			buf.WriteString("\n")
		}
		b = buf.Bytes()
	}

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
