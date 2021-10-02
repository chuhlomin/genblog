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
	Title              string `env:"INPUT_TITLE,required"`
	ShortDescription   string `env:"INPUT_SHORT_DESCRIPTION,required"`
	Author             string `env:"INPUT_AUTHOR,required"`
	SourceDirectory    string `env:"INPUT_SOURCE_DIRECTORY" envDefault:"."`
	OutputDirectory    string `env:"INPUT_OUTPUT_DIRECTORY" envDefault:"output"`
	TemplatesDirectory string `env:"INPUT_TEMPLATES_DIRECTORY" envDefault:"templates"`
	Templates          string `env:"INPUT_TEMPLATES" envDefault:"index.html,404.html"`
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
	Created  string   `yaml:"created"`
	Title    string   `yaml:"title"`
	Tags     []string `yaml:"tags"`
	Filename string   // set by code
}

type htmlPage struct {
	Path     string
	Metadata *metadata
	Body     template.HTML
}

type defaultData struct {
	htmlPage
	Posts []*metadata
}

type ByCreated []*metadata

func (c ByCreated) Len() int           { return len(c) }
func (c ByCreated) Less(i, j int) bool { return c[i].Created > c[j].Created }
func (c ByCreated) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

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

	// write posts
	postTemplate := getPostTemplate(t)
	var posts []*metadata // needed for index.html

	filesChannel := make(chan string)
	done := make(chan bool)

	go func() {
		for {
			path, more := <-filesChannel
			if more {
				if strings.HasSuffix(path, ".md") {
					m, err := convertMarkdownFile(path, c.OutputDirectory, postTemplate)
					if err != nil {
						log.Printf("ERROR processing markdown file %s: %v", path, err)
						continue
					}
					posts = append(posts, m)
				} else {
					copyFile(path, c.OutputDirectory+"/"+path)
				}
			} else {
				done <- true
				return
			}
		}
	}()

	if err := readSourceDirectory(c.SourceDirectory, filesChannel); err != nil {
		return errors.Wrap(err, "read posts directory")
	}

	close(filesChannel)
	<-done

	sort.Sort(ByCreated(posts))
	return processTemplates(t, c.Templates, c.OutputDirectory, posts)
}

func processTemplates(t *template.Template, templates, outputDir string, posts []*metadata) error {
	tmpls := strings.Split(templates, ",")

	for _, tmpl := range tmpls {
		if t.Lookup(tmpl) == nil {
			log.Printf("WARNING: template %q not found", tmpl)
			continue
		}

		err := writeFile(
			"./"+outputDir+"/"+tmpl,
			defaultData{
				htmlPage: htmlPage{
					Metadata: &metadata{
						Title: "Hello",
					},
					Body: template.HTML(``),
					Path: "",
				},
				Posts: posts,
			},
			t.Lookup(tmpl),
		)
		if err != nil {
			return errors.Wrapf(err, "write template %q", tmpl)
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

func readSourceDirectory(path string, filesChannel chan string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path != "README.md" {
			filesChannel <- path
		}
		return nil
	})
}

func convertMarkdownFile(path, outputDirectory string, tpl *template.Template) (*metadata, error) {
	b, err := ioutil.ReadFile("./" + path)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", path)
	}

	metadataBytes, bodyBytes := getMetadataAndBody(b)

	m, bodyBytes, err := buildMetadata(metadataBytes, bodyBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "build metadata %s", path)
	}

	output := markdown.ToHTML(bodyBytes, nil, nil)

	data := htmlPage{
		Metadata: m,
		Body:     template.HTML(string(output)),
		Path:     path,
	}

	m.Filename = strings.Replace(path, ".md", ".html", 1)
	filename := outputDirectory + "/" + m.Filename

	return m, writeFile(filename, data, tpl)
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
					m.Title = strings.TrimSpace(line[2:])
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
