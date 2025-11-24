#!/usr/bin/env bash
set -euo pipefail

# Seed the database with sample links using the lnk CLI
# This script can be run directly or via 'make seed'

# Links to add: format is URL|TITLE|DESCRIPTION
# Use empty string "" for optional fields
LINKS=(
    "https://blog.cloudflare.com/18-november-2025-outage/|Cloudflare outage on November 18, 2025|Post-mortem of Cloudflare's network outage caused by a Bot Management feature file issue"
    "https://slavoj.substack.com/p/why-we-remain-alive-also-in-a-dead-954|Why we remain alive also in a dead world|Philosophical exploration by Slavoj Žižek"
    "https://disassociated.com/personal-blogs-back-niche-blogs-next/|Personal blogs back, niche blogs next|Discussion about the resurgence of personal blogging and the future of niche blogs"
    "https://lucumr.pocoo.org/2025/11/21/agents-are-hard/|Agents are hard|Armin Ronacher's thoughts on the challenges of building AI agents"
)

# Determine which lnk command to use
if command -v lnk &> /dev/null; then
    LNK_CMD="lnk"
elif [ -f "target/release/lnk" ]; then
    LNK_CMD="./target/release/lnk"
else
    LNK_CMD="cargo run --"
fi

echo "Seeding database with ${#LINKS[@]} links..."
echo "Using: $LNK_CMD"
echo ""

for link_entry in "${LINKS[@]}"; do
    # Parse URL|TITLE|DESCRIPTION
    IFS='|' read -r url title description <<< "$link_entry"

    echo "Adding: $url"

    # Build command arguments
    ARGS=("save" "$url")
    if [ -n "$title" ]; then
        ARGS+=("--title" "$title")
    fi
    if [ -n "$description" ]; then
        ARGS+=("--description" "$description")
    fi

    # Execute the command
    if [ "$LNK_CMD" = "cargo run --" ]; then
        cargo run -- "${ARGS[@]}" || echo "  ⚠️  Failed to add: $url"
    else
        $LNK_CMD "${ARGS[@]}" || echo "  ⚠️  Failed to add: $url"
    fi
    echo ""
done

echo "✓ Database seeding complete!"
