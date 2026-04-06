#!/bin/bash

# Get the latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Strip 'v' prefix for parsing
VERSION=${LATEST_TAG#v}

# Extract base version (remove pre-release tags like -test)
BASE_VERSION=$(echo "$VERSION" | cut -d'-' -f1)

# Split into array
IFS='.' read -r -a VERSION_PARTS <<< "$BASE_VERSION"
MAJOR=${VERSION_PARTS[0]:-0}
MINOR=${VERSION_PARTS[1]:-0}
PATCH=${VERSION_PARTS[2]:-0}

NEXT_PATCH="$MAJOR.$MINOR.$((PATCH + 1))"
NEXT_MINOR="$MAJOR.$((MINOR + 1)).0"
NEXT_MAJOR="$((MAJOR + 1)).0.0"

echo "? Current Version is $VERSION - choose the version increment method:"

CHOICE=$(gum choose \
    "No version update - $VERSION" \
    "Patch - $NEXT_PATCH" \
    "Minor - $NEXT_MINOR" \
    "Major - $NEXT_MAJOR" \
    "Custom"
)

# Exit if cancelled or ESC pressed
if [ -z "$CHOICE" ]; then
    echo "Release cancelled."
    exit 0
fi

if [[ "$CHOICE" == "Custom" ]]; then
    NEW_VERSION=$(gum input --placeholder "Enter new version (e.g. $NEXT_PATCH-test)")
    if [ -z "$NEW_VERSION" ]; then
        echo "Release cancelled."
        exit 0
    fi
    NEW_TAG="v${NEW_VERSION#v}" # Ensure exactly one 'v' prefix
else
    # Extract the version number from the choice string (it's the last word)
    NEW_VERSION=$(echo "$CHOICE" | awk '{print $NF}')
    NEW_TAG="v$NEW_VERSION"
fi

echo ""
if gum confirm "Create and push tag $NEW_TAG?"; then
    git tag "$NEW_TAG"
    git push origin "$NEW_TAG"
    echo "✅ Successfully released $NEW_TAG"
else
    echo "Release cancelled."
fi
