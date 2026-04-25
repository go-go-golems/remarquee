#!/bin/bash
# scan-activity.sh — Quick reMarkable cloud activity scanner
# Usage: ./scan-activity.sh /ai/2026/04  (scans all day subdirectories)
# 
# For each day folder, lists files with modified_client timestamps (UTC→EDT).
# No downloads required — uses cloud ls JSON only.

set -euo pipefail

BASE="${1:?Usage: $0 <cloud-path-prefix> (e.g. /ai/2026/04)}"
EDT_OFFSET=-4

# Get list of day directories
echo "Scanning $BASE ..."
echo ""

days=$(remarquee cloud ls "$BASE" --with-glaze-output --output json 2>/dev/null \
  | jq -r '.[] | select(.is_dir == true) | .name' | sort)

for day in $days; do
    remarquee cloud ls "$BASE/$day" --with-glaze-output --output json 2>/dev/null \
      | jq -r '.[] | "\(.modified_client // "?") \(.is_dir // false) \(.name)"' \
      | while IFS= read -r line; do
        ts=$(echo "$line" | awk '{print $1}')
        is_dir=$(echo "$line" | awk '{print $2}')
        name=$(echo "$line" | awk '{$1=""; $2=""; print}' | sed 's/^ *//')
        
        # Convert UTC to EDT
        if [ "$ts" != "?" ]; then
            edt=$(date -d "$ts $EDT_OFFSET hours" "+%H:%M" 2>/dev/null || echo "$ts")
        else
            edt="??:??"
        fi
        
        if [ "$is_dir" = "true" ]; then
            prefix="📁"
        else
            prefix="📄"
        fi
        
        echo "$edt  $prefix $name"
    done | sort
    
    echo ""
done
