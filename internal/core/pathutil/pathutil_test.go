package pathutil

import (
	"testing"
)

func TestValidateContentPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{"valid movies path", "/movies/Inception (2010)/movie.mkv", false},
		{"valid data path", "/data/media/movies/file.mkv", false},
		{"valid mnt path", "/mnt/storage/movies/file.mkv", false},
		{"valid home media", "/home/user/media/file.mkv", false},

		// Empty / relative
		{"empty path", "", true},
		{"relative path", "movies/file.mkv", true},
		{"dot relative", "./movies/file.mkv", true},

		// Sensitive directories
		{"config dir", "/config/something", true},
		{"config exact", "/config", true},
		{"etc dir", "/etc/passwd", true},
		{"proc dir", "/proc/self/environ", true},
		{"sys dir", "/sys/kernel", true},
		{"dev dir", "/dev/null", true},
		{"run dir", "/run/secrets", true},
		{"var dir", "/var/log/syslog", true},

		// Traversal attempts
		{"traversal cleaned", "/movies/../etc/passwd", true},
		{"double traversal", "/movies/../../etc/shadow", true},

		// Paths that look like sensitive dirs but aren't
		{"configuration ok", "/configuration/file", false},
		{"etcetera ok", "/etcetera/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContentPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContentPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
