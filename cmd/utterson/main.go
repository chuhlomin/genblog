package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/gomarkdown/markdown"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	defaultIndexTemplate = `<p>Hello</p>
`

	defaultPostTemplate = `
<!DOCTYPE html>
<html>
<head>
<title>{{.Metadata.Title}}</title>
</head>
<body>
{{.Body}}
</body>
</html>
`

	postTemplate = "templates/post.html"
)

const (
	permFile = 0644
	permDir  = 0755
)

type config struct {
	Title                      string `env:"PLUGIN_TITLE,required"`
	ShortDescription           string `env:"PLUGIN_SHORT_DESCRIPTION,required"`
	Author                     string `env:"PLUGIN_AUTHOR,required"`
	Template                   string `env:"PLUGIN_TEMPLATE" envDefault:"acute"`
	Timezone                   string `env:"PLUGIN_TIMEZONE" envDefault:"America/New_York"`
	Encoding                   string `env:"PLUGIN_ENCODING" envDefault:"utf-8"`
	Language                   string `env:"PLUGIN_LANGUAGE" envDefault:"en"`
	ShowDrafts                 bool   `env:"PLUGIN_SHOW_DRAFTS"`
	Future                     bool   `env:"PLUGIN_FUTURE"`
	PostsPerPage               int    `env:"PLUGIN_POSTS_PER_PAGE" envDefault:"10"`
	DisableTypography          bool   `env:"PLUGIN_DISABLE_TYPOGRAPHY"`
	ShowSocialSharingButtons   bool   `env:"PLUGIN_SHOW_SOCIAL_SHARING_BUTTONS"`
	CommentsAllowByDefault     bool   `env:"PLUGIN_COMMENTS_ALLOW_BY_DEFAULT" envDefault:"true"`
	CommandsOnlyForRecentPosts bool   `env:"PLUGIN_COMMANDS_ONLY_FOR_RECENT_POSTS"`
	CommentsSendByEmail        string `env:"PLUGIN_COMMENTS_SEND_BY_EMAIL"`
	YandexMetrika              string `env:"PLUGIN_YANDEX_METRIKA"`
	GoogleAnalytics            string `env:"PLUGIN_GOOGLE_ANALYTICS"`
	RobotsDisallow             bool   `env:"PLUGIN_ROBOTS_DISALLOW"`
	PostsDirectory             string `env:"PLUGIN_POSTS_DIRECTORY" envDefault:"posts"`
	OutputDirectory            string `env:"PLUGIN_OUTPUT_DIRECTORY" envDefault:"html"`
}

type metadata struct {
	Title string   `yaml:"title"`
	Tags  []string `yaml:"tags"`
}

type postData struct {
	Metadata metadata
	Body     template.HTML
}

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
		return errors.Wrapf(err, "output directory creation (%s)", c.OutputDirectory)
	}

	if err = createDirectory(c.OutputDirectory + "/posts"); err != nil {
		return errors.Wrap(err, "posts directory creation")
	}

	if err = createIndexHTML(c.OutputDirectory); err != nil {
		return errors.Wrap(err, "index.html creation")
	}

	markdownChannel := make(chan string)
	errorChannel := make(chan error)
	done := make(chan bool)

	t := getPostTemplate()

	go func() {
		for {
			path, more := <-markdownChannel
			if more {
				if err := convertMarkdownFile(path, c.OutputDirectory, t); err != nil {
					log.Printf("ERROR processing markdown file %s: %v", path, err)
					continue
				}
			} else {
				close(errorChannel)
				done <- true
				return
			}
		}
	}()

	if err := readPostsDirectory(c.PostsDirectory, markdownChannel); err != nil {
		return errors.Wrap(err, "read posts directory")
	}

	close(markdownChannel)
	<-done

	return nil
}

func createDirectory(name string) error {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return os.Mkdir(name, permDir)
	}

	return nil
}

func createIndexHTML(path string) error {
	return ioutil.WriteFile("./"+path+"/index.html", []byte(defaultIndexTemplate), 0644)
}

func readPostsDirectory(path string, markdownChannel chan string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".md") {
			markdownChannel <- path
		}
		return nil
	})
}

func convertMarkdownFile(path string, outputDirectory string, t *template.Template) error {
	b, err := ioutil.ReadFile("./" + path)
	if err != nil {
		return errors.Wrapf(err, "read file %s", path)
	}

	metadataBytes, bodyBytes := getMetadataAndBody(b)

	metadata := metadata{}
	err = yaml.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return errors.Wrapf(err, "reading metadata")
	}

	output := markdown.ToHTML(bodyBytes, nil, nil)

	data := postData{
		Metadata: metadata,
		Body:     template.HTML(string(output)),
	}

	filename := "./" + outputDirectory + "/" + strings.Replace(path, ".md", ".html", 1)

	return writeFile(filename, data, t)
}

func getMetadataAndBody(b []byte) ([]byte, []byte) {
	if bytes.HasPrefix(b, []byte("---")) {
		if parts := bytes.SplitN(b, []byte("---"), 3); len(parts) == 3 {
			return parts[1], parts[2]
		}
	}

	return []byte{}, b
}

func writeFile(filename string, data postData, t *template.Template) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
	if err != nil {
		return errors.Wrapf(err, "creating %s", filename)
	}
	defer f.Close()

	if err = t.Execute(f, data); err != nil {
		return errors.Wrap(err, "template execution")
	}

	return nil
}

func getPostTemplate() *template.Template {
	if _, err := os.Stat(postTemplate); os.IsNotExist(err) {
		return template.Must(template.New("post").Parse(defaultPostTemplate))
	}

	return template.Must(template.ParseFiles(postTemplate))
}
