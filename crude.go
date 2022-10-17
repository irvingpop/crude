package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	_ "github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const DefaultRulesContent = `# For docs on Refinery Rules, see: https://docs.honeycomb.io/manage-data-volume/refinery/sampling-methods/
Sampler = "DeterministicSampler"
SampleRate = 1
`

var tracer = otel.Tracer("crude")

func main() {
	// use honeycomb distro to setup OpenTelemetry SDK
	otelShutdown, err := launcher.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	router := gin.Default()
	router.Use(otelgin.Middleware("crude"))
	router.Static("/assets", "./assets")
	router.LoadHTMLGlob("templates/*")
	router.Use(sessions.Sessions("crudesession", cookie.NewStore([]byte("BoxContainsLiveB33s!"))))

	router.GET("/", homePage)
	router.POST("/", rulesReceiver)

	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.HTML(http.StatusInternalServerError, "error.html.tmpl", gin.H{
				"Code":    http.StatusInternalServerError,
				"Message": err,
			})
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	// if environment variable ENV=development, then run in debug mode
	if os.Getenv("ENV") == "development" {
		router.Run("localhost:5000")
	} else {
		router.Run(":8000")
	}
}

func homePage(c *gin.Context) {
	session := sessions.Default(c)

	rulesContent, err := getRulesFromS3(c)
	if err != nil {
		rulesContent, err = getRulesFromTmpFile(c)
		if err != nil {
			rulesContent = DefaultRulesContent
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

	err2 := writeRulesToS3(c, rulesContent)
	if err2 != nil {
		span.SetAttributes(attribute.String("error", err2.Error()))
		session.AddFlash(fmt.Sprintf("Something went wrong: %s", err2), "Error")
		session.Save()
		c.Redirect(http.StatusFound, "/")
	}

	session.AddFlash("Rules saved successfully", "Success")
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
