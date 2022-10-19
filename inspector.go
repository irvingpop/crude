package main

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func rulesInspector(c *gin.Context) {
	session := sessions.Default(c)

	rulesContent, err := getRulesFromS3(c)
	if err != nil {
		rulesContent, err = getRulesFromTmpFile(c)
		session.AddFlash("Rules not found in S3 or unable to fetch, using local file", "Error")
		session.Save()
		if err != nil {
			rulesContent = DefaultRulesContent
			session.AddFlash("Rules not found in S3 or locally, here's a Default config", "Error")
			session.Save()
		}
	}

	// parsedRules, err := validateRules(c, rulesContent)

	parsedRules, err := rulesToJSON(c, rulesContent)
	if err != nil {
		parsedRules = fmt.Sprintf("%s: %s", parsedRules, err.Error())
	}


	otelgin.HTML(c, http.StatusOK, "inspector.html.tmpl", gin.H{
		"rules_content": rulesContent,
		"parsed_rules":  parsedRules,
		"title":        "Analyze the deployed Rules File",
	})
}
