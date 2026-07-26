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
	VRChatQuestAppID           = "1856672347794301"
	QuestClientName            = "android-quest"
	QuestOpenBetaClientName    = "open-beta-android-quest"
	oculusGraphQLURL           = "https://graph.oculus.com/graphql"
	oculusStoreToken           = "OC|752908224809889|"
	oculusVersionsDocID        = "1586217024733717"
	oculusReleaseChannelsDocID = "3828663700542720"
	oculusReleaseChannelDocID  = "3973666182694273"
)

type QuestBinary struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	VersionCode int    `json:"versionCode"`
	ChangeLog   string `json:"changeLog"`
}

type questReleaseChannel struct {
	ID                    string
	Name                  string
	LatestSupportedBinary *QuestBinary
}

func FetchVRChatQuestBinaries(httpClient *http.Client) ([]QuestBinary, error) {
	body, err := oculusGraphQL(httpClient, oculusVersionsDocID, map[string]any{"id": VRChatQuestAppID})
	if err != nil {
		return nil, err
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

func FetchVRChatQuestReleaseChannels(httpClient *http.Client) ([]questReleaseChannel, error) {
	body, err := oculusGraphQL(httpClient, oculusReleaseChannelsDocID, map[string]any{"applicationID": VRChatQuestAppID})
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data struct {
			Node struct {
				ReleaseChannels struct {
					Nodes []struct {
						ID                    string `json:"id"`
						ChannelName           string `json:"channel_name"`
						LatestSupportedBinary *struct {
							ID          string `json:"id"`
							Version     string `json:"version"`
							VersionCode int    `json:"version_code"`
						} `json:"latest_supported_binary"`
					} `json:"nodes"`
				} `json:"release_channels"`
			} `json:"node"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse oculus release channels: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("oculus graphql: %s", parsed.Errors[0].Message)
	}
	out := make([]questReleaseChannel, 0, len(parsed.Data.Node.ReleaseChannels.Nodes))
	for _, n := range parsed.Data.Node.ReleaseChannels.Nodes {
		ch := questReleaseChannel{ID: n.ID, Name: n.ChannelName}
		if n.LatestSupportedBinary != nil && strings.TrimSpace(n.LatestSupportedBinary.Version) != "" {
			ch.LatestSupportedBinary = &QuestBinary{
				ID:          n.LatestSupportedBinary.ID,
				Version:     n.LatestSupportedBinary.Version,
				VersionCode: n.LatestSupportedBinary.VersionCode,
			}
		}
		out = append(out, ch)
	}
	return out, nil
}

func FetchVRChatQuestPrimaryBinaries(httpClient *http.Client, releaseChannelID string) ([]QuestBinary, error) {
	if strings.TrimSpace(releaseChannelID) == "" {
		return nil, fmt.Errorf("oculus graphql: empty release channel id")
	}
	body, err := oculusGraphQL(httpClient, oculusReleaseChannelDocID, map[string]any{"releaseChannelID": releaseChannelID})
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data struct {
			Node struct {
				Application struct {
					PrimaryBinaries struct {
						Edges []struct {
							Node struct {
								ID          string `json:"id"`
								Version     string `json:"version"`
								VersionCode int    `json:"version_code"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"primary_binaries"`
				} `json:"application"`
			} `json:"node"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse oculus release channel: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("oculus graphql: %s", parsed.Errors[0].Message)
	}
	out := make([]QuestBinary, 0, len(parsed.Data.Node.Application.PrimaryBinaries.Edges))
	for _, e := range parsed.Data.Node.Application.PrimaryBinaries.Edges {
		if strings.TrimSpace(e.Node.Version) == "" {
			continue
		}
		out = append(out, QuestBinary{
			ID:          e.Node.ID,
			Version:     e.Node.Version,
			VersionCode: e.Node.VersionCode,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("oculus graphql: no primary binaries for channel %s", releaseChannelID)
	}
	return out, nil
}

func FetchVRChatQuestBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	bins, err := FetchVRChatQuestBinaries(httpClient)
	if err != nil {
		return nil, err
	}
	return questBinaryToClientBuild(bins[0], QuestClientName)
}

func FetchVRChatQuestOpenBetaBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	channels, err := FetchVRChatQuestReleaseChannels(httpClient)
	if err != nil {
		return nil, err
	}
	var live *questReleaseChannel
	for i := range channels {
		ch := &channels[i]
		if isOpenBetaChannelName(ch.Name) && ch.LatestSupportedBinary != nil {
			return questBinaryToClientBuild(*ch.LatestSupportedBinary, QuestOpenBetaClientName)
		}
		if strings.EqualFold(ch.Name, "LIVE") {
			live = ch
		}
	}
	if live == nil || live.ID == "" {
		return nil, fmt.Errorf("oculus graphql: LIVE release channel not found")
	}
	if live.LatestSupportedBinary == nil {
		return nil, fmt.Errorf("oculus graphql: LIVE channel has no latest binary")
	}
	primaries, err := FetchVRChatQuestPrimaryBinaries(httpClient, live.ID)
	if err != nil {
		return nil, err
	}
	newest := primaries[0]
	if newest.VersionCode > live.LatestSupportedBinary.VersionCode {
		return questBinaryToClientBuild(newest, QuestOpenBetaClientName)
	}
	return questBinaryToClientBuild(*live.LatestSupportedBinary, QuestOpenBetaClientName)
}

func isOpenBetaChannelName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(n, "open beta") || n == "beta" || n == "open-beta" || n == "open_beta"
}

func questBinaryToClientBuild(bin QuestBinary, clientName string) (*steam.ClientBuild, error) {
	cb, err := steam.ExtractBuildFromBytes([]byte(bin.Version), clientName)
	if err != nil {
		return nil, fmt.Errorf("parse quest version %q: %w", bin.Version, err)
	}
	cb.SteamBuildID = bin.ID
	return cb, nil
}

func oculusGraphQL(httpClient *http.Client, docID string, variables map[string]any) ([]byte, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	vars, err := json.Marshal(variables)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("access_token", oculusStoreToken)
	form.Set("doc_id", docID)
	form.Set("variables", string(vars))

	req, err := http.NewRequest(http.MethodPost, oculusGraphQLURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "VRC-API2Wiki/dev")
	req.Header.Set("TE", "Trailers")

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
	return body, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
