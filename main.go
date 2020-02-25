package main

import (
	"bufio"
	"github.com/gin-gonic/gin"
	"github.com/hanglics/gocheck/pkg/checker"
	"github.com/hanglics/gocheck/pkg/loader"
	"github.com/hscells/cqr"
	"github.com/hscells/groove/analysis"
	"github.com/hscells/transmute"
	tpipeline "github.com/hscells/transmute/pipeline"
	"github.com/ielab/searchrefiner"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

type AutoDocPlugin struct {
}

func handleQueryValidation(s searchrefiner.Server, c *gin.Context) {
	rawQuery := c.PostForm("query")
	lang := c.PostForm("lang")
	absPathFields, _ := filepath.Abs("../searchrefiner/dictionary/fields.txt")
	fieldsDictionary := loader.LoadDictionary(absPathFields)
	absPath, _ := filepath.Abs("../searchrefiner/dictionary/words.txt")
	keywordDictionary := loader.LoadDictionary(absPath)

	lang = strings.ToLower(lang)

	var fieldsError []string

	if strings.ToLower(lang) == "medline" {
		scanner := bufio.NewScanner(strings.NewReader(rawQuery))
		var extractedFields []string
		for scanner.Scan() {
			temp := scanner.Text()
			line := temp[3:]
			reg := regexp.MustCompile(`\.([^.]+)\.`)
			rawFields := reg.FindAllStringSubmatch(line, -1)
			if len(rawFields) > 0 {
				extractedFields = append(extractedFields, rawFields[0][1])
			}
		}
		for _, i := range extractedFields {
			flag := checker.CheckWord(fieldsDictionary, strings.ToLower(i), 0)
			if !flag {
				fieldsError = append(fieldsError, i)
			}
		}
	} else if strings.ToLower(lang) == "pubmed" {
		reg := regexp.MustCompile(`\[([^]]+)\]`)
		rawFields := reg.FindAllStringSubmatch(rawQuery, -1)
		for _, i := range rawFields {
			flag := checker.CheckWord(fieldsDictionary, strings.ToLower(i[1]), 0)
			if !flag {
				fieldsError = append(fieldsError, i[1])
			}
		}
	}
	p := make(map[string]tpipeline.TransmutePipeline)
	p["medline"] = transmute.Medline2Cqr
	p["pubmed"] = transmute.Pubmed2Cqr
	compiler := p["medline"]
	if v, ok := p[lang]; ok {
		compiler = v
	} else {
		lang = "medline"
	}
	cq, err := compiler.Execute(rawQuery)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	repr, err := cq.Representation()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
	commonRepr := repr.(cqr.CommonQueryRepresentation)
	keywords := analysis.QueryKeywords(commonRepr)
	var spellErrors []string
	resp := make(map[string][]string)
	for i := 0; i < len(keywords); i++ {
		keyword := keywords[i].QueryString
		keyword = strings.ToLower(keyword)
		var slices []string
		if strings.Contains(keyword, " ") {
			slices = strings.Split(keyword, " ")
			for _, s := range slices {
				if strings.Contains(s, "*") {
					s = s[:len(s)-1]
				}
				flag := checker.CheckWord(keywordDictionary, s, 0)
				if !flag {
					spellErrors = append(spellErrors, s)
				}
			}
		} else {
			if strings.Contains(keyword, "*") {
				keyword = keyword[:len(keyword)-1]
			}
			flag := checker.CheckWord(keywordDictionary, keyword, 0)
			if !flag {
				spellErrors = append(spellErrors, keyword)
			}
		}
	}
	resp["keyword"] = spellErrors
	resp["fields"] = fieldsError
	c.JSON(http.StatusOK, resp)
}

func (AutoDocPlugin) Serve(s searchrefiner.Server, c *gin.Context) {
	if c.Request.Method == "POST" && c.Query("validate") == "y" {
		handleQueryValidation(s, c)
		return
	}
	rawQuery := c.PostForm("query")
	lang := c.PostForm("lang")
	c.Render(http.StatusOK, searchrefiner.RenderPlugin(searchrefiner.TemplatePlugin("plugin/autodoc/index.html"), struct {
		searchrefiner.Query
		View string
	}{searchrefiner.Query{QueryString: rawQuery, Language: lang}, c.Query("view")}))
	return
}

func (AutoDocPlugin) PermissionType() searchrefiner.PluginPermission {
	return searchrefiner.PluginUser
}

func (AutoDocPlugin) Details() searchrefiner.PluginDetails {
	return searchrefiner.PluginDetails{
		Title:       "AutoDoc",
		Description: "Automatically generate search reports based on the information provided.",
		Author:      "ielab",
		Version:     "23.Jan.2020",
		ProjectURL:  "ielab.io/searchrefiner",
	}
}

var Autodoc = AutoDocPlugin{}
