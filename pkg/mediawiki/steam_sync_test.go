package mediawiki

import "testing"

func TestClientBuildPageTitle(t *testing.T) {
	if got := ClientBuildPageTitle("public", ""); got != "Template:ClientBuild/public" {
		t.Fatal(got)
	}
	if got := ClientBuildPageTitle("public", "version"); got != "Template:ClientBuild/public/version" {
		t.Fatal(got)
	}
	if got := ClientBuildPageTitle("android-steamos", "buildNumber"); got != "Template:ClientBuild/android-steamos/buildNumber" {
		t.Fatal(got)
	}
}
