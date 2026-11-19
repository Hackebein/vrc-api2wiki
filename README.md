# VRC-API2WIKI

A Go service that keeps [VRChat Wiki](https://wiki.vrchat.com) in sync with live VRChat API data:

1. **Worlds** — populate `Template:World/<id>/…` data pages for infoboxes. `publicationDate` is written only when that subpage is missing, then left for wiki editors.
2. **Marketplace** — store shelves and paid avatars as `{{InventoryContentDisplay}}` pages plus listing images
3. **Client builds** — Steam depots via shipped [DepotDownloader](https://github.com/SteamRE/DepotDownloader), plus Meta Quest, Pico Store, Google Play, Viveport and the Apple App Store. The main release comes from VRChat's published release notes.
4. **Creator tools** — VRChat Creator Companion (stable + beta) and VRC Quick Launcher versions for `Template:ClientBuild/…`

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

### Client builds

| Variable | Required | Description |
|----------|----------|-------------|
| `STEAM_USERNAME` | Yes for steam depots | Steam account that owns VRChat |
| `STEAM_PASSWORD` | Yes for steam depots | Password |
| `STEAM_SHARED_SECRET` | Yes for steam depots | Mobile authenticator `shared_secret` (base64); used to generate Steam Guard codes |

In GitHub Actions, the existing `STEAM_SHARED_SECRET` also encrypts the saved Steam session, which is reused between runs. Clear the Steam session caches if this secret changes.

| Wiki data template | Source and usage |
|----------|-------------|
| `{{ClientBuild/ios-app-store\|version}}` | Newest version from the US and German Apple listings for app 6448578770. Both listings must be available. |
| `{{ClientBuild/ios-app-store\|buildNumber}}` | VRChat build number from release notes for that exact store version. Empty when VRChat has no matching published build number. |
| `{{ClientBuild/ios-app-store\|releaseNotes}}` | Matching VRChat release notes, or the listing URL if there are no matching notes. Apple's public lookup provides the store version. Apple's internal bundle build number is unavailable from this lookup. |
| `{{ClientBuild/live-mainpage\|version}}` | Original live release from [VRChat's latest release link](https://docs.vrchat.com/docs/latest-release). A patch link resolves to the original release in that family. |
| `{{ClientBuild/live-mainpage\|buildNumber}}`, `\|releaseDate`, `\|releaseNotes` | Original release build, ISO date and link from those same notes. The bot writes all fields in one template revision. |
| `{{ClientBuild/windows/unityVersion}}`, `{{ClientBuild/open-beta-windows/unityVersion}}` | Unity engine version from the `VRChat_Data/globalgamemanagers` header in each Windows Steam depot (app 438100, depot 438101). |

The sync records the parsed values in `steam-output/*.json`. To verify a deployment, check the sync log and read back the four generated Wiki pages above before updating the Wiki's `Version` family and main-page card. Wiki presentation changes are made on the Wiki. Its Git mirror imports edits from the site.

### Creator tools

| Variable | Required | Description |
|----------|----------|-------------|
| `VRC_API2WIKI_GITHUB_TOKEN` | No | Read-only GitHub token for the Quick Launcher release lookup |

The sync workflow uses its built-in GitHub token with `contents: read`. Local runs use GitHub's anonymous quota when the variable is unset or blank.

## Offline mode

When wiki credentials are omitted, the tool never logs in and never writes to the wiki:

- **Reads** — Discovery and existing page checks still hit the live wiki API when needed for worlds.
- **Writes** — Page edits and image uploads go to `./wiki-output/` instead.

## License

This project is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE) (AGPL-3.0-or-later).

Bundled DepotDownloader is GPL-2.0; see [third_party/DepotDownloader/NOTICE](third_party/DepotDownloader/NOTICE).
