package steam

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	VRChatAppID          = "438100"
	VRChatWindowsDepotID = "438101"
	VRChatAndroidDepotID = "438102"
)

type WindowsClient struct {
	SteamBranch string
	ClientName  string
}

var DefaultWindowsClients = []WindowsClient{
	{SteamBranch: "public", ClientName: "windows"},
	{SteamBranch: "open-beta", ClientName: "open-beta-windows"},
}

const AndroidClientName = "android-steamos"

func DepotDownloaderPath(repoRoot string) string {
	name := "DepotDownloader"
	if runtime.GOOS == "windows" {
		name = "DepotDownloader.exe"
	}
	return filepath.Join(repoRoot, "third_party", "DepotDownloader", name)
}

func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func DownloadBranch(ddPath, branch, outDir, username, password, sharedSecret string) (manifestID string, steamBuildID string, err error) {
	return DownloadDepot(ddPath, VRChatWindowsDepotID, branch, outDir, username, password, sharedSecret, []string{
		"VRChat_Data/globalgamemanagers",
		"VRChat.exe",
		"UnityPlayer.dll",
		"regex:VRChat_Data/Managed/.*\\.dll",
	})
}

func DownloadAndroidDepot(ddPath, outDir, username, password, sharedSecret string) (manifestID string, steamBuildID string, err error) {
	return DownloadDepot(ddPath, VRChatAndroidDepotID, "public", outDir, username, password, sharedSecret, []string{
		"VRChat.apk",
	})
}

func DownloadDepot(ddPath, depotID, branch, outDir, username, password, sharedSecret string, filelistLines []string) (manifestID string, steamBuildID string, err error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", "", err
	}
	filelist := filepath.Join(outDir, "filelist.txt")
	if err := os.WriteFile(filelist, []byte(strings.Join(filelistLines, "\n")+"\n"), 0o644); err != nil {
		return "", "", err
	}

	code, err := GenerateSteamGuardCode(sharedSecret, time.Now())
	if err != nil {
		return "", "", err
	}

	args := []string{
		"-app", VRChatAppID,
		"-depot", depotID,
		"-dir", outDir,
		"-filelist", filelist,
		"-no-mobile",
		"-remember-password",
	}
	if branch != "" {
		args = append(args, "-branch", branch)
	}
	if username != "" {
		args = append(args, "-username", username)
		if password != "" {
			args = append(args, "-password", password)
		}
	}

	cmd := exec.Command(ddPath, args...)
	cmd.Dir = filepath.Dir(ddPath)
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(code + "\n")

	runErr := cmd.Run()
	out := stdout.String() + stderr.String()
	manifestID = firstMatch(out, regexp.MustCompile(`(?i)manifest[:\s]+(\d+)`))
	steamBuildID = firstMatch(out, regexp.MustCompile(`(?i)build(?:id)?[:\s]+(\d+)`))
	if runErr != nil {
		hint := ""
		if strings.Contains(out, "STEAM GUARD") || strings.Contains(out, "No code was provided") {
			hint = "\nHint: set STEAM_SHARED_SECRET to the mobile authenticator shared_secret (base64)."
		}
		label := depotID
		if branch != "" {
			label = branch + "/" + depotID
		}
		return manifestID, steamBuildID, fmt.Errorf("DepotDownloader %s: %w\n%s%s", label, runErr, truncate(out, 2000), hint)
	}
	return manifestID, steamBuildID, nil
}

func firstMatch(s string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
