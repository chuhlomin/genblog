package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
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

	if err = createOutputDirectory(c.OutputDirectory); err != nil {
		return errors.Wrap(err, "output directory creation")
	}

	if err = createRobotsTxt(c.OutputDirectory, c.RobotsDisallow); err != nil {
		return errors.Wrap(err, "robots.txt creation")
	}

	if err = createIndexHTML(c.OutputDirectory); err != nil {
		return errors.Wrap(err, "index.html creation")
	}

	return nil
}

func createOutputDirectory(name string) error {
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
