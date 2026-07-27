package mediawiki

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/meta"
	"github.com/Hackebein/vrc-api2wiki/pkg/pico"
	"github.com/Hackebein/vrc-api2wiki/pkg/playstore"
	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
	"github.com/Hackebein/vrc-api2wiki/pkg/vcc"
	"github.com/Hackebein/vrc-api2wiki/pkg/viveport"
	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

func ClientBuildPageTitle(name, subpath string) string {
	if subpath == "" {
		return "Template:ClientBuild/" + name
	}
	return "Template:ClientBuild/" + name + "/" + subpath
}

func RunSteamSync(wiki *MediaWikiClient, logger *slog.Logger) error {
	root, err := steam.FindRepoRoot()
	if err != nil {
		return err
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	vrc := vrchat.NewClient(httpClient)
	mins, sdkUnity, err := vrc.GetClientBuildConfig()
	if err != nil {
		return fmt.Errorf("vrchat config: %w", err)
	}
	if logger != nil {
		logger.Info("vrchat config loaded", "platforms", len(mins), "sdkUnityVersion", sdkUnity)
	}
	if err := syncUnitySDK(wiki, root, sdkUnity, logger); err != nil {
		return err
	}
	if err := syncCreatorCompanionTools(wiki, root, logger); err != nil {
		return err
	}

	if err := syncMetaQuestAndroid(wiki, root, mins, logger); err != nil {
		return err
	}
	if err := syncPicoAndroid(wiki, root, mins, logger); err != nil {
		return err
	}
	if err := syncGooglePlayAndroid(wiki, root, mins, logger); err != nil {
		return err
	}
	if err := syncViveport(wiki, root, mins, logger); err != nil {
		return err
	}

	username := strings.TrimSpace(os.Getenv("STEAM_USERNAME"))
	password := strings.TrimSpace(os.Getenv("STEAM_PASSWORD"))
	shared := strings.TrimSpace(os.Getenv("STEAM_SHARED_SECRET"))
	if username == "" || password == "" {
		if logger != nil {
			logger.Info("skipping steam depot sync: STEAM_USERNAME/STEAM_PASSWORD not set")
		}
	} else if shared == "" {
		return fmt.Errorf("STEAM_SHARED_SECRET is required for steam depot sync (mobile authenticator shared_secret, base64)")
	} else {
		dd := steam.DepotDownloaderPath(root)
		if st, err := os.Stat(dd); err != nil || st.IsDir() {
			return fmt.Errorf("shipped DepotDownloader missing at %s (run scripts/fetch-depotdownloader.sh)", dd)
		}

		for _, client := range steam.DefaultWindowsClients {
			client := client
			if err := syncSteamClient(wiki, root, client.ClientName, mins, logger,
				func(outDir string) (string, string, error) {
					return steam.DownloadBranch(dd, client.SteamBranch, outDir, username, password, shared)
				},
				func(outDir string) (*steam.ClientBuild, error) {
					return steam.ExtractBuildFromDir(outDir, client.ClientName)
				},
			); err != nil {
				return err
			}
		}

		androidName := steam.AndroidClientName
		if err := syncSteamClient(wiki, root, androidName, mins, logger,
			func(outDir string) (string, string, error) {
				return steam.DownloadAndroidDepot(dd, outDir, username, password, shared)
			},
			func(outDir string) (*steam.ClientBuild, error) {
				return steam.ExtractBuildFromAndroidDir(outDir, androidName)
			},
		); err != nil {
			return err
		}
	}

	return nil
}

func syncUnitySDK(wiki *MediaWikiClient, root, version string, logger *slog.Logger) error {
	return writeToolVersion(wiki, root, vrchat.UnitySDKClientName, version, logger)
}

func syncCreatorCompanionTools(wiki *MediaWikiClient, root string, logger *slog.Logger) error {
	httpClient := &http.Client{Timeout: 60 * time.Second}

	if logger != nil {
		logger.Info("fetching VRChat Creator Companion version")
	}
	stable, err := vcc.FetchCreatorCompanion(httpClient)
	if err != nil {
		return fmt.Errorf("creator companion version: %w", err)
	}
	if err := writeToolVersion(wiki, root, vcc.CreatorCompanionClientName, stable.Version, logger); err != nil {
		return err
	}

	if logger != nil {
		logger.Info("fetching VRChat Creator Companion beta version")
	}
	beta, err := vcc.FetchCreatorCompanionBeta(httpClient)
	if err != nil {
		return fmt.Errorf("creator companion beta version: %w", err)
	}
	if err := writeToolVersion(wiki, root, vcc.CreatorCompanionBetaClientName, beta.Version, logger); err != nil {
		return err
	}

	if logger != nil {
		logger.Info("fetching VRC Quick Launcher version")
	}
	ql, err := vcc.FetchQuickLauncher(httpClient)
	if err != nil {
		return fmt.Errorf("quick launcher version: %w", err)
	}
	return writeToolVersion(wiki, root, vcc.QuickLauncherClientName, ql.Version, logger)
}

func writeToolVersion(wiki *MediaWikiClient, root, name, version string, logger *slog.Logger) error {
	build := &steam.ClientBuild{
		Version:   version,
		Branch:    name,
		FetchedAt: time.Now().UTC(),
	}
	jsonPath := filepath.Join(root, "steam-output", name+".json")
	if err := steam.WriteBuildJSON(jsonPath, build); err != nil {
		return err
	}
	versionTitle := ClientBuildPageTitle(name, "version")
	if err := wiki.EditPage(versionTitle, version, true); err != nil {
		return fmt.Errorf("edit %s: %w", versionTitle, err)
	}
	marker := fmt.Sprintf("{{ClientBuild/%s/version}}", name)
	if err := wiki.EditPage(ClientBuildPageTitle(name, ""), marker, true); err != nil {
		return fmt.Errorf("edit %s: %w", ClientBuildPageTitle(name, ""), err)
	}
	if logger != nil {
		logger.Info("tool version written",
			"client", name,
			"version", version,
			"json", jsonPath)
	}
	return nil
}

func syncMetaQuestAndroid(wiki *MediaWikiClient, root string, mins map[string]int, logger *slog.Logger) error {
	httpClient := &http.Client{Timeout: 60 * time.Second}
	if logger != nil {
		logger.Info("fetching Meta Quest LIVE version", "appId", meta.VRChatQuestAppID)
	}
	live, err := meta.FetchVRChatQuestBuild(httpClient)
	if err != nil {
		return fmt.Errorf("meta quest live version: %w", err)
	}
	if err := writeClientBuild(wiki, root, meta.QuestClientName, live, mins, logger); err != nil {
		return err
	}
	if logger != nil {
		logger.Info("fetching Meta Quest Open Beta version", "appId", meta.VRChatQuestAppID)
	}
	openBeta, err := meta.FetchVRChatQuestOpenBetaBuild(httpClient)
	if err != nil {
		return fmt.Errorf("meta quest open beta version: %w", err)
	}
	return writeClientBuild(wiki, root, meta.QuestOpenBetaClientName, openBeta, mins, logger)
}

func syncPicoAndroid(wiki *MediaWikiClient, root string, mins map[string]int, logger *slog.Logger) error {
	httpClient := &http.Client{Timeout: 60 * time.Second}
	if logger != nil {
		logger.Info("fetching Pico Store LIVE version", "appId", pico.VRChatPicoAppID)
	}
	build, err := pico.FetchVRChatPicoBuild(httpClient)
	if err != nil {
		return fmt.Errorf("pico store live version: %w", err)
	}
	return writeClientBuild(wiki, root, pico.PicoClientName, build, mins, logger)
}

func syncGooglePlayAndroid(wiki *MediaWikiClient, root string, mins map[string]int, logger *slog.Logger) error {
	httpClient := &http.Client{Timeout: 60 * time.Second}
	if logger != nil {
		logger.Info("fetching Google Play version", "package", playstore.VRChatPlayStorePackageID)
	}
	build, err := playstore.FetchVRChatPlayStoreBuild(httpClient)
	if err != nil {
		return fmt.Errorf("google play version: %w", err)
	}
	return writeClientBuild(wiki, root, playstore.PlayStoreClientName, build, mins, logger)
}

func syncViveport(wiki *MediaWikiClient, root string, mins map[string]int, logger *slog.Logger) error {
	httpClient := &http.Client{Timeout: 60 * time.Second}
	if logger != nil {
		logger.Info("fetching Viveport Android LIVE version", "appId", viveport.VRChatAndroidAppID)
	}
	androidLive, err := viveport.FetchAndroidBuild(httpClient)
	if err != nil {
		return fmt.Errorf("viveport android live version: %w", err)
	}
	if err := writeClientBuild(wiki, root, viveport.AndroidClientName, androidLive, mins, logger); err != nil {
		return err
	}

	if logger != nil {
		logger.Info("fetching Viveport Android Open Beta version", "appId", viveport.VRChatAndroidAppID)
	}
	androidBeta, err := viveport.FetchAndroidOpenBetaBuild(httpClient)
	if err != nil {
		return fmt.Errorf("viveport android open beta version: %w", err)
	}
	if viveport.BuildsDiffer(androidLive, androidBeta) {
		if err := writeClientBuild(wiki, root, viveport.AndroidOpenBetaClientName, androidBeta, mins, logger); err != nil {
			return err
		}
	} else if logger != nil {
		logger.Info("skipping Viveport Android Open Beta: same as LIVE", "client", viveport.AndroidOpenBetaClientName)
	}

	if logger != nil {
		logger.Info("fetching Viveport Windows LIVE version", "appId", viveport.VRChatWindowsAppID)
	}
	windowsLive, err := viveport.FetchWindowsBuild(httpClient)
	if err != nil {
		if logger != nil {
			logger.Info("skipping Viveport Windows LIVE: incomplete or unavailable version", "err", err)
		}
		return nil
	}
	if err := writeClientBuild(wiki, root, viveport.WindowsClientName, windowsLive, mins, logger); err != nil {
		return err
	}

	if logger != nil {
		logger.Info("fetching Viveport Windows Open Beta version", "appId", viveport.VRChatWindowsAppID)
	}
	windowsBeta, err := viveport.FetchWindowsOpenBetaBuild(httpClient)
	if err != nil {
		if logger != nil {
			logger.Info("skipping Viveport Windows Open Beta: incomplete or unavailable version", "err", err)
		}
		return nil
	}
	if viveport.BuildsDiffer(windowsLive, windowsBeta) {
		return writeClientBuild(wiki, root, viveport.WindowsOpenBetaClientName, windowsBeta, mins, logger)
	}
	if logger != nil {
		logger.Info("skipping Viveport Windows Open Beta: same as LIVE", "client", viveport.WindowsOpenBetaClientName)
	}
	return nil
}

func syncSteamClient(
	wiki *MediaWikiClient,
	root, name string,
	mins map[string]int,
	logger *slog.Logger,
	download func(outDir string) (manifestID, steamBuildID string, err error),
	extract func(outDir string) (*steam.ClientBuild, error),
) error {
	outDir := filepath.Join(root, "steam-output", "depots", name)
	if logger != nil {
		logger.Info("steam depot download", "client", name, "dir", outDir)
	}
	manifestID, steamBuildID, err := download(outDir)
	if err != nil {
		return err
	}
	build, err := extract(outDir)
	if err != nil {
		return fmt.Errorf("extract build %s: %w", name, err)
	}
	if manifestID != "" {
		build.ManifestID = manifestID
	}
	if steamBuildID != "" {
		build.SteamBuildID = steamBuildID
	}
	return writeClientBuild(wiki, root, name, build, mins, logger)
}

func writeClientBuild(wiki *MediaWikiClient, root, name string, build *steam.ClientBuild, mins map[string]int, logger *slog.Logger) error {
	min, err := vrchat.MinBuildForClient(mins, name)
	if err != nil {
		return err
	}
	extracted := build.BuildNumber
	if vrchat.ClearClientBuildIfBelowMin(build, min) {
		if logger != nil {
			logger.Info("client build below min; clearing version",
				"client", name,
				"platform", vrchat.ClientConfigPlatform(name),
				"extractedBuildNumber", extracted,
				"minBuildNumber", min,
				"rawMatch", build.RawMatch)
		}
	}

	jsonPath := filepath.Join(root, "steam-output", name+".json")
	if err := steam.WriteBuildJSON(jsonPath, build); err != nil {
		return err
	}

	pages := steam.ClientBuildPages(build)
	for subpath, value := range pages {
		title := ClientBuildPageTitle(name, subpath)
		if err := wiki.EditPage(title, value, true); err != nil {
			return fmt.Errorf("edit %s: %w", title, err)
		}
	}
	marker := fmt.Sprintf("{{ClientBuild/%s/version}} ({{ClientBuild/%s/buildNumber}})", name, name)
	if err := wiki.EditPage(ClientBuildPageTitle(name, ""), marker, true); err != nil {
		return fmt.Errorf("edit %s: %w", ClientBuildPageTitle(name, ""), err)
	}

	if logger != nil {
		logger.Info("client build written",
			"client", name,
			"version", build.Version,
			"buildNumber", build.BuildNumber,
			"buildHash", build.BuildHash,
			"minBuildNumber", min,
			"pages", len(pages),
			"json", jsonPath)
	}
	return nil
}
