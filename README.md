# VRC-API2WIKI

A Go service that updates [VRChat Wiki](https://wiki.vrchat.com) with worlds,
marketplace listings, client builds, and creator tool versions.

## Run locally

Requires Go 1.26.1 or newer, Bash, `curl`, and `unzip`.

```bash
git clone https://github.com/Hackebein/vrc-api2wiki.git
cd vrc-api2wiki
bash scripts/fetch-depotdownloader.sh
```

Export the credentials you need from the table below, for example:

```bash
export VRCHAT_USERNAME='your-username'
export VRCHAT_PASSWORD='your-password'
export VRCHAT_TOTP_SECRET='your-base32-secret'
go run ./cmd/vrc-api2wiki
```

When both wiki credentials are unset, edits and uploads are saved to `./wiki-output/`
instead of published. The service still reads from the live APIs. To publish,
also export `VRCWIKI_USERNAME` and `VRCWIKI_PASSWORD`.

## Credentials

| Variable | Needed for | Value |
|----------|------------|-------|
| `VRCWIKI_USERNAME` | Publishing to the wiki | Wiki bot username |
| `VRCWIKI_PASSWORD` | Publishing to the wiki | Wiki account password |
| `VRCHAT_USERNAME` | Marketplace sync | VRChat email or username |
| `VRCHAT_PASSWORD` | Marketplace sync | VRChat password |
| `VRCHAT_TOTP_SECRET` | Marketplace sync | Raw base32 authenticator secret, not an `otpauth` URI |
| `STEAM_USERNAME` | Steam client builds | Steam account with access to VRChat |
| `STEAM_PASSWORD` | Steam client builds | Steam password |
| `STEAM_SHARED_SECRET` | Steam client builds | Base64 mobile authenticator `shared_secret` |

Optional environment variables:

| Variable | Purpose |
|----------|---------|
| `VRCWIKI_API_URL` | Override the default `https://wiki.vrchat.com/api.php` endpoint |
| `VRCWIKI_AUTHORIZATION_HEADER` / `VRCWIKI_AUTHORIZATION_VALUE` | Add an HTTP header if the wiki requires one |
| `VRC_API2WIKI_WORLD_IDS` | Sync only these comma-separated world IDs |
| `VRC_API2WIKI_LIMIT` | Limit the number of worlds and paid marketplace avatars synced |

## GitHub Actions

Add the credentials above as repository Actions secrets. Add the optional wiki
endpoint or HTTP header as secrets too, if needed. Enable the **sync** workflow;
it runs hourly and can also be started manually without changing any inputs.

Steam sessions are encrypted and reused automatically. The first run, or an
expired or revoked session, may need a new authenticator login. If you change
`STEAM_SHARED_SECRET`, clear the Steam session caches before running again.

## License

This project is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE) (AGPL-3.0-or-later).

Bundled DepotDownloader is GPL-2.0; see [third_party/DepotDownloader/NOTICE](third_party/DepotDownloader/NOTICE).
