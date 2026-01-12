package ocr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	cmdmiddlewares "github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	glazedsettings "github.com/go-go-golems/glazed/pkg/settings"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type OcrCommand struct {
	*glazecmds.CommandDescription
}

type OcrSettings struct {
	ImagePath string `glazed.parameter:"image"`

	SystemPrompt string `glazed.parameter:"system"`
	Prompt       string `glazed.parameter:"prompt"`

	MediaType string `glazed.parameter:"media-type"`
}

var _ glazecmds.WriterCommand = (*OcrCommand)(nil)

const defaultSystemPrompt = "You are an OCR engine. Extract all legible text from the provided image. Output plain text only (no markdown), preserve line breaks and spacing as best as possible. Do not add commentary."

const defaultUserPrompt = "OCR this image."

func NewOcrCommand() (*OcrCommand, error) {
	glazedLayer, err := glazedsettings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	// Override AI defaults for this command so users get a vision-capable model out of the box.
	ss, err := aisettings.NewStepSettings()
	if err != nil {
		return nil, err
	}
	model := "gpt-4o-mini"
	ss.Chat.Engine = &model
	maxTokens := 4096
	ss.Chat.MaxResponseTokens = &maxTokens
	temp := 0.0
	ss.Chat.Temperature = &temp

	geppettoLayers, err := geppettolayers.CreateGeppettoLayers(
		geppettolayers.WithDefaultsFromStepSettings(ss),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layers")
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"ocr",
		glazecmds.WithShort("OCR an image using an LLM vision model (Geppetto)"),
		glazecmds.WithLong(`
Runs a single multimodal inference request to extract text from an image.

Examples:
  remarquee ocr ./scan.png --openai-api-key "$OPENAI_API_KEY"
  remarquee ocr ./photo.jpg --ai-engine gpt-4o --openai-api-key "$OPENAI_API_KEY"

Notes:
  - This command sends the full image to the configured provider/model.
  - Use a vision-capable model (default: gpt-4o-mini).
`),
		glazecmds.WithArguments(
			parameters.NewParameterDefinition(
				"image",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Path to the image file to OCR"),
			),
		),
		glazecmds.WithFlags(
			parameters.NewParameterDefinition(
				"system",
				parameters.ParameterTypeString,
				parameters.WithDefault(defaultSystemPrompt),
				parameters.WithHelp("System prompt (OCR instructions)"),
			),
			parameters.NewParameterDefinition(
				"prompt",
				parameters.ParameterTypeString,
				parameters.WithDefault(defaultUserPrompt),
				parameters.WithHelp("User prompt to accompany the image"),
			),
			parameters.NewParameterDefinition(
				"media-type",
				parameters.ParameterTypeString,
				parameters.WithDefault(""),
				parameters.WithHelp("Override detected media type (e.g. image/png, image/jpeg)"),
			),
		),
		glazecmds.WithLayersList(append([]layers.ParameterLayer{glazedLayer, commandSettingsLayer}, geppettoLayers...)...),
	)

	return &OcrCommand{CommandDescription: cmdDesc}, nil
}

func (c *OcrCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
	s := &OcrSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	img, err := os.ReadFile(s.ImagePath)
	if err != nil {
		return errors.Wrapf(err, "failed to read image file %q", s.ImagePath)
	}
	if len(img) == 0 {
		return errors.Errorf("image file %q is empty", s.ImagePath)
	}

	mediaType := strings.TrimSpace(s.MediaType)
	if mediaType == "" {
		sniffLen := minInt(512, len(img))
		mediaType = http.DetectContentType(img[:sniffLen])
		// Fallback for generic octet-stream: infer from extension.
		if mediaType == "application/octet-stream" {
			switch strings.ToLower(filepath.Ext(s.ImagePath)) {
			case ".png":
				mediaType = "image/png"
			case ".jpg", ".jpeg":
				mediaType = "image/jpeg"
			case ".webp":
				mediaType = "image/webp"
			case ".gif":
				mediaType = "image/gif"
			}
		}
	}
	if !strings.HasPrefix(mediaType, "image/") {
		return errors.Errorf("detected media type %q does not look like image/* (file=%q); use --media-type to override", mediaType, s.ImagePath)
	}

	images := []map[string]any{
		{
			"media_type": mediaType,
			"content":    img,
		},
	}

	t := &turns.Turn{}
	if strings.TrimSpace(s.SystemPrompt) != "" {
		turns.AppendBlock(t, turns.NewSystemTextBlock(s.SystemPrompt))
	}
	turns.AppendBlock(t, turns.NewUserMultimodalBlock(s.Prompt, images))

	engine, err := factory.NewEngineFromParsedLayers(parsedLayers)
	if err != nil {
		return errors.Wrap(err, "failed to create geppetto engine (check --ai-api-type/--ai-engine and provider api-key flags)")
	}

	updated, err := engine.RunInference(ctx, t)
	if err != nil {
		return errors.Wrap(err, "ocr inference failed")
	}
	if updated == nil {
		return errors.New("ocr inference returned nil turn")
	}

	// Extract the last assistant text block.
	var out string
	for i := len(updated.Blocks) - 1; i >= 0; i-- {
		b := updated.Blocks[i]
		if b.Kind != turns.BlockKindLLMText && b.Kind != turns.BlockKindOther {
			continue
		}
		if txt, ok := b.Payload[turns.PayloadKeyText].(string); ok && strings.TrimSpace(txt) != "" {
			out = txt
			break
		}
	}
	if out == "" {
		return errors.New("model returned no assistant text blocks")
	}

	_, _ = fmt.Fprintln(w, out)
	return nil
}

func NewOcrCobraCommand() (*cobra.Command, error) {
	cmd, err := NewOcrCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithProfileSettingsLayer(),
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: func(parsedCommandLayers *layers.ParsedLayers, cmd_ *cobra.Command, args []string) ([]cmdmiddlewares.Middleware, error) {
				// Support pinocchio-style profiles without pulling in deprecated viper middleware.
				profileSettings := &cli.ProfileSettings{}
				if err := parsedCommandLayers.InitializeStruct(cli.ProfileSettingsSlug, profileSettings); err != nil {
					return nil, err
				}

				xdgConfigPath, err := os.UserConfigDir()
				if err != nil {
					return nil, err
				}
				defaultProfileFile := fmt.Sprintf("%s/pinocchio/profiles.yaml", xdgConfigPath)

				profileFile := profileSettings.ProfileFile
				if profileFile == "" {
					profileFile = defaultProfileFile
				}
				profile := profileSettings.Profile
				if profile == "" {
					profile = "default"
				}

				return []cmdmiddlewares.Middleware{
					// Highest precedence
					cmdmiddlewares.ParseFromCobraCommand(cmd_, parameters.WithParseStepSource("cobra")),
					cmdmiddlewares.GatherArguments(args, parameters.WithParseStepSource("arguments")),
					cmdmiddlewares.UpdateFromEnv("REMARQUEE", parameters.WithParseStepSource("env")),
					cmdmiddlewares.GatherFlagsFromProfiles(
						defaultProfileFile,
						profileFile,
						profile,
						parameters.WithParseStepSource("profiles"),
						parameters.WithParseStepMetadata(map[string]any{
							"profileFile": profileFile,
							"profile":     profile,
						}),
					),
					// Lowest precedence
					cmdmiddlewares.SetFromDefaults(parameters.WithParseStepSource(parameters.SourceDefaults)),
				}, nil
			},
		}),
	)
}

// NewOCRCommand returns a cobra.Command and never fails; init errors are surfaced at runtime.
func NewOCRCommand() *cobra.Command {
	cmd, err := NewOcrCobraCommand()
	if err != nil {
		return &cobra.Command{
			Use:   "ocr",
			Short: "OCR an image using an LLM vision model (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		}
	}
	return cmd
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
