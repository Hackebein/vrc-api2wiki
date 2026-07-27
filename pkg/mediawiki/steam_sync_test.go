package mediawiki

import "testing"

func TestClientBuildPageTitle(t *testing.T) {
	if got := ClientBuildPageTitle("windows", ""); got != "Template:ClientBuild/windows" {
		t.Fatal(got)
	}
	if got := ClientBuildPageTitle("windows", "version"); got != "Template:ClientBuild/windows/version" {
		t.Fatal(got)
	}
	if got := ClientBuildPageTitle("android-google-play", "buildNumber"); got != "Template:ClientBuild/android-google-play/buildNumber" {
		t.Fatalf("%q", got)
	}
	if got := ClientBuildPageTitle("android-steamos", "buildNumber"); got != "Template:ClientBuild/android-steamos/buildNumber" {
		t.Fatal(got)
	}
}
