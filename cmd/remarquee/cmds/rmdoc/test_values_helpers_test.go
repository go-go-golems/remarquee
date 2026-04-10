package rmdoc

import (
	"testing"

	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func newDefaultParsedValues(t *testing.T, desc *glazecmds.CommandDescription, data map[string]interface{}) *values.Values {
	t.Helper()

	defaultSection, ok := desc.GetDefaultSection()
	if !ok {
		t.Fatal("default section not found")
	}

	options := make([]values.SectionValuesOption, 0, len(data))
	for k, v := range data {
		options = append(options, values.WithFieldValue(k, v))
	}

	sectionValues, err := values.NewSectionValues(defaultSection, options...)
	if err != nil {
		t.Fatalf("NewSectionValues: %v", err)
	}

	return values.New(values.WithSectionValues(schema.DefaultSlug, sectionValues))
}
