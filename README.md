# VRC-API2WIKI

A Go service that keeps [VRChat Wiki](https://wiki.vrchat.com) world infoboxes in sync with live metadata from the public VRChat API. When a world page uses `Template:Infobox/World` or `Template:Infobox/Official World` with an `id` parameter, this tool populates the backing data subpages so the infobox renders current API values automatically.

## How it works

Each sync run follows three steps:

1. **Discover world IDs** — Query the MediaWiki API for pages transcluding `Template:Infobox/World` or `Template:Infobox/Official World`, then parse wikitext for world IDs from the `id=wrld_...` parameter or from `link=` (`{{World link|wrld_...|Name}}` / `{{VRC link|https://vrchat.com/home/world/wrld_...|Name}}`). For each valid ID, keep the marker page `Template:World/<id>` in sync with the id-only infobox call(s) used during discovery.
2. **Fetch VRChat API data** — For each discovered ID, call `GET https://api.vrchat.cloud/api/1/worlds/<id>`.
3. **Write world data** — Create or update `Template:World/<id>/<property>` pages. Nested JSON objects become nested page paths (e.g. `unityPackages/standalonewindows/created_at`). Scalar arrays (e.g. `tags`) are stored comma-separated on a single page.

## Quick start

```bash
git clone https://github.com/Hackebein/vrc-api2wiki.git
cd vrc-api2wiki

export VRCWIKI_USERNAME='your-bot'
export VRCWIKI_PASSWORD='your-password'
export VRCWIKI_AUTHORIZATION_HEADER='...'
export VRCWIKI_AUTHORIZATION_VALUE='...'

go run ./cmd/vrc-api2wiki
```

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VRCWIKI_API_URL` | No | `https://wiki.vrchat.com/api.php` | MediaWiki API endpoint |
| `VRCWIKI_USERNAME` | Yes* | — | Wiki bot or service account username |
| `VRCWIKI_PASSWORD` | Yes* | — | Wiki account password |
| `VRCWIKI_AUTHORIZATION_HEADER` | No | — | Extra HTTP header name (Cloudflare bypass) |
| `VRCWIKI_AUTHORIZATION_VALUE` | No | — | Extra HTTP header value (Cloudflare bypass) |
| `VRC_API2WIKI_WORLD_IDS` | No | — | Comma-separated world IDs; skips wiki discovery |
| `VRC_API2WIKI_LIMIT` | No | all | Positive integer: sync only the first *n* discovered IDs. |

\* Omitting both `VRCWIKI_USERNAME` and `VRCWIKI_PASSWORD` enables [offline mode](#offline-mode).

## Offline mode

When credentials are omitted, the tool never logs in and never writes to the wiki:

- **Reads** — Discovery, existing page wikitext, and file SHA1s are fetched from the live wiki API.
- **Writes** — Page edits and image uploads are written to `./wiki-output/` instead. Logs indicate what would have happened (`would create page`, `would edit page (content changed)`, `skip page (unchanged on wiki)`, etc.).

## Development

```bash
go build -o /dev/null ./...
go vet ./...
go test ./...
```

## Docker

```bash
docker build -t vrc-api2wiki .
docker run --rm \
  -e VRCWIKI_USERNAME -e VRCWIKI_PASSWORD \
  -e VRCWIKI_AUTHORIZATION_HEADER -e VRCWIKI_AUTHORIZATION_VALUE \
  vrc-api2wiki
```
