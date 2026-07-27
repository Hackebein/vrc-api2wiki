package mediawiki

import (
	"testing"

	"github.com/Hackebein/vrc-api2wiki/pkg/vcc"
	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

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
	if got := ClientBuildPageTitle(vrchat.UnitySDKClientName, "version"); got != "Template:ClientBuild/unity-sdk/version" {
		t.Fatal(got)
	}
	if got := ClientBuildPageTitle(vcc.CreatorCompanionClientName, "version"); got != "Template:ClientBuild/creator-companion/version" {
		t.Fatal(got)
	}
	if got := ClientBuildPageTitle(vcc.CreatorCompanionBetaClientName, "version"); got != "Template:ClientBuild/creator-companion-beta/version" {
		t.Fatal(got)
	}
	if got := ClientBuildPageTitle(vcc.QuickLauncherClientName, "version"); got != "Template:ClientBuild/quick-launcher/version" {
		t.Fatal(got)
	}
}
