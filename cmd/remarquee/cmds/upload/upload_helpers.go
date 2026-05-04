package upload

import (
	"fmt"

	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func uploadPDFToRemote(cmd *cobra.Command, apiCtx api.ApiCtx, dstNodeCache map[string]*model.Node, dst string, outPDF string, label string) error {
	dstNode, ok := dstNodeCache[dst]
	if !ok {
		node, err := rmcloud.MkdirAll(apiCtx, dst)
		if err != nil {
			return err
		}
		dstNode = node
		dstNodeCache[dst] = node
	}

	document, err := apiCtx.UploadDocument(dstNode.Id(), outPDF, true, nil, nil, nil, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to upload file [%s]", outPDF)
	}
	apiCtx.Filetree().AddDocument(document)
	fmt.Fprintf(cmd.OutOrStdout(), "OK: uploaded %s -> %s\n", label, dst)
	return nil
}
