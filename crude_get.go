package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func homePage(c *gin.Context) {
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

	// clear the flashes
	// https://github.com/gin-gonic/contrib/issues/54
	success_flashes := session.Flashes("Success")
	error_flashes := session.Flashes("Error")
	session.Save()

	otelgin.HTML(c, http.StatusOK, "index.html.tmpl", gin.H{
		"rules_content": rulesContent,
		"MsgSuccess":    success_flashes,
		"MsgError":      error_flashes,
		"title":         "Deploy a Rules File",
	})
}

func getRulesFromTmpFile(c *gin.Context) (string, error) {
	_, span := tracer.Start(c.Request.Context(), "getRulesFromTmpFile", oteltrace.WithAttributes())
	defer span.End()

	tmpFile := "/tmp/rules.txt"
	rulesContent, err := os.ReadFile(tmpFile)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return "", err
	}
	span.SetAttributes(attribute.String("rulesContent", string(rulesContent)))
	return string(rulesContent), nil
}

func getRulesFromS3(c *gin.Context) (string, error) {
	_, span := tracer.Start(c.Request.Context(), "getRulesFromS3", oteltrace.WithAttributes())
	defer span.End()

	cfg, err := awsconfig.LoadDefaultConfig(c)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetAttributes(attribute.String("error_stage", "LoadDefaultConfig"))
		return "", err
	}

	client := s3.NewFromConfig(cfg)
	result, err := client.GetObject(c, &s3.GetObjectInput{
		Bucket: aws.String("irving-fafo"),
		Key:    aws.String("rules.txt"),
	})

	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetAttributes(attribute.String("error_stage", "GetObject"))
		return "", err
	}

	rulesContent, err := ioutil.ReadAll(result.Body)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetAttributes(attribute.String("error_stage", "ioutil.ReadAll"))
		return "", err
	}

	span.SetAttributes(attribute.String("rulesContent", string(rulesContent)))
	return string(rulesContent), nil
}
