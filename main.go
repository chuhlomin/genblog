package main

import (
	goimage "image"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/caarlos0/env/v6"
	"github.com/chuhlomin/search"
	"github.com/disintegration/imaging"
	i "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"golang.org/x/text/language"
)

const (
	permDir  = 0755 // permission used to create directories in cfg.OutputDirectory
	permFile = 0644 // permissions for
)

type config struct {
	BasePath              string   `env:"INPUT_BASE_PATH" envDefault:"/"`
	SourceDirectory       string   `env:"INPUT_SOURCE_DIRECTORY" envDefault:"."`
	OutputDirectory       string   `env:"INPUT_OUTPUT_DIRECTORY" envDefault:"./output"`
	AllowedFileExtensions []string `env:"INPUT_ALLOWED_FILE_EXTENSIONS" envDefault:".jpeg,.jpg,.png,.mp4,.pdf" envSeparator:","`
	TemplatesDirectory    string   `env:"INPUT_TEMPLATES_DIRECTORY" envDefault:"_templates"`
	DefaultTemplate       string   `env:"INPUT_DEFAULT_TEMPLATE" envDefault:"_post.html"`
	DefaultLanguage       string   `env:"INPUT_DEFAULT_LANGUAGE" envDefault:"en"`
	CommentsEnabled       bool     `env:"INPUT_COMMENTS_ENABLED" envDefault:"true"`
	CommentsSiteID        string   `env:"INPUT_COMMENTS_SITE_ID" envDefault:""`
	ShowDrafts            bool     `env:"INPUT_SHOW_DRAFTS"`
	ThumbPath             string   `env:"INPUT_THUMB_PATH" envDefault:"thumb"`
	ThumbMaxWidth         int      `env:"INPUT_THUMB_MAX_WIDTH" envDefault:"140"`
	ThumbMaxHeight        int      `env:"INPUT_THUMB_MAX_HEIGHT" envDefault:"140"`
	SearchEnabled         bool     `env:"INPUT_SEARCH_ENABLED"`
	SearchURL             string   `env:"INPUT_SEARCH_URL"`
	SearchPath            string   `env:"INPUT_SEARCH_PATH" envDefault:"search_index"`
}

// GetString returns the value of the environment variable named by the key.
// If the variable is not present, GetString returns empty string.
// Used in `config` template function to access config values.
func (c config) GetString(key string) string {
	// use reflect to get the value of the key
	v := reflect.ValueOf(c)
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == key {
			return v.Field(i).String()
		}
	}
	return ""
}

// cfg is a global variable, used in MarkdownFile struct and in template
var cfg config

// Data
type Data struct {
	Current            *MarkdownFile
	All                []*MarkdownFile
	LanguageVariations []*MarkdownFile // used only for index.html
	Timestamp          int64
}

var bundle *i.Bundle

func main() {
	log.Println("Starting")
	t := time.Now()

	if err := run(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	log.Printf("Finished in %dms", time.Now().Sub(t).Milliseconds())
}

var ts int64

func run() error {
	ts = time.Now().Unix()

	err := env.Parse(&cfg)
	if err != nil {
		return errors.Wrap(err, "environment variables parsing")
	}
	if cfg.DefaultLanguage == "" {
		cfg.DefaultLanguage = "en"
	}

	lang, err := language.Parse(cfg.DefaultLanguage)
	if err != nil {
		return errors.Wrapf(err, "parse language %q", cfg.DefaultLanguage)
	}
	bundle = i.NewBundle(lang) // used in templates/i18n to get translated strings
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	if err = createDirectory(cfg.OutputDirectory); err != nil {
		return errors.Wrapf(err, "output directory creation %q", cfg.OutputDirectory)
	}

	t, err := template.New("").Funcs(fm).ParseGlob(cfg.TemplatesDirectory + "/*")
	if err != nil {
		return errors.Wrap(err, "templates parsing")
	}

	defaultTemplate := t.Lookup(cfg.DefaultTemplate)
	if defaultTemplate == nil {
		return errors.Errorf("template %q not found", cfg.DefaultTemplate)
	}

	// scan source directory
	var markdownFiles []*MarkdownFile
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
					md, err := ParseMarkdownFile(path)
					if err != nil {
						log.Printf("ERROR: processing markdown file %s: %v", path, err)
						continue
					}

					if md.Draft && !cfg.ShowDrafts {
						log.Printf("DEBUG skipping draft %v", path)
						continue
					}

					for _, image := range md.Images {
						channelImages <- image
					}

					if md.Language == cfg.DefaultLanguage {
						// count tags only for default language,
						// assuming that post in different languages have the same tags
						// and that all posts have a version in default language
						tagsCounter.Add(md.Tags)
					}

					markdownFiles = append(markdownFiles, md)

				case ".toml":
					// Optional .toml files are used to define translations.
					// They power `i18n` template function.
					_, err := bundle.LoadMessageFile(cfg.SourceDirectory + "/" + path)
					if err != nil {
						log.Printf("ERROR load message file %q: %v", cfg.SourceDirectory+"/"+path, err)
					}

				default:
					// any other files will be copied to output directory
					copyFile(
						cfg.SourceDirectory+"/"+path,
						cfg.OutputDirectory+"/"+path,
					)
				}
			} else {
				doneFiles <- true
				return
			}
		}
	}()

	go func() {
		processedImages := make(map[string]bool)

		for {
			img, more := <-channelImages
			if more {
				if _, ok := processedImages[img.Path]; !ok {
					if err := resizeImage(
						cfg.SourceDirectory,
						img.Path,
						cfg.OutputDirectory+"/"+img.ThumbPath,
						cfg.ThumbMaxWidth,
						cfg.ThumbMaxHeight,
					); err != nil {
						log.Printf("ERROR resize image %q: %v", img.Path, err)
					}
					processedImages[img.Path] = true
				}
			} else {
				doneImages <- true
				return
			}
		}
	}()

	if err := readSourceDirectory(
		cfg.SourceDirectory,
		cfg.AllowedFileExtensions,
		channelFiles,
	); err != nil {
		return errors.Wrap(err, "read posts directory")
	}

	close(channelFiles)

	<-doneFiles
	close(channelImages)

	sort.Sort(ByCreated(markdownFiles))

	log.Println("Rendering markdown files...")
	if err = renderMarkdownFiles(markdownFiles, defaultTemplate); err != nil {
		return errors.Wrap(err, "rendering pages")
	}

	printTagsStags(tagsCounter)

	log.Println("Rendering templates...")
	if err := renderTemplates(t, markdownFiles); err != nil {
		return errors.Wrap(err, "rendering templates")
	}

	if cfg.SearchEnabled {
		if err := createSearchIndex(markdownFiles, cfg.SearchPath); err != nil {
			return errors.Wrap(err, "search index creation")
		}
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

func renderMarkdownFiles(files []*MarkdownFile, defaultTmpl *template.Template) error {
	for _, file := range files {
		tmpl := defaultTmpl
		if file.Template != "" {
			tmpl = defaultTmpl.Lookup(file.Template)
			if tmpl == nil {
				return errors.Errorf("template %q not found", file.Template)
			}
		}

		if err := renderTemplate(
			cfg.OutputDirectory+"/"+file.Path,
			Data{
				Current:   file,
				All:       files,
				Timestamp: ts,
			},
			tmpl,
		); err != nil {
			return errors.Wrapf(err, "rendering page %q", file.Path)
		}
	}
	return nil
}

func renderTemplates(t *template.Template, files []*MarkdownFile) error {
	mapID := make(map[string][]*MarkdownFile)

	// group pages by ID
	for _, tpl := range t.Templates() {
		filename := tpl.Name()
		// if template starts with underscore
		// or if template name is empty (root)
		// or it is a "post" template – skip it
		if strings.HasPrefix(filename, "_") ||
			filename == "" ||
			filename == cfg.DefaultTemplate {
			continue
		}

		id, lang := getIDAndLangFromFilename(filename)
		if lang == "" {
			lang = cfg.DefaultLanguage
		}

		mapID[id] = append(mapID[id], &MarkdownFile{
			ID:       id,
			Path:     filename,
			Language: lang,
		})
	}

	for _, pages := range mapID {
		sort.Sort(ByLanguage(pages))

		for _, p := range pages {

			tmpl := t.Lookup(p.Path)
			if tmpl == nil {
				log.Printf("WARNING: template %q not found", p.Path)
				continue
			}

			err := renderTemplate(
				cfg.OutputDirectory+"/"+p.Path,
				Data{
					Current:            p,
					All:                files,
					LanguageVariations: pages,
					Timestamp:          ts,
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

// createDirectory creates directory if it doesn't exsist
func createDirectory(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, permDir); err != nil {
				return errors.Wrapf(err, "create directory %q", dir)
			}
			return nil
		}

		return errors.Wrapf(err, "stat directory %q", dir)
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

		if (ext == ".md" && path != "README.md") ||
			ext == ".toml" ||
			inArray(allowedExtensions, ext) {

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

func createSearchIndex(pagesData []*MarkdownFile, searchIndexPath string) error {
	// check if search index exists, if not create it
	if _, err := os.Stat(searchIndexPath); !os.IsNotExist(err) {
		log.Println("Search index alrady exists, skipping creation")
		return nil
	}

	log.Println("Building search index...")

	indexer, err := search.NewIndexer(searchIndexPath, searchIndexPath+"_temp")
	if err != nil {
		return errors.Wrap(err, "failed to create search index")
	}

	err = indexer.RegisterType(MarkdownFile{Language: "en"}, "en")
	if err != nil {
		return errors.Wrap(err, "failed to register pageData type")
	}

	err = indexer.RegisterType(MarkdownFile{Language: "ru"}, "ru")
	if err != nil {
		return errors.Wrap(err, "failed to register pageData type")
	}

	for _, pageData := range pagesData {
		err = indexer.Index(pageData.Source, pageData)
		if err != nil {
			return errors.Wrapf(err, "failed to index %s", pageData.Source)
		}
	}

	return indexer.Close()
}
