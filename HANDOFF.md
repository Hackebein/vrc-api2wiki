# Marketplace sync handoff

## 2026-11-25 · Current Offers shelf links

`Template:VRChatStoreCurrentOffers` revision `169932` transcludes `Winter's Blessing` and `Magician's Bundle` with apostrophes. The marketplace sync published the shelves as `Winter s Blessing` and `Magician s Bundle`. The [scheduled sync log](https://github.com/Hackebein/vrc-api2wiki/actions/runs/45710520171/job/139547355270) shows both shelves under their published titles.

Shelf writes and Current Offers now share the same title normalization so the transclusions point to the existing pages. The shelf index reads the display heading back from the listing page when it resolves to the same target. This preserves the punctuation when a later sync matches a renamed shelf by product IDs.

### Verification

- The regression test reproduced both broken references on `main` and passes with the fix. It exercises the index read on a subsequent sync. Run `go test ./pkg/mediawiki -run '^TestCurrentOffersLinksToPublishedShelfPages$' -count=1` from the repository root. Go 1.26.1 or later is required. The test uses a local MediaWiki stub.
- `go test ./... -count=1`, `go vet ./...`, and `go build ./...` passed.
- MediaWiki `action=query` confirmed that all six preview targets exist. A read-only `action=parse` preview of revision `169932` with the corrected references rendered both shelves with zero red links. [Current Offers source revision](https://wiki.vrchat.com/index.php?title=Template:VRChatStoreCurrentOffers&oldid=169932) · [Winter listing revision](https://wiki.vrchat.com/index.php?oldid=169889) · [Magician listing revision](https://wiki.vrchat.com/index.php?oldid=169931)

### Pending

After merge, verify that the next scheduled sync updates [Current Offers](https://wiki.vrchat.com/wiki/Template:VRChatStoreCurrentOffers) and both shelves render there. The template is rebuilt even when the shelf cache skips both child pages.
