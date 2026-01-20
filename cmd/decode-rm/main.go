package main

import (
"fmt"
"os"

"github.com/go-go-golems/remarquee/pkg/rmdoc"
)

func main() {
if len(os.Args) < 2 {
fmt.Println("Usage: decode-rm <file.rm>")
os.Exit(1)
}

filename := os.Args[1]

fmt.Printf("Decoding: %s\n\n", filename)

	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	
	tree, err := rmdoc.ParseRMV6SceneTree(f)
	if err != nil {
		fmt.Printf("Error parsing: %v\n", err)
		os.Exit(1)
	}

fmt.Printf("Root ID: %+v\n", tree.RootID)
fmt.Printf("Root: %+v\n", tree.Root)
fmt.Printf("\nGroups: %d\n", len(tree.Groups()))

for i, group := range tree.Groups() {
fmt.Printf("\nGroup %d:\n", i)
fmt.Printf("  NodeID: %+v\n", group.NodeID)
		items, err := group.Children.Items()
		if err != nil {
			fmt.Printf("  Error getting items: %v\n", err)
			continue
		}
		fmt.Printf("  Children count: %d\n", len(items))

	for j, item := range items {
fmt.Printf("\n  Item %d:\n", j)
		fmt.Printf("    ItemID: %+v\n", item.ItemID)
		fmt.Printf("    LeftID: %+v\n", item.LeftID)
		fmt.Printf("    RightID: %+v\n", item.RightID)
fmt.Printf("    DeletedLength: %d\n", item.DeletedLength)
fmt.Printf("    Value Kind: %v\n", item.Value.Kind)

if item.Value.Line != nil {
fmt.Printf("    Line BlockVersion: %d\n", item.Value.Line.BlockVersion)
fmt.Printf("    Line Raw length: %d bytes\n", len(item.Value.Line.Raw))
}
}
}
}
