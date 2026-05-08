#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
VOICE_URL="${1:-http://localhost:8003/v1/voice/recognize}"
AUDIO_FILE="$ROOT_DIR/samples/voice/sample-dining.wav"

if [[ ! -f "$AUDIO_FILE" ]]; then
  echo "Missing sample audio: $AUDIO_FILE"
  exit 1
fi

echo "POST $VOICE_URL"
curl -sS -X POST "$VOICE_URL" -F "audio=@$AUDIO_FILE;type=audio/wav" | python3 -m json.tool
