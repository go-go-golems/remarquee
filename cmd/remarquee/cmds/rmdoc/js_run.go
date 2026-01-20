package rmdoc

import (
	"context"
	"fmt"
	"os"
	
	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/remarquee/pkg/rmdoc/js"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type JSRunCommand struct {
	*glazecmds.CommandDescription
}

var _ glazecmds.BareCommand = &JSRunCommand{}

func NewJSRunCommand() (*JSRunCommand, error) {
	cmdDesc := glazecmds.NewCommandDescription(
		"js-run",
		glazecmds.WithShort("Run a JavaScript script to create an rmdoc document"),
		glazecmds.WithLong(`
Run a JavaScript script using the Remarquee.js API to create an rmdoc document.

The script has access to the following API:
  - RMDoc: Create a new document
  - Canvas: Draw on pages with canvas-like API
  - Stroke: Low-level stroke manipulation
  - Colors: Color constants and utilities

Examples:
  remarquee rmdoc js-run script.js
  remarquee rmdoc js-run examples/simple-line.js
`),
		glazecmds.WithFlags(
			parameters.NewParameterDefinition(
				"script",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Path to the JavaScript file to execute"),
			),
		),
	)
	
	return &JSRunCommand{CommandDescription: cmdDesc}, nil
}

type JSRunSettings struct {
	Script string `glazed.parameter:"script"`
}

func (c *JSRunCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &JSRunSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}
	
	// Read the JavaScript file
	scriptContent, err := os.ReadFile(s.Script)
	if err != nil {
		return errors.Wrap(err, "failed to read script file")
	}
	
	// Create a new goja runtime with the Remarquee.js API
	vm := js.NewRuntime()
	
	// Execute the script
	_, err = js.RunScript(vm, string(scriptContent))
	if err != nil {
		return errors.Wrap(err, "failed to execute script")
	}
	
	fmt.Println("Script executed successfully")
	
	return nil
}

func NewJSRunCobraCommand() (*cobra.Command, error) {
	cmd, err := NewJSRunCommand()
	if err != nil {
		return nil, err
	}
	
	cobraCmd, err := cli.BuildCobraCommand(cmd,
		cli.WithCobraShortHelpLayers(layers.DefaultSlug),
	)
	if err != nil {
		return nil, err
	}
	
	return cobraCmd, nil
}
