package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v6"
	"github.com/gomarkdown/markdown"
	"github.com/pkg/errors"
)

const defaultRobotsTxtDisallow = `User-agent: *
Disallow: /
`

const defaultIndexHTML = `<p>Hello</p>
`

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

func main() {
	log.Println("Starting...")

	if err := run(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	log.Println("Stopped")
}

func run() error {
	var c config
	err := env.Parse(&c)
	if err != nil {
		return errors.Wrap(err, "environment variables parsing")
	}

	if err = createDirectory(c.OutputDirectory); err != nil {
		return errors.Wrap(err, "output directory creation")
	}

	if err = createDirectory(c.OutputDirectory + "/posts"); err != nil {
		return errors.Wrap(err, "posts directory creation")
	}

	if err = createRobotsTxt(c.OutputDirectory, c.RobotsDisallow); err != nil {
		return errors.Wrap(err, "robots.txt creation")
	}

	if err = createIndexHTML(c.OutputDirectory); err != nil {
		return errors.Wrap(err, "index.html creation")
	}

	markdownChannel := make(chan string)
	errorChannel := make(chan error)
	done := make(chan bool)

	go func() {
		for {
			path, more := <-markdownChannel
			if more {
				if err := convertMarkdownFile(path, c.OutputDirectory); err != nil {
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
		return os.Mkdir(name, 0755)
	}

	return nil
}

func createRobotsTxt(path string, disallow bool) error {
	return ioutil.WriteFile(
		"./"+path+"/robots.txt",
		[]byte(getRobotsTxtContent(disallow)),
		00644,
	)
}

func getRobotsTxtContent(disallow bool) string {
	if disallow {
		return defaultRobotsTxtDisallow
	}

	return ""
}

func createIndexHTML(path string) error {
	return ioutil.WriteFile("./"+path+"/index.html", []byte(defaultIndexHTML), 0644)
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

func convertMarkdownFile(path string, outputDirectory string) error {
	b, err := ioutil.ReadFile("./" + path)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", path)
	}

	if bytes.HasPrefix(b, []byte("---")) {
		if parts := bytes.SplitN(b, []byte("---"), 3); len(parts) == 3 {
			_ = parts[1]
			b = parts[2]
		}
	}

	output := markdown.ToHTML(b, nil, nil)

	return ioutil.WriteFile(
		"./"+outputDirectory+"/"+strings.Replace(path, ".md", ".html", 1),
		output,
		0644,
	)
}
