package mdpdf

func mermaidConfigWithImagePrefix(cfg *MermaidRendererConfig, prefix string) *MermaidRendererConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.ImagePrefix = prefix + cloned.ImagePrefix
	return &cloned
}
