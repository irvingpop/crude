package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	_ "github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
)

const DefaultRulesContent = `# For docs on Refinery Rules, see: https://docs.honeycomb.io/manage-data-volume/refinery/sampling-methods/
Sampler = "DeterministicSampler"
SampleRate = 1
`

var tracer = otel.Tracer("crude")

//go:embed templates/*
var embeddedTemplates embed.FS

//go:embed images/* js/*
var embeddedAssets embed.FS

func main() {
	// use honeycomb distro to setup OpenTelemetry SDK
	otelShutdown, err := launcher.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	router := gin.Default()

	// embed all the templates into the binary
	// thx https://github.com/gin-gonic/examples/blob/master/assets-in-binary/example02/main.go
	templ := template.Must(template.New("").ParseFS(embeddedTemplates, "templates/*.tmpl"))
	router.SetHTMLTemplate(templ)
	router.StaticFS("/assets", http.FS(embeddedAssets))

	router.Use(otelgin.Middleware("crude"))
	router.Use(sessions.Sessions("crudesession", cookie.NewStore([]byte("BoxContainsLiveB33s!"))))

	router.GET("/", homePage)
	router.GET("/inspector", rulesInspector)
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
