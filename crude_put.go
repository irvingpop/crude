package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func rulesReceiver(c *gin.Context) {
	rulesContent := c.PostForm("rules_content")

	_, span := tracer.Start(c.Request.Context(), "rulesReceiver", oteltrace.WithAttributes(attribute.String("rulesContent", rulesContent)))
	defer span.End()

	session := sessions.Default(c)

	err := writeRulesToTmpFile(c, rulesContent)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		session.AddFlash(fmt.Sprintf("Something went wrong: %s", err), "Error")
		session.Save()
		c.Redirect(http.StatusFound, "/")
	}

	err = validateRules(c, rulesContent)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		session.AddFlash(fmt.Sprintf("Unable to validate the rules file: %s", err), "Error")
		session.Save()
		c.Redirect(http.StatusFound, "/")
	}

	err = writeRulesToS3(c, rulesContent)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		session.AddFlash(fmt.Sprintf("Something went wrong: %s", err), "Error")
		session.Save()
		c.Redirect(http.StatusFound, "/")
	}

	session.AddFlash("Rules validated and saved successfully", "Success")
	session.Save()
	c.Redirect(http.StatusFound, "/")
}

func writeRulesToTmpFile(c *gin.Context, rulesContent string) error {
	_, span := tracer.Start(c.Request.Context(), "writeRulesToTmpFile", oteltrace.WithAttributes())
	defer span.End()

	tmpFile := "/tmp/rules.txt"
	err := os.WriteFile(tmpFile, []byte(rulesContent), 0644)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}
	return nil
}

// write rules file to S3
func writeRulesToS3(c *gin.Context, rulesContent string) error {
	_, span := tracer.Start(c.Request.Context(), "writeRulesToS3", oteltrace.WithAttributes())
	defer span.End()

	cfg, err := awsconfig.LoadDefaultConfig(c)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	client := s3.NewFromConfig(cfg)
	_, err = client.PutObject(c, &s3.PutObjectInput{
		Bucket: aws.String("irving-fafo"),
		Key:    aws.String("rules.txt"),
		Body:   strings.NewReader(rulesContent),
	})

	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	return nil
}
