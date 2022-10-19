package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"

	refineryconfig "github.com/honeycombio/refinery/config"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const defaultConfig = `[InMemCollector]
CacheCapacity = 1000
[HoneycombMetrics]
MetricsHoneycombAPI = "https://api.honeycomb.io"
MetricsAPIKey = "abcd1234"
MetricsDataset = "Refinery Metrics"
MetricsReportingInterval = 3
`

func validateRules(c *gin.Context, rulesContent string) error {
	_, span := tracer.Start(c.Request.Context(), "validateRules", oteltrace.WithAttributes())
	defer span.End()

	configFile := "/tmp/config.toml"
	err := os.WriteFile(configFile, []byte(defaultConfig), 0644)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	rulesFile := "/tmp/rules.toml"
	err = os.WriteFile(rulesFile, []byte(rulesContent), 0644)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	config, err := refineryconfig.NewConfig("/tmp/config.toml", "/tmp/rules.toml", func(err error) {})
	if err != nil {
		log.Printf("Unable to load config: %+v\n", err)
	}

	allrules, _ := config.GetAllSamplerRules()
	// for index, rule := range allrules {
	// 	ruletoml, _ := toml.Marshal(rule)
	// 	log.Printf("rule:\n%s: (%s)\n %s\n", index, reflect.TypeOf(rule), ruletoml)
	// }

	tomlrules, _ := toml.Marshal(allrules)
	log.Printf("ALL RULES: --------- \n%s\n", tomlrules)

	return err
}
