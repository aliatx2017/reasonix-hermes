#!/usr/bin/env bash
# install-skills.sh — Install the Hermes curated skill pack into Reasonix.
#
# Usage:
#   ./scripts/install-skills.sh [--dry-run] [--target DIR]
#
# Options:
#   --dry-run    Show what would be installed without copying
#   --target DIR Override the target directory (default: auto-detected)
#
# Default target resolution:
#   1. $REASONIX_SKILLS_DIR (if set)
#   2. XDG_CONFIG_HOME/reasonix/skills/hermes/
#   3. ~/.config/reasonix/skills/hermes/
#   4. .reasonix/skills/hermes/ (project-local, if reasonix.toml exists)
#
# After installation, add to reasonix.toml:
#   [skills]
#   paths = ["~/.config/reasonix/skills/hermes"]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILLS_SRC="$SCRIPT_DIR/../skills-hub/skills"
REGISTRY_SRC="$SCRIPT_DIR/../skills-hub/registry.json"

DRY_RUN=false
TARGET=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=true; shift ;;
    --target) TARGET="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $0 [--dry-run] [--target DIR]"
      echo ""
      echo "Install the Hermes curated skill pack into Reasonix."
      echo ""
      echo "Options:"
      echo "  --dry-run    Show what would be installed without copying"
      echo "  --target DIR Override the target directory"
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# Resolve target directory
if [[ -z "$TARGET" ]]; then
  if [[ -n "${REASONIX_SKILLS_DIR:-}" ]]; then
    TARGET="$REASONIX_SKILLS_DIR/hermes"
  elif [[ -n "${XDG_CONFIG_HOME:-}" ]]; then
    TARGET="$XDG_CONFIG_HOME/reasonix/skills/hermes"
  elif [[ -f "reasonix.toml" ]]; then
    TARGET=".reasonix/skills/hermes"
  else
    TARGET="$HOME/.config/reasonix/skills/hermes"
  fi
fi

# Check source directory
if [[ ! -d "$SKILLS_SRC" ]]; then
  echo "Error: Skills source directory not found: $SKILLS_SRC" >&2
  exit 1
fi

# Count skills
SKILL_COUNT=$(find "$SKILLS_SRC" -name '*.md' -maxdepth 1 | wc -l | tr -d ' ')

echo "=== Hermes Skills Installer ==="
echo "Source:   $SKILLS_SRC"
echo "Target:  $TARGET"
echo "Skills:  $SKILL_COUNT files"
echo ""

if $DRY_RUN; then
  echo "[DRY RUN] Would install:"
  ls -1 "$SKILLS_SRC"/*.md 2>/dev/null | while read -r f; do
    echo "  $(basename "$f")"
  done
  if [[ -f "$REGISTRY_SRC" ]]; then
    echo "  registry.json"
  fi
  echo ""
  echo "[DRY RUN] Add to reasonix.toml:"
  echo "  [skills]"
  echo "  paths = [\"$TARGET\"]"
  exit 0
fi

# Create target directory
mkdir -p "$TARGET"

# Copy skills
COPIED=0
for f in "$SKILLS_SRC"/*.md; do
  [[ -f "$f" ]] || continue
  NAME=$(basename "$f")
  cp "$f" "$TARGET/$NAME"
  echo "  ✓ $NAME"
  COPIED=$((COPIED + 1))
done

# Copy registry
if [[ -f "$REGISTRY_SRC" ]]; then
  cp "$REGISTRY_SRC" "$TARGET/registry.json"
  echo "  ✓ registry.json"
fi

echo ""
echo "Installed $COPIED skills to $TARGET"
echo ""
echo "Add to reasonix.toml:"
echo "  [skills]"
echo "  paths = [\"$TARGET\"]"