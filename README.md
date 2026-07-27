# VRC-API2WIKI

A Go service that keeps [VRChat Wiki](https://wiki.vrchat.com) in sync with live VRChat API data:

1. **Worlds** — populate `Template:World/<id>/…` data pages for infoboxes
2. **Marketplace** — store shelves and paid avatars as `{{InventoryContentDisplay}}` pages plus listing images
3. **Client builds** — Steam depots via shipped [DepotDownloader](https://github.com/SteamRE/DepotDownloader), plus Meta Quest, Pico Store, Google Play, and Viveport listing version metadata

## Quick start

```bash
git clone https://github.com/Hackebein/vrc-api2wiki.git
cd vrc-api2wiki
bash scripts/fetch-depotdownloader.sh

# Local / offline wiki writes (recommended while testing):
unset VRCWIKI_USERNAME VRCWIKI_PASSWORD

export VRCHAT_USERNAME='…'
export VRCHAT_PASSWORD='…'
export VRCHAT_TOTP_SECRET='…'   # base32

export STEAM_USERNAME='…'
export STEAM_PASSWORD='…'
export STEAM_SHARED_SECRET='…'  # base64

go run ./cmd/vrc-api2wiki
```

## Configuration

### Wiki

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VRCWIKI_API_URL` | No | `https://wiki.vrchat.com/api.php` | MediaWiki API endpoint |
| `VRCWIKI_USERNAME` | Yes* | — | Wiki bot username |
| `VRCWIKI_PASSWORD` | Yes* | — | Wiki account password |
| `VRCWIKI_AUTHORIZATION_HEADER` | No | — | Extra HTTP header name (Cloudflare bypass) |
| `VRCWIKI_AUTHORIZATION_VALUE` | No | — | Extra HTTP header value |
| `VRC_API2WIKI_WORLD_IDS` | No | — | Comma-separated world IDs; skips discovery |
| `VRC_API2WIKI_LIMIT` | No | all | Cap worlds **and** paid marketplace avatars to the first *n* each |

\* Omitting both username and password enables [offline mode](#offline-mode).

### VRChat marketplace

| Variable | Required | Description |
|----------|----------|-------------|
| `VRCHAT_USERNAME` | Yes for marketplace | Account email/username |
| `VRCHAT_PASSWORD` | Yes for marketplace | Password |
| `VRCHAT_TOTP_SECRET` | Yes for marketplace | Raw base32 TOTP secret (not an otpauth URI) |

### Client builds (Steam / Quest / Pico / Google Play / Viveport)

| Variable | Required | Description |
|----------|----------|-------------|
| `STEAM_USERNAME` | Yes for steam depots | Steam account that owns VRChat |
| `STEAM_PASSWORD` | Yes for steam depots | Password |
| `STEAM_SHARED_SECRET` | Yes for steam depots | Mobile authenticator `shared_secret` (base64); used to generate Steam Guard codes |

## Offline mode

When wiki credentials are omitted, the tool never logs in and never writes to the wiki:

- **Reads** — Discovery and existing page checks still hit the live wiki API when needed for worlds.
- **Writes** — Page edits and image uploads go to `./wiki-output/` instead.

## License

This project is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE) (AGPL-3.0-or-later).

Bundled DepotDownloader is GPL-2.0; see [third_party/DepotDownloader/NOTICE](third_party/DepotDownloader/NOTICE).
