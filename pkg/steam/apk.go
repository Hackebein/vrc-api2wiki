package steam

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractBuildFromAPK(apkPath, label string) (*ClientBuild, error) {
	zr, err := zip.OpenReader(apkPath)
	if err != nil {
		return nil, fmt.Errorf("open apk %s: %w", apkPath, err)
	}
	defer zr.Close()

	candidates := []string{
		"assets/bin/Data/globalgamemanagers",
		"assets/bin/Data/data.unity3d",
	}
	var lastErr error
	for _, name := range candidates {
		f, err := zr.Open(name)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			lastErr = err
			continue
		}
		cb, err := ExtractBuildFromBytes(data, label)
		if err != nil {
			lastErr = err
			continue
		}
		return cb, nil
	}

	for _, zf := range zr.File {
		base := filepath.Base(zf.Name)
		if !strings.EqualFold(base, "globalgamemanagers") && !strings.HasSuffix(strings.ToLower(zf.Name), ".dll") {
			continue
		}
		if zf.UncompressedSize64 == 0 || zf.UncompressedSize64 > 64<<20 {
			continue
		}
		f, err := zf.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			continue
		}
		cb, err := ExtractBuildFromBytes(data, label)
		if err != nil {
			lastErr = err
			continue
		}
		return cb, nil
	}

	data, err := os.ReadFile(apkPath)
	if err != nil {
		if lastErr != nil {
			return nil, fmt.Errorf("extract build from apk %s: %w", apkPath, lastErr)
		}
		return nil, err
	}
	cb, err := ExtractBuildFromBytes(data, label)
	if err != nil {
		if lastErr != nil {
			return nil, fmt.Errorf("extract build from apk %s: %w", apkPath, lastErr)
		}
		return nil, err
	}
	return cb, nil
}

func ExtractBuildFromAndroidDir(dir, label string) (*ClientBuild, error) {
	apk := filepath.Join(dir, "VRChat.apk")
	if _, err := os.Stat(apk); err != nil {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.apk"))
		if len(matches) == 0 {
			return nil, fmt.Errorf("no APK in %s", dir)
		}
		apk = matches[0]
	}
	return ExtractBuildFromAPK(apk, label)
}
