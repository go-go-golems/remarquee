package upload

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type syncAction string

const (
	syncActionUpload syncAction = "upload"
	syncActionSkip   syncAction = "skip"
	syncActionStale  syncAction = "stale"
	syncActionOrphan syncAction = "orphan"
)

type syncPlanSettings struct {
	RemoteDir     string
	PreserveDirs  bool
	CompareMTime  bool
	DeleteOrphans bool
	Name          string
}

type syncLocalDoc struct {
	Input     markdownInput
	PDFName   string
	DocName   string
	RemoteDir string
	RemoteKey string
	ModTime   time.Time
}

type syncRemoteEntry struct {
	Name         string
	Path         string
	IsDir        bool
	ModifiedTime time.Time
	Node         interface{}
}

type syncPlanItem struct {
	Action syncAction
	Local  *syncLocalDoc
	Remote *syncRemoteEntry
	Reason string
}

type syncPlan struct {
	Items []syncPlanItem
}

func (p syncPlan) Count(action syncAction) int {
	count := 0
	for _, item := range p.Items {
		if item.Action == action {
			count++
		}
	}
	return count
}

func buildSyncPlan(inputs []markdownInput, remoteIndex map[string]syncRemoteEntry, settings syncPlanSettings) (syncPlan, error) {
	localDocs, err := buildSyncLocalDocs(inputs, settings)
	if err != nil {
		return syncPlan{}, err
	}

	items := make([]syncPlanItem, 0, len(localDocs)+len(remoteIndex))
	localKeys := map[string]struct{}{}

	for i := range localDocs {
		localDoc := localDocs[i]
		localKeys[localDoc.RemoteKey] = struct{}{}

		remote, ok := remoteIndex[localDoc.RemoteKey]
		if !ok {
			items = append(items, syncPlanItem{
				Action: syncActionUpload,
				Local:  &localDoc,
				Reason: "remote document is missing",
			})
			continue
		}

		remoteCopy := remote
		if remote.IsDir {
			items = append(items, syncPlanItem{
				Action: syncActionStale,
				Local:  &localDoc,
				Remote: &remoteCopy,
				Reason: "remote path is a directory",
			})
			continue
		}

		if settings.CompareMTime && !remote.ModifiedTime.IsZero() && localDoc.ModTime.After(remote.ModifiedTime) {
			items = append(items, syncPlanItem{
				Action: syncActionStale,
				Local:  &localDoc,
				Remote: &remoteCopy,
				Reason: "local markdown is newer than remote document",
			})
			continue
		}

		items = append(items, syncPlanItem{
			Action: syncActionSkip,
			Local:  &localDoc,
			Remote: &remoteCopy,
			Reason: "remote document already exists",
		})
	}

	if settings.DeleteOrphans {
		keys := make([]string, 0, len(remoteIndex))
		for key := range remoteIndex {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			remote := remoteIndex[key]
			if remote.IsDir {
				continue
			}
			if _, ok := localKeys[key]; ok {
				continue
			}
			remoteCopy := remote
			items = append(items, syncPlanItem{
				Action: syncActionOrphan,
				Remote: &remoteCopy,
				Reason: "remote document has no matching local markdown input",
			})
		}
	}

	return syncPlan{Items: items}, nil
}

func buildSyncLocalDocs(inputs []markdownInput, settings syncPlanSettings) ([]syncLocalDoc, error) {
	seenRemoteKeys := map[string]string{}
	localDocs := make([]syncLocalDoc, 0, len(inputs))

	for _, in := range inputs {
		pdfName, err := markdownPDFName(in, settings.Name, len(inputs))
		if err != nil {
			return nil, err
		}
		docName := strings.TrimSuffix(pdfName, filepath.Ext(pdfName))

		relDir := ""
		if settings.PreserveDirs {
			relDir = in.RelDir()
		}
		dst := joinRemoteDir(settings.RemoteDir, relDir)
		remoteKey := remoteDocKey(settings.RemoteDir, relDir, docName)

		if other, ok := seenRemoteKeys[remoteKey]; ok {
			return nil, errors.Errorf("duplicate document %q from %q and %q (rename one file or upload to different remote directories)", docName, other, in.AbsPath)
		}
		seenRemoteKeys[remoteKey] = in.AbsPath

		info, err := os.Stat(in.AbsPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to stat markdown input: %s", in.AbsPath)
		}

		localDocs = append(localDocs, syncLocalDoc{
			Input:     in,
			PDFName:   pdfName,
			DocName:   docName,
			RemoteDir: dst,
			RemoteKey: remoteKey,
			ModTime:   info.ModTime(),
		})
	}

	return localDocs, nil
}
