package generator

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/RomanosTrechlis/blog-gen/config"
	"github.com/RomanosTrechlis/blog-gen/util/fs"
	"github.com/RomanosTrechlis/blog-gen/util/url"
	"github.com/russross/blackfriday"
	"github.com/sourcegraph/syntaxhighlight"
)

// post holds data for a post
type post struct {
	name      string
	html      []byte
	meta      *Meta
	imagesDir string
	images    []string
}

// byDateDesc is the sorting object for posts
type byDateDesc []*post

// postGenerator object
type postGenerator struct {
	post        *post
	siteInfo    *config.SiteInformation
	template    *template.Template
	destination string
}

// Generate generates a post
func (g *postGenerator) Generate() (err error) {
	post := g.post
	fmt.Printf("\tGenerating Post: %s...\n", post.meta.Title)
	staticPath := filepath.Join(g.destination, post.name)
	err = fs.CreateFolderIfNotExist(staticPath)
	if err != nil {
		return fmt.Errorf("error creating directory at %s: %v", staticPath, err)
	}
	if post.imagesDir != "" {
		err := g.copyImagesDir(post.imagesDir, staticPath)
		if err != nil {
			return err
		}
	}

	c := htmlConfig{
		path:       url.ChangePathToUrl(staticPath),
		pageTitle:  post.meta.Title,
		pageNum:    0,
		maxPageNum: 0,
		isPost:     true,
		temp:       g.template,
		content:    template.HTML(string(post.html)),
		siteInfo:   g.siteInfo,
	}
	err = c.writeHTML()
	if err != nil {
		return err
	}

	err = g.copyAdditionalArtifacts(staticPath, post.name)
	if err != nil {
		return err
	}
	fmt.Printf("\tFinished generating Post: %s...\n", post.meta.Title)
	return nil
}

func (g *postGenerator) copyAdditionalArtifacts(path, postName string) (err error) {
	src := filepath.Join(g.siteInfo.TempFolder, postName, "artifacts")
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return nil
	}
	for _, file := range files {
		src = filepath.Join(src, file.Name())
		err := fs.CopyFile(src, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (*postGenerator) copyImagesDir(source, destination string) (err error) {
	path := filepath.Join(destination, "images")
	err = fs.CreateFolderIfNotExist(path)
	if err != nil {
		return fmt.Errorf("error creating images directory at %s: %v", path, err)
	}
	files, err := ioutil.ReadDir(source)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %v", path, err)
	}
	for _, file := range files {
		src := filepath.Join(source, file.Name())
		err := fs.CopyFile(src, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func getHTML(path string) (html []byte, err error) {
	filePath := filepath.Join(path, "post.md")
	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error while reading file %s: %v", filePath, err)
	}
	html = blackfriday.MarkdownCommon(input)
	replaced, err := replaceCodeParts(html)
	if err != nil {
		return nil, fmt.Errorf("error during syntax highlighting of %s: %v", filePath, err)
	}
	html = []byte(replaced)
	return html, nil
}

func getImages(path string) (dirPath string, images []string, err error) {
	dirPath = filepath.Join(path, "images")
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("error while reading folder %s: %v", dirPath, err)
	}
	images = []string{}
	for _, file := range files {
		images = append(images, file.Name())
	}
	return dirPath, images, nil
}

func replaceCodeParts(htmlFile []byte) (new string, err error) {
	byteReader := bytes.NewReader(htmlFile)
	doc, err := goquery.NewDocumentFromReader(byteReader)
	if err != nil {
		return "", fmt.Errorf("error while parsing html: %v", err)
	}
	// find code-parts via css selector and replace them with highlighted versions
	doc.Find("code[class*=\"language-\"]").Each(func(i int, s *goquery.Selection) {
		oldCode := s.Text()
		formatted, _ := syntaxhighlight.AsHTML([]byte(oldCode))
		s.SetHtml(string(formatted))
	})
	new, err = doc.Html()
	if err != nil {
		return "", fmt.Errorf("error while generating html: %v", err)
	}
	// replace unnecessarily added html tags
	new = strings.Replace(new, "<html><head></head><body>", "", 1)
	new = strings.Replace(new, "</body></html>", "", 1)
	return new, nil
}

func (p byDateDesc) Len() int {
	return len(p)
}

func (p byDateDesc) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p byDateDesc) Less(i, j int) bool {
	return p[i].meta.ParsedDate.After(p[j].meta.ParsedDate)
}
