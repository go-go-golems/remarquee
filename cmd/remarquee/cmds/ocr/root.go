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
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedsettings "github.com/go-go-golems/glazed/pkg/settings"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type OcrCommand struct {
	*glazecmds.CommandDescription
}

type OcrSettings struct {
	ImagePath string `glazed:"image"`

	SystemPrompt string `glazed:"system"`
	Prompt       string `glazed:"prompt"`

	MediaType string `glazed:"media-type"`
}

var _ glazecmds.WriterCommand = (*OcrCommand)(nil)

const defaultSystemPrompt = "You are an OCR engine. Extract all legible text from the provided image. Output plain text only (no markdown), preserve line breaks and spacing as best as possible. Do not add commentary."

const defaultUserPrompt = "OCR this image."

func NewOcrCommand() (*OcrCommand, error) {
	glazedSection, err := glazedsettings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	// Override AI defaults for this command so users get a vision-capable model out of the box.
	ss, err := aisettings.NewInferenceSettings()
	if err != nil {
		return nil, err
	}
	model := "gpt-4o-mini"
	ss.Chat.Engine = &model
	maxTokens := 4096
	ss.Chat.MaxResponseTokens = &maxTokens
	temp := 0.0
	ss.Chat.Temperature = &temp

	geppettoSections, err := geppettosections.CreateGeppettoSections(
		geppettosections.WithDefaultsFromInferenceSettings(ss),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto sections")
	}

	sections := append([]schema.Section{glazedSection, commandSettingsSection}, geppettoSections...)

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
			fields.New(
				"image",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("Path to the image file to OCR"),
			),
		),
		glazecmds.WithFlags(
			fields.New(
				"system",
				fields.TypeString,
				fields.WithDefault(defaultSystemPrompt),
				fields.WithHelp("System prompt (OCR instructions)"),
			),
			fields.New(
				"prompt",
				fields.TypeString,
				fields.WithDefault(defaultUserPrompt),
				fields.WithHelp("User prompt to accompany the image"),
			),
			fields.New(
				"media-type",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Override detected media type (e.g. image/png, image/jpeg)"),
			),
		),
		glazecmds.WithSections(sections...),
	)

	return &OcrCommand{CommandDescription: cmdDesc}, nil
}

func (c *OcrCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	s := &OcrSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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

	engine, err := factory.NewEngineFromParsedValues(parsedValues)
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
		cli.WithCobraShortHelpSections(schema.DefaultSlug),
		cli.WithCobraMiddlewaresFunc(geppettosections.GetCobraCommandGeppettoMiddlewares),
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
