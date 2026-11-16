package vcc

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestParseStableVersionSkipsBeta(t *testing.T) {
	html := `
		<a href="/news/release-2.5.0">Release 2.5.0-beta.2</a>
		<a href="/news/release-2.4.5">Release 2.4.5</a>
		<a href="/news/release-2.4.4">Release 2.4.4</a>
	`
	got, err := parseStableVersion([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.4.5" {
		t.Fatalf("got %q", got)
	}
}

func TestParsePrereleaseVersion(t *testing.T) {
	html := `
		<a href="/news/release-2.4.5">Release 2.4.5</a>
		<a href="/news/release-2.5.0">Release 2.5.0-beta.2</a>
		<a href="/news/release-2.5.0">Release 2.5.0-beta.1</a>
	`
	got, err := parsePrereleaseVersion([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.5.0-beta.2" {
		t.Fatalf("got %q", got)
	}
}

func TestParsePrereleaseVersionMissing(t *testing.T) {
	html := `
		<a href="/news/release-2.4.5">Release 2.4.5</a>
		<a href="/news/release-2.4.4">Release 2.4.4</a>
	`
	_, err := parsePrereleaseVersion([]byte(html))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no prerelease Release version found") {
		t.Fatalf("got %v", err)
	}
}

func TestParsePortableZipReleaseConstructsURL(t *testing.T) {
	body := `[
		{"tag_name":"2.5.0-beta.2","prerelease":true,"assets":[]}
	]`
	tag, zipURL, err := parsePortableZipRelease([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if tag != "2.5.0-beta.2" {
		t.Fatalf("tag %q", tag)
	}
	if zipURL != "" {
		t.Fatalf("expected empty zipURL, got %q", zipURL)
	}
}

func TestParsePortableZipReleaseUsesPrereleaseTagName(t *testing.T) {
	body := `[
		{"tag_name":"2.5.0-beta.2","prerelease":false,"assets":[]}
	]`
	tag, zipURL, err := parsePortableZipRelease([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if tag != "2.5.0-beta.2" {
		t.Fatalf("tag %q", tag)
	}
	if zipURL != "" {
		t.Fatalf("expected empty zipURL, got %q", zipURL)
	}
}

func TestParsePortableZipReleaseUsesAssetURL(t *testing.T) {
	body := `[
		{
			"tag_name":"2.5.0-beta.2",
			"prerelease":true,
			"assets":[{
				"name":"web_vcc_2.5.0-beta.2_Release_Portable.zip",
				"browser_download_url":"https://example.test/portable.zip"
			}]
		}
	]`
	tag, zipURL, err := parsePortableZipRelease([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if tag != "2.5.0-beta.2" || zipURL != "https://example.test/portable.zip" {
		t.Fatalf("tag=%q url=%q", tag, zipURL)
	}
}

func TestProductVersionFromPE(t *testing.T) {
	data := fakePEWithProductVersion("1.6.2")
	got, err := productVersionFromPE(data)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.6.2" {
		t.Fatalf("got %q", got)
	}
}

func TestQuickLauncherVersionFromZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(quickLauncherZipPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(fakePEWithProductVersion("1.6.2")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := quickLauncherVersionFromZip(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.6.2" {
		t.Fatalf("got %q", got)
	}
}

func TestFetchCreatorCompanion(t *testing.T) {
	t.Setenv(githubTokenEnv, "test-token")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "vcc.docs.vrchat.com" {
			t.Fatalf("host %s", req.URL.Host)
		}
		if req.Header.Get("Authorization") != "" {
			t.Fatal("VCC docs request carried an Authorization header")
		}
		body := `<a href="/news/release-2.5.0">Release 2.5.0-beta.2</a><a href="/news/release-2.4.5">Release 2.4.5</a>`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	cb, err := FetchCreatorCompanion(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2.4.5" || cb.Branch != CreatorCompanionClientName {
		t.Fatalf("%+v", cb)
	}
}

func TestFetchCreatorCompanionBeta(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "vcc.docs.vrchat.com" {
			t.Fatalf("host %s", req.URL.Host)
		}
		body := `<a href="/news/release-2.5.0">Release 2.5.0-beta.2</a><a href="/news/release-2.4.5">Release 2.4.5</a>`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	cb, err := FetchCreatorCompanionBeta(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2.5.0-beta.2" || cb.Branch != CreatorCompanionBetaClientName {
		t.Fatalf("%+v", cb)
	}
}

func TestFetchCreatorCompanionBetaMissing(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `<a href="/news/release-2.4.5">Release 2.4.5</a>`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	_, err := FetchCreatorCompanionBeta(client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no prerelease Release version found") {
		t.Fatalf("got %v", err)
	}
}

func TestFetchQuickLauncher(t *testing.T) {
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	w, err := zw.Create(quickLauncherZipPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(fakePEWithProductVersion("1.6.2")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zipBytes := zipBuf.Bytes()

	const tag = "2.5.0-beta.2"
	const redirectedZipURL = "https://release-assets.githubusercontent.com/quick-launcher.zip"
	zipURL := fmt.Sprintf(portableZipURL, tag, tag)
	for _, tt := range []struct {
		name          string
		token         string
		authorization string
	}{
		{name: "unset token"},
		{name: "configured token", token: " test-token ", authorization: "Bearer test-token"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(githubTokenEnv, tt.token)
			if tt.token == "" {
				if err := os.Unsetenv(githubTokenEnv); err != nil {
					t.Fatal(err)
				}
			}
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				authorization := req.Header.Get("Authorization")
				switch url := req.URL.String(); url {
				case githubReleasesURL:
					if authorization != tt.authorization {
						t.Fatal("unexpected Authorization header on release lookup")
					}
					if req.Header.Get("Accept") != "application/vnd.github+json" {
						t.Fatalf("Accept %q", req.Header.Get("Accept"))
					}
					body := `[{"tag_name":"2.5.0-beta.2","prerelease":false,"assets":[]}]`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
					}, nil
				case zipURL, redirectedZipURL:
					if authorization != "" {
						t.Fatal("portable ZIP request carried an Authorization header")
					}
					if url == zipURL {
						return &http.Response{
							StatusCode: http.StatusFound,
							Header:     http.Header{"Location": {redirectedZipURL}},
							Body:       http.NoBody,
						}, nil
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(zipBytes)),
					}, nil
				default:
					t.Fatalf("unexpected URL %s", req.URL)
					return nil, nil
				}
			})}
			cb, err := FetchQuickLauncher(client)
			if err != nil {
				t.Fatal(err)
			}
			if cb.Version != "1.6.2" || cb.Branch != QuickLauncherClientName {
				t.Fatalf("%+v", cb)
			}
			if cb.RawMatch != tag {
				t.Fatalf("rawMatch %q", cb.RawMatch)
			}
		})
	}
}

func TestFetchQuickLauncherRejectsAuthenticatedRedirect(t *testing.T) {
	t.Setenv(githubTokenEnv, "test-token")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != githubReleasesURL {
			t.Fatalf("followed release-list redirect to %s", req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": {"https://api.github.com/"}},
			Body:       http.NoBody,
		}, nil
	})}
	if _, err := FetchQuickLauncher(client); err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("expected HTTP 302 error, got %v", err)
	}
}

func fakePEWithProductVersion(version string) []byte {
	var b bytes.Buffer
	b.WriteString("MZ")
	b.Write(bytes.Repeat([]byte{0}, 64))
	writeUTF16LE(&b, "ProductVersion")
	b.Write([]byte{0, 0})
	writeUTF16LE(&b, version)
	b.Write([]byte{0, 0})
	return b.Bytes()
}

func writeUTF16LE(b *bytes.Buffer, s string) {
	for _, u := range utf16.Encode([]rune(s)) {
		_ = binary.Write(b, binary.LittleEndian, u)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
