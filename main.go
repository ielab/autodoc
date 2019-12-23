package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ielab/searchrefiner"
	"net/http"
)

type AutoDocPlugin struct {
}

func (AutoDocPlugin) Serve(s searchrefiner.Server, c *gin.Context) {
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
		Title:       "Auto Doc",
		Description: "Auto generate search reports based on the information provided.",
		Author:      "ielab",
		Version:     "17.Dec.2019",
		ProjectURL:  "ielab.io/searchrefiner",
	}
}

var Autodoc = AutoDocPlugin{}
