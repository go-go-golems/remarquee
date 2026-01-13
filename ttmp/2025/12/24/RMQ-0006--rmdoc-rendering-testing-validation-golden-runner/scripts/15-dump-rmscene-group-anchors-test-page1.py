#!/usr/bin/env python3
"""
RMQ-0006 helper: dump rmscene group anchors for Test.rmdoc page 1.

Goal: Compare what rmscene sees (ground truth) vs what our Go parser extracts.

Run (from remarks repo, after poetry install):
  poetry run python /path/to/15-dump-rmscene-group-anchors-test-page1.py
"""

import zipfile

from rmscene import SceneTree, build_tree, read_blocks


FIXTURE = "/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/Test.rmdoc"

# Observed page 1 pageID from our Go parsing (fixture is stable).
PAGE_ID = "df87211e-12cc-46f7-9681-9840e995bea3"


def walk_group(g, indent=0):
    pad = "  " * indent
    nid = getattr(g, "node_id", None)
    aid = getattr(g, "anchor_id", None)
    aox = getattr(g, "anchor_origin_x", None)

    print(
        f"{pad}group node_id={nid} anchor_id={aid} anchor_origin_x={aox}"
    )

    for child_id in getattr(g, "children", {}):
        child = g.children[child_id]
        if child.__class__.__name__ == "Group":
            walk_group(child, indent + 1)


def main():
    with zipfile.ZipFile(FIXTURE, "r") as z:
        rm_name = f"{PAGE_ID}.rm"
        if rm_name not in z.namelist():
            raise SystemExit(f"missing {rm_name} in rmdoc zip; found={len(z.namelist())} entries")
        rm_bytes = z.read(rm_name)

    blocks = list(read_blocks(rm_bytes))
    tree = SceneTree()
    build_tree(tree, blocks)

    print(f"fixture={FIXTURE}")
    print(f"page_id={PAGE_ID}")
    print("root:", tree.root)
    walk_group(tree.root, 0)


if __name__ == "__main__":
    main()


