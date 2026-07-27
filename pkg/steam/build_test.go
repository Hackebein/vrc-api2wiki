package steam

import "testing"

func TestExtractBuildFromBytes(t *testing.T) {
	data := []byte("noise VRChat Build: 2025.3.1-1675-93d462e81f-Release more")
	cb, err := ExtractBuildFromBytes(data, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2025.3.1" || cb.BuildNumber != "1675" || cb.BuildHash != "93d462e81f" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != "windows" {
		t.Fatalf("branch %q", cb.Branch)
	}
}

func TestExtractBuildPatchSuffix(t *testing.T) {
	data := []byte("2026.2.3p3-1867-42912f4b5c-Release")
	cb, err := ExtractBuildFromBytes(data, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.2.3p3" || cb.BuildNumber != "1867" || cb.BuildHash != "42912f4b5c" {
		t.Fatalf("%+v", cb)
	}
}

func TestExtractBuildMissing(t *testing.T) {
	_, err := ExtractBuildFromBytes([]byte("nothing here"), "windows")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtractHighestBuildFromBytes(t *testing.T) {
	data := []byte(`
		2026.2.1p4-1839-94254b59aa-Release
		2026.2.3p3-1867-62fb4319cb-Release
		2026.2.3p1-1865-f71d38272d-Release
	`)
	cb, err := ExtractHighestBuildFromBytes(data, "android-google-play")
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.2.3p3" || cb.BuildNumber != "1867" || cb.BuildHash != "62fb4319cb" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != "android-google-play" {
		t.Fatalf("branch %q", cb.Branch)
	}
}

func TestClientBuildPages(t *testing.T) {
	cb, err := ExtractBuildFromBytes([]byte("2026.2.3p3-1867-42912f4b5c-Release"), "windows")
	if err != nil {
		t.Fatal(err)
	}
	pages := ClientBuildPages(cb)
	if len(pages) != 3 {
		t.Fatalf("want 3 pages, got %v", pages)
	}
	if pages["version"] != "2026.2.3p3" || pages["buildNumber"] != "1867" || pages["buildHash"] != "42912f4b5c" {
		t.Fatalf("%v", pages)
	}
}
