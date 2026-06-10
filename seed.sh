#!/usr/bin/env bash
set -euo pipefail

BASE=${1:-http://localhost:8000}
PASS="password123"
COOKIE_JAR=$(mktemp)

cleanup() { rm -f "$COOKIE_JAR"; }
trap cleanup EXIT

for i in $(seq 1 100); do
  USER="user${i}"
  EMAIL="${USER}@example.com"

  echo "--- Creating $USER ---"

  curl -s -c "$COOKIE_JAR" "$BASE/register" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$USER\",\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" > /dev/null

  for j in $(seq 1 $(shuf -i 3-7 -n1)); do
    curl -s -b "$COOKIE_JAR" "$BASE/notes" \
      -H 'Content-Type: application/json' \
      -d "{\"title\":\"Note $j by $USER\",\"content\":\"<p>This is note $j from $USER.</p>\"}" > /dev/null
  done

  echo "  -> $USER created with 3-7 notes"
done

echo ""
echo "Done! All passwords are: $PASS"
echo "Example login: curl -X POST \"$BASE/login\" -H 'Content-Type: application/json' -d '{\"username\":\"user1\",\"password\":\"password123\"}' -c cookies.txt"
