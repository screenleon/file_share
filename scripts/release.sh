#!/bin/sh
# Usage: ./scripts/release.sh <major|minor|patch>
# Example: ./scripts/release.sh patch   → 0.1.0 → 0.1.1
#          ./scripts/release.sh minor   → 0.1.0 → 0.2.0
#          ./scripts/release.sh major   → 0.1.0 → 1.0.0
set -e

BUMP=${1:-patch}
CURRENT=$(cat VERSION)

MAJOR=$(echo "$CURRENT" | cut -d. -f1)
MINOR=$(echo "$CURRENT" | cut -d. -f2)
PATCH=$(echo "$CURRENT" | cut -d. -f3)

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
  *) echo "Usage: $0 <major|minor|patch>"; exit 1 ;;
esac

NEW="$MAJOR.$MINOR.$PATCH"
TAG="v$NEW"

echo "Bumping $CURRENT → $NEW"

# Update VERSION file
echo "$NEW" > VERSION

# Commit and tag
git add VERSION
git commit -m "chore: release $TAG"
git tag "$TAG"

echo ""
echo "Done. Push with:"
echo "  git push origin main && git push origin $TAG"
