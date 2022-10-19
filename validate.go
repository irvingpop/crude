package main

import (
	"bytes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
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

func validateRules(c *gin.Context, rulesContent string) (string, error) {
	_, span := tracer.Start(c.Request.Context(), "validateRules", oteltrace.WithAttributes())
	defer span.End()

	configFile := "/tmp/config.toml"
	err := os.WriteFile(configFile, []byte(defaultConfig), 0644)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return "Something went terribly wrong, but not with your rules", err
	}

	rulesFile := "/tmp/rules.toml"
	err = os.WriteFile(rulesFile, []byte(rulesContent), 0644)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return "Unable to write the rules file to disk", err
	}

	config, err := refineryconfig.NewConfig("/tmp/config.toml", "/tmp/rules.toml", func(err error) {})
	if err != nil {
		log.Printf("Unable to load config: %+v\n", err)
		return "Failed to parse the rules", err
	}

	allrules, _ := config.GetAllSamplerRules()
	// for index, rule := range allrules {
	// 	ruletoml, _ := toml.Marshal(rule)
	// 	log.Printf("rule:\n%s: (%s)\n %s\n", index, reflect.TypeOf(rule), ruletoml)
	// }

	buf := bytes.Buffer{}
	enc := toml.NewEncoder(&buf)
	enc.SetIndentTables(true)
	enc.Encode(allrules)

	// TOMLv2 only returns single quotes but all our docs use double quotes, so standardize
	rulesReplacedQuotes := string(bytes.ReplaceAll(buf.Bytes(), []byte(`'`), []byte(`"`)))
	log.Printf("ALL RULES: --------- \n%s\n", string(buf.Bytes()))

	return rulesReplacedQuotes, err
}

// represent a rules TOML file as a tree view
func rulesToJSON(c *gin.Context, rulesContent string) (string, error) {
	_, span := tracer.Start(c.Request.Context(), "rulesToJSON", oteltrace.WithAttributes())
	defer span.End()

	configFile := "/tmp/config.toml"
	err := os.WriteFile(configFile, []byte(defaultConfig), 0644)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return "Something went terribly wrong, but not with your rules", err
	}

	rulesFile := "/tmp/rules.toml"
	err = os.WriteFile(rulesFile, []byte(rulesContent), 0644)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return "Unable to write the rules file to disk", err
	}

	config, err := refineryconfig.NewConfig("/tmp/config.toml", "/tmp/rules.toml", func(err error) {})
	if err != nil {
		log.Printf("Unable to load config: %+v\n", err)
		return "Failed to parse the rules", err
	}

	allrules, _ := config.GetAllSamplerRules()

	rulesJSON, err := json.MarshalIndent(allrules, "", "  ")

	return string(rulesJSON), err
}

