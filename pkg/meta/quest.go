package meta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

const (
	VRChatQuestAppID       = "1856672347794301"
	QuestAndroidClientName = "android"

	oculusGraphQLURL    = "https://graph.oculus.com/graphql"
	oculusStoreToken    = "OC|752908224809889|"
	oculusVersionsDocID = "1586217024733717"
)

type QuestBinary struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	VersionCode int    `json:"versionCode"`
	ChangeLog   string `json:"changeLog"`
}

func FetchVRChatQuestBinaries(httpClient *http.Client) ([]QuestBinary, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	form := url.Values{}
	form.Set("access_token", oculusStoreToken)
	form.Set("doc_id", oculusVersionsDocID)
	form.Set("variables", fmt.Sprintf(`{"id":%q}`, VRChatQuestAppID))

	req, err := http.NewRequest(http.MethodPost, oculusGraphQLURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "VRC-API2Wiki/dev")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oculus graphql HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed struct {
		Data struct {
			Node struct {
				SupportedBinaries struct {
					Edges []struct {
						Node QuestBinary `json:"node"`
					} `json:"edges"`
				} `json:"supportedBinaries"`
			} `json:"node"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse oculus graphql: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("oculus graphql: %s", parsed.Errors[0].Message)
	}

	out := make([]QuestBinary, 0, len(parsed.Data.Node.SupportedBinaries.Edges))
	for _, e := range parsed.Data.Node.SupportedBinaries.Edges {
		if strings.TrimSpace(e.Node.Version) == "" {
			continue
		}
		out = append(out, e.Node)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("oculus graphql: no VRChat Quest binaries")
	}
	return out, nil
}

func FetchVRChatQuestBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	bins, err := FetchVRChatQuestBinaries(httpClient)
	if err != nil {
		return nil, err
	}
	latest := bins[0]
	cb, err := steam.ExtractBuildFromBytes([]byte(latest.Version), QuestAndroidClientName)
	if err != nil {
		return nil, fmt.Errorf("parse quest version %q: %w", latest.Version, err)
	}
	cb.SteamBuildID = latest.ID
	return cb, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
