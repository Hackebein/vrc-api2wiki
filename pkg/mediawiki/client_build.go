package mediawiki

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

// publishClientBuild stores all fields in one Wiki revision. An interrupted
// sync can leave an older build visible, but its fields still agree.
func publishClientBuild(wiki *MediaWikiClient, root string, build *steam.ClientBuild, logger *slog.Logger) error {
	title := "Template:ClientBuild/" + build.Branch
	text, err := renderClientBuild(build)
	if err != nil {
		return fmt.Errorf("render %s: %w", title, err)
	}
	jsonPath := filepath.Join(root, "steam-output", build.Branch+".json")
	if err := steam.WriteBuildJSON(jsonPath, build); err != nil {
		return err
	}
	if err := wiki.EditPage(title, text, true); err != nil {
		return fmt.Errorf("edit %s: %w", title, err)
	}
	if logger != nil {
		logger.Info("client build written", "client", build.Branch, "version", build.Version,
			"buildNumber", build.BuildNumber, "buildHash", build.BuildHash,
			"unityVersion", build.UnityVersion, "releaseDate", build.ReleaseDate,
			"source", build.SourceURL, "releaseNotes", build.ReleaseNotesURL, "json", jsonPath)
	}
	return nil
}

func renderClientBuild(build *steam.ClientBuild) (string, error) {
	display := build.Version
	if display != "" && build.BuildNumber != "" {
		display += " (" + build.BuildNumber + ")"
	}
	fields := []struct {
		key   string
		value string
	}{
		{"version", build.Version},
		{"buildNumber", build.BuildNumber},
		{"buildHash", build.BuildHash},
		{"unityVersion", build.UnityVersion},
		{"releaseDate", build.ReleaseDate},
		{"releaseNotes", build.ReleaseNotesURL},
		{"display", display},
	}
	var text strings.Builder
	text.WriteString("{{#switch:{{{1|display}}}\n")
	for _, field := range fields {
		if strings.ContainsAny(field.value, "|{}[]<>\r\n") {
			return "", fmt.Errorf("%s contains wikitext syntax", field.key)
		}
		fmt.Fprintf(&text, " |%s = %s\n", field.key, field.value)
	}
	text.WriteString(" |#default =\n}}")
	return text.String(), nil
}
