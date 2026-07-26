package steam

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractBuildFromAPK(t *testing.T) {
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "VRChat.apk")
	f, err := os.Create(apkPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("assets/bin/Data/globalgamemanagers")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("pad 2026.2.3-1862-c97c6f631a-Release pad")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	cb, err := ExtractBuildFromAPK(apkPath, "android-steamos")
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.2.3" || cb.BuildNumber != "1862" || cb.BuildHash != "c97c6f631a" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != "android-steamos" {
		t.Fatalf("branch %q", cb.Branch)
	}
}
