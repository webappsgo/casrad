#!/bin/bash

# Generate test audio files for CASRAD
# Creates simple sine wave test files using ffmpeg

echo "Generating test audio files..."

cd /app/test-music || exit 1

# Generate 3 test tracks using ffmpeg
ffmpeg -f lavfi -i "sine=frequency=440:duration=30" -metadata title="Test Track 1" -metadata artist="CASRAD Test" -metadata album="Test Album" test-track-1.mp3 2>/dev/null
ffmpeg -f lavfi -i "sine=frequency=523:duration=30" -metadata title="Test Track 2" -metadata artist="CASRAD Test" -metadata album="Test Album" test-track-2.mp3 2>/dev/null
ffmpeg -f lavfi -i "sine=frequency=659:duration=30" -metadata title="Test Track 3" -metadata artist="CASRAD Test" -metadata album="Test Album" test-track-3.mp3 2>/dev/null

# Generate a longer track
ffmpeg -f lavfi -i "sine=frequency=440:duration=180" -metadata title="Long Test Track" -metadata artist="CASRAD Test" -metadata album="Test Album 2" long-test-track.mp3 2>/dev/null

echo "Test audio files generated:"
ls -la *.mp3