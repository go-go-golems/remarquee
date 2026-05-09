package upload

import (
	"fmt"

	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func uploadPDFToRemoteWithAuthRetry(cmd *cobra.Command, authSettings rmcloud.AuthSettings, apiCtx api.ApiCtx, dst string, outPDF string, label string, force bool) (api.ApiCtx, error) {
	return rmcloud.WithAuthRetry(authSettings, apiCtx, func(currentCtx api.ApiCtx) (api.ApiCtx, error) {
		dstNode, err := rmcloud.MkdirAll(currentCtx, dst)
		if err != nil {
			return currentCtx, err
		}

		docName, _ := util.DocPathToName(outPDF)
		existingNode, err := currentCtx.Filetree().NodeByPath(docName, dstNode)
		if err == nil {
			if !force {
				fmt.Fprintf(cmd.OutOrStdout(), "SKIP: %s already exists in %s (use --force to overwrite)\n", docName, dst)
				return currentCtx, nil
			}

			if existingNode.IsDirectory() {
				return currentCtx, errors.Errorf("cannot overwrite directory %q in %s", docName, dst)
			}

			if err := currentCtx.DeleteEntry(existingNode, false, false); err != nil {
				return currentCtx, errors.Wrap(err, "failed to delete existing file")
			}
			currentCtx.Filetree().DeleteNode(existingNode)
		}

		document, err := currentCtx.UploadDocument(dstNode.Id(), outPDF, true, nil, nil, nil, nil)
		if err != nil {
			return currentCtx, errors.Wrapf(err, "failed to upload file [%s]", outPDF)
		}
		currentCtx.Filetree().AddDocument(document)
		fmt.Fprintf(cmd.OutOrStdout(), "OK: uploaded %s -> %s\n", label, dst)
		return currentCtx, nil
	})
}

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
