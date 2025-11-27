package poml

import "testing"

func TestGuessMimeExtended(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"file.png", "image/png"},
		{"file.JPG", "image/jpeg"},
		{"file.gif", "image/gif"},
		{"vector.svg", "image/svg+xml"},
		{"photo.webp", "image/webp"},
		{"photo.tiff", "image/tiff"},
		{"photo.heic", "image/heic"},
		{"photo.avif", "image/avif"},
		{"unknown.bin", ""},
	}
	for _, tt := range cases {
		if got := guessMime(tt.path); got != tt.want {
			t.Fatalf("guessMime(%s) = %s, want %s", tt.path, got, tt.want)
		}
	}
}

func TestGuessMediaMimeExtended(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"sound.mp3", "audio/mpeg"},
		{"sound.wav", "audio/wav"},
		{"sound.ogg", "audio/ogg"},
		{"sound.flac", "audio/flac"},
		{"sound.m4a", "audio/mp4"},
		{"sound.aac", "audio/aac"},
		{"sound.opus", "audio/ogg; codecs=opus"},
		{"clip.mp4", "video/mp4"},
		{"clip.mov", "video/quicktime"},
		{"clip.webm", "video/webm"},
		{"clip.mpeg", "video/mpeg"},
		{"clip.mpg", "video/mpeg"},
		{"clip.m4v", "video/x-m4v"},
		{"clip.avi", "video/x-msvideo"},
		{"clip.mkv", "video/x-matroska"},
		{"clip.3gp", "video/3gpp"},
		{"unknown.bin", "application/octet-stream"},
	}
	for _, tt := range cases {
		if got := guessMediaMime(tt.path); got != tt.want {
			t.Fatalf("guessMediaMime(%s) = %s, want %s", tt.path, got, tt.want)
		}
	}
}
