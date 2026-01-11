package compile

import (
	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/google/uuid"
)

const (
	rmv6BlockMigrationInfo = 0x00
	rmv6BlockSceneTree     = 0x01
	rmv6BlockTreeNode      = 0x02
	rmv6BlockSceneGroup    = 0x04
	rmv6BlockSceneLine     = 0x05
	rmv6BlockAuthorIds     = 0x09
	rmv6BlockPageInfo      = 0x0A
	rmv6BlockSceneInfo     = 0x0D
)

var rmv6ZeroID = rmdoc.RMV6CrdtID{Part1: 0, Part2: 0}

const rmv6SceneGroupItemType = 0x02

func writeAuthorIdsBlock(w *rmv6Writer, authorID uint16, authorUUID uuid.UUID) error {
	return w.writeBlock(rmv6BlockAuthorIds, 1, 1, func(bw *rmv6Writer) error {
		if err := bw.writeVarUint(1); err != nil {
			return err
		}
		return bw.writeSubBlock(0, func(sw *rmv6Writer) error {
			if err := sw.writeVarUint(16); err != nil {
				return err
			}
			if err := sw.writeBytes(uuidToBytesLE(authorUUID)); err != nil {
				return err
			}
			return sw.writeUint16LE(authorID)
		})
	})
}

func writeMigrationInfoBlock(w *rmv6Writer, migrationID rmdoc.RMV6CrdtID, isDevice bool) error {
	return w.writeBlock(rmv6BlockMigrationInfo, 1, 1, func(bw *rmv6Writer) error {
		if err := bw.writeID(1, migrationID); err != nil {
			return err
		}
		return bw.writeBool(2, isDevice)
	})
}

func writePageInfoBlock(w *rmv6Writer, loads, merges, textChars, textLines, folioUses uint32) error {
	return w.writeBlock(rmv6BlockPageInfo, 0, 1, func(bw *rmv6Writer) error {
		if err := bw.writeUint32(1, loads); err != nil {
			return err
		}
		if err := bw.writeUint32(2, merges); err != nil {
			return err
		}
		if err := bw.writeUint32(3, textChars); err != nil {
			return err
		}
		if err := bw.writeUint32(4, textLines); err != nil {
			return err
		}
		return bw.writeUint32(5, folioUses)
	})
}

func writeSceneInfoBlock(w *rmv6Writer, currentLayer rmdoc.RMV6CrdtID, backgroundVisible bool, rootVisible bool, paperW uint32, paperH uint32) error {
	return w.writeBlock(rmv6BlockSceneInfo, 0, 1, func(bw *rmv6Writer) error {
		if err := bw.writeLWWID(1, rmv6ZeroID, currentLayer); err != nil {
			return err
		}
		if err := bw.writeLWWBool(2, rmv6ZeroID, backgroundVisible); err != nil {
			return err
		}
		if err := bw.writeLWWBool(3, rmv6ZeroID, rootVisible); err != nil {
			return err
		}
		return bw.writeIntPair(5, paperW, paperH)
	})
}

func writeSceneTreeBlock(w *rmv6Writer, treeID rmdoc.RMV6CrdtID, parentID rmdoc.RMV6CrdtID) error {
	return w.writeBlock(rmv6BlockSceneTree, 1, 1, func(bw *rmv6Writer) error {
		if err := bw.writeID(1, treeID); err != nil {
			return err
		}
		if err := bw.writeID(2, rmv6ZeroID); err != nil {
			return err
		}
		if err := bw.writeBool(3, true); err != nil {
			return err
		}
		return bw.writeSubBlock(4, func(sw *rmv6Writer) error {
			return sw.writeID(1, parentID)
		})
	})
}

func writeTreeNodeBlock(w *rmv6Writer, nodeID rmdoc.RMV6CrdtID, label string, labelTS rmdoc.RMV6CrdtID) error {
	return w.writeBlock(rmv6BlockTreeNode, 1, 2, func(bw *rmv6Writer) error {
		if err := bw.writeID(1, nodeID); err != nil {
			return err
		}
		if err := bw.writeLWWString(2, labelTS, label); err != nil {
			return err
		}
		return bw.writeLWWBool(3, rmv6ZeroID, true)
	})
}

func writeSceneGroupItemBlock(w *rmv6Writer, parentID rmdoc.RMV6CrdtID, itemID rmdoc.RMV6CrdtID, leftID rmdoc.RMV6CrdtID, rightID rmdoc.RMV6CrdtID, groupID rmdoc.RMV6CrdtID) error {
	return w.writeBlock(rmv6BlockSceneGroup, 1, 1, func(bw *rmv6Writer) error {
		if err := bw.writeID(1, parentID); err != nil {
			return err
		}
		if err := bw.writeID(2, itemID); err != nil {
			return err
		}
		if err := bw.writeID(3, leftID); err != nil {
			return err
		}
		if err := bw.writeID(4, rightID); err != nil {
			return err
		}
		if err := bw.writeUint32(5, 0); err != nil {
			return err
		}
		return bw.writeSubBlock(6, func(sw *rmv6Writer) error {
			if err := sw.writeUint8(rmv6SceneGroupItemType); err != nil {
				return err
			}
			return sw.writeID(2, groupID)
		})
	})
}

func writeSceneLineItemBlock(w *rmv6Writer, parentID rmdoc.RMV6CrdtID, itemID rmdoc.RMV6CrdtID, leftID rmdoc.RMV6CrdtID, rightID rmdoc.RMV6CrdtID, payload []byte) error {
	return w.writeBlock(rmv6BlockSceneLine, 2, 2, func(bw *rmv6Writer) error {
		if err := bw.writeID(1, parentID); err != nil {
			return err
		}
		if err := bw.writeID(2, itemID); err != nil {
			return err
		}
		if err := bw.writeID(3, leftID); err != nil {
			return err
		}
		if err := bw.writeID(4, rightID); err != nil {
			return err
		}
		if err := bw.writeUint32(5, 0); err != nil {
			return err
		}
		return bw.writeSubBlock(6, func(sw *rmv6Writer) error {
			if err := sw.writeUint8(rmv6LineItemType); err != nil {
				return err
			}
			return sw.writeBytes(payload)
		})
	})
}
