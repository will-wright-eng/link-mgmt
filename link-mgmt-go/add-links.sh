#!/bin/bash

# Script to add multiple links from a text file
# Usage: ./add-links.sh <links.txt>

set -euo pipefail

# Check if file argument is provided
if [ $# -eq 0 ]; then
    echo "Error: No file provided"
    echo "Usage: $0 <links.txt>"
    exit 1
fi

LINKS_FILE="$1"

# Check if file exists
if [ ! -f "$LINKS_FILE" ]; then
    echo "Error: File '$LINKS_FILE' does not exist"
    exit 1
fi

# Check if file is readable
if [ ! -r "$LINKS_FILE" ]; then
    echo "Error: File '$LINKS_FILE' is not readable"
    exit 1
fi

# Get the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Count total links
TOTAL=$(grep -v '^[[:space:]]*$' "$LINKS_FILE" | grep -v '^[[:space:]]*#' | wc -l | tr -d ' ')
echo "Found $TOTAL link(s) to add from '$LINKS_FILE'"
echo ""

# Counter for tracking progress
COUNT=0
SUCCESS=0
FAILED=0

# Loop through each line in the file
while IFS= read -r link || [ -n "$link" ]; do
    # Skip empty lines and comments
    if [[ -z "$link" ]] || [[ "$link" =~ ^[[:space:]]*# ]]; then
        continue
    fi

    # Trim leading and trailing whitespace (handles quotes safely)
    link="${link#"${link%%[![:space:]]*}"}"
    link="${link%"${link##*[![:space:]]}"}"

    # Skip if empty after trimming
    if [ -z "$link" ]; then
        continue
    fi

    COUNT=$((COUNT + 1))
    echo "[$COUNT/$TOTAL] Adding: $link"

    # Run the add command (properly quote the URL)
    if go run ./cmd/cli --save "$link"; then
        echo "✓ Successfully added: $link"
        echo ""
        SUCCESS=$((SUCCESS + 1))
    else
        echo "✗ Failed to add: $link"
        echo ""
        FAILED=$((FAILED + 1))
    fi
done < "$LINKS_FILE"

# Summary
echo "=========================================="
echo "Summary:"
echo "  Total:   $COUNT"
echo "  Success: $SUCCESS"
echo "  Failed:  $FAILED"
echo "=========================================="

# Exit with error if any failed
if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0
