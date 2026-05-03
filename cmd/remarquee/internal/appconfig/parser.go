package appconfig

import (
	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedconfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/spf13/cobra"
)

const AppName = "remarquee"

func DefaultParserConfig() glazedcli.CobraParserConfig {
	return glazedcli.CobraParserConfig{
		AppName:           AppName,
		ShortHelpSections: []string{schema.DefaultSlug},
		ConfigPlanBuilder: BuildConfigPlan,
	}
}

func BuildConfigPlan(parsed *values.Values, _ *cobra.Command, _ []string) (*glazedconfig.Plan, error) {
	cs := &glazedcli.CommandSettings{}
	_ = parsed.DecodeSectionInto(glazedcli.CommandSettingsSlug, cs)

	return glazedconfig.NewPlan(
		glazedconfig.WithLayerOrder(
			glazedconfig.LayerSystem,
			glazedconfig.LayerUser,
			glazedconfig.LayerExplicit,
		),
	).Add(
		glazedconfig.SystemAppConfig(AppName).Named("system-app-config"),
		glazedconfig.XDGAppConfig(AppName).Named("xdg-app-config"),
		glazedconfig.HomeAppConfig(AppName).Named("home-app-config"),
		glazedconfig.ExplicitFile(cs.ConfigFile).Named("explicit-config"),
	), nil
}
