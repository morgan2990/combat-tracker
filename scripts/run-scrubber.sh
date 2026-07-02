#!/usr/bin/env bash
#
# Runs the monster scrubber against a local 5etools checkout, sourcing its
# MongoDB/Typesense credentials from either .env (dev) or .env.production
# (prod). Bash port of run-scrubber.ps1 — same behavior.
#
# The scrubber (cmd/scrubber) reads MONGODB_URI, TYPESENSE_URL, and
# TYPESENSE_API_KEY from the environment - it does not load an env file
# itself. This script picks which file to load based on --environment:
#   dev  -> .env             (already used for local `go run .`)
#   prod -> .env.production  (gitignored; you maintain this by hand)
# and exports those values for this process only, then invokes:
#   go run ./cmd/scrubber --source <path> --edition <edition>
#
# Usage:
#   ./scripts/run-scrubber.sh
#   ./scripts/run-scrubber.sh --environment prod --edition 5e --source-path /path/to/5etools-src
#
# Options:
#   --environment dev|prod   Prompted for (default "dev") if omitted.
#   --edition 5e|5.5e        Prompted for (default "5e") if omitted.
#   --source-path PATH       If omitted, defaults to SCRUBBER_SOURCE_5E or
#                             SCRUBBER_SOURCE_5_5E (whichever matches
#                             --edition) from the env file, and prompts only
#                             if that's also unset.

set -euo pipefail

usage() {
    sed -n '2,25p' "$0" | sed 's/^# \{0,1\}//'
}

ENVIRONMENT=""
EDITION=""
SOURCE_PATH=""

while [ $# -gt 0 ]; do
    case "$1" in
        --environment) ENVIRONMENT="$2"; shift 2 ;;
        --environment=*) ENVIRONMENT="${1#*=}"; shift ;;
        --edition) EDITION="$2"; shift 2 ;;
        --edition=*) EDITION="${1#*=}"; shift ;;
        --source-path) SOURCE_PATH="$2"; shift 2 ;;
        --source-path=*) SOURCE_PATH="${1#*=}"; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

# Reads KEY=value out of an env file (last match wins, whitespace trimmed).
# Blank if the file or key doesn't exist.
read_env_var() {
    local file="$1" key="$2" line val
    [ -f "$file" ] || return 0
    line=$(grep -E "^[[:space:]]*${key}[[:space:]]*=" "$file" | tail -n1) || true
    [ -z "$line" ] && return 0
    val="${line#*=}"
    val="$(printf '%s' "$val" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    printf '%s' "$val"
}

read_with_default() {
    local prompt="$1" default="$2" response
    if [ -n "$default" ]; then
        read -r -p "$prompt [$default]: " response
    else
        read -r -p "$prompt: " response
    fi
    if [ -z "$response" ]; then
        printf '%s' "$default"
    else
        printf '%s' "$response"
    fi
}

echo "== Monster scrubber setup =="
echo "Populates MongoDB and Typesense from a local 5etools data checkout."
echo ""

if [ -z "$ENVIRONMENT" ]; then
    ENVIRONMENT="$(read_with_default 'Environment (dev or prod)' 'dev')"
fi
if [ "$ENVIRONMENT" != "dev" ] && [ "$ENVIRONMENT" != "prod" ]; then
    echo "Environment must be 'dev' or 'prod', got '$ENVIRONMENT'." >&2
    exit 1
fi

if [ -z "$EDITION" ]; then
    EDITION="$(read_with_default 'Edition (5e or 5.5e)' '5e')"
fi
if [ "$EDITION" != "5e" ] && [ "$EDITION" != "5.5e" ]; then
    echo "Edition must be '5e' or '5.5e', got '$EDITION'." >&2
    exit 1
fi

if [ "$ENVIRONMENT" = "dev" ]; then
    ENV_FILE_NAME=".env"
else
    ENV_FILE_NAME=".env.production"
fi
ENV_FILE_PATH="$REPO_ROOT/$ENV_FILE_NAME"
if [ ! -f "$ENV_FILE_PATH" ]; then
    echo "$ENV_FILE_NAME not found at $ENV_FILE_PATH - create it first (see .env.example for the expected keys)." >&2
    exit 1
fi

MONGO_URI="$(read_env_var "$ENV_FILE_PATH" MONGODB_URI)"
TYPESENSE_URL_VAL="$(read_env_var "$ENV_FILE_PATH" TYPESENSE_URL)"
TYPESENSE_KEY_VAL="$(read_env_var "$ENV_FILE_PATH" TYPESENSE_API_KEY)"

if [ -z "$MONGO_URI" ]; then
    echo "MONGODB_URI not set in $ENV_FILE_NAME - the scrubber cannot write monsters without it." >&2
    exit 1
fi
if [ -z "$TYPESENSE_URL_VAL" ] || [ -z "$TYPESENSE_KEY_VAL" ]; then
    echo "TYPESENSE_URL/TYPESENSE_API_KEY incomplete in $ENV_FILE_NAME - MongoDB will still be populated, but the search index will NOT be updated." >&2
fi

echo "Using credentials from $ENV_FILE_NAME ($ENVIRONMENT)"

if [ "$EDITION" = "5e" ]; then
    SOURCE_VAR="SCRUBBER_SOURCE_5E"
else
    SOURCE_VAR="SCRUBBER_SOURCE_5_5E"
fi

if [ -z "$SOURCE_PATH" ]; then
    SOURCE_PATH="$(read_env_var "$ENV_FILE_PATH" "$SOURCE_VAR")"
    if [ -n "$SOURCE_PATH" ]; then
        echo "Using $SOURCE_VAR from $ENV_FILE_NAME as source path: $SOURCE_PATH"
    else
        read -r -p "Path to local 5etools repository root: " SOURCE_PATH
    fi
fi

if [ -z "$SOURCE_PATH" ]; then
    echo "No source path given, and $SOURCE_VAR is not set in $ENV_FILE_NAME." >&2
    exit 1
fi
if [ ! -d "$SOURCE_PATH" ]; then
    echo "Source path not found: $SOURCE_PATH" >&2
    exit 1
fi

export MONGODB_URI="$MONGO_URI"
export TYPESENSE_URL="$TYPESENSE_URL_VAL"
export TYPESENSE_API_KEY="$TYPESENSE_KEY_VAL"

echo ""
echo "Running scrubber: --source \"$SOURCE_PATH\" --edition $EDITION"
echo ""

cd "$REPO_ROOT"
go run ./cmd/scrubber --source "$SOURCE_PATH" --edition "$EDITION"
