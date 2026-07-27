package vcc

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"net/http"
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

func TestParseNewestPrereleaseTag(t *testing.T) {
	body := `[
		{"tag_name":"2.5.0-beta.2","prerelease":true,"assets":[]},
		{"tag_name":"2.5.0-beta.1","prerelease":true,"assets":[]},
		{"tag_name":"2.3.0-beta.3","prerelease":false,"assets":[]}
	]`
	got, err := parseNewestPrereleaseTag([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.5.0-beta.2" {
		t.Fatalf("got %q", got)
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
		if !strings.Contains(req.URL.Host, "api.github.com") {
			t.Fatalf("host %s", req.URL.Host)
		}
		body := `[{"tag_name":"2.5.0-beta.2","prerelease":true,"assets":[]}]`
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

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Host, "api.github.com"):
			body := `[{"tag_name":"2.5.0-beta.2","prerelease":true,"assets":[]}]`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		case strings.Contains(req.URL.Path, "Release_Portable.zip"):
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(zipBytes)),
				Header:     make(http.Header),
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
	if cb.RawMatch != "2.5.0-beta.2" {
		t.Fatalf("rawMatch %q", cb.RawMatch)
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
