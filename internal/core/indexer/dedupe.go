package indexer

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// dedupeByInfoHash collapses search results that point at the same torrent
// (by info_hash) into a single row whose Seeds/Peers are the median across
// the contributing indexers. Without this, a single indexer that publishes
// an inflated count (the canonical case: TorrentGalaxyClone reporting 159
// while four other indexers covering the same release report 50–70) ranks
// the inflated row first and crowds the search dialog with duplicates.
//
// Releases whose info_hash can't be determined (HTTP .torrent URLs whose
// hash is only inside the file body, not the URL) pass through ungrouped.
//
// Inputs are expected to already be sorted by quality+seeds — the FIRST
// occurrence of each info_hash is kept as the "winning" representative,
// preserving that prior sort decision.
func dedupeByInfoHash(in []SearchResult) []SearchResult {
	if len(in) < 2 {
		return in
	}

	type group struct {
		keepIdx int
		seeds   []int
		peers   []int
		sources []string
	}

	groups := make(map[string]*group, len(in))
	keep := make([]bool, len(in))

	for i, r := range in {
		key := dedupeKey(r)
		if key == "" {
			// Ungroupable — always keep as-is.
			keep[i] = true
			continue
		}
		g, ok := groups[key]
		if !ok {
			groups[key] = &group{
				keepIdx: i,
				seeds:   []int{r.Seeds},
				peers:   []int{r.Peers},
				sources: nil,
			}
			keep[i] = true
			continue
		}
		// Duplicate of an earlier (higher-ranked) entry. Fold its data in.
		g.seeds = append(g.seeds, r.Seeds)
		g.peers = append(g.peers, r.Peers)
		if r.Indexer != "" && r.Indexer != in[g.keepIdx].Indexer {
			g.sources = append(g.sources, r.Indexer)
		}
	}

	// Apply medians + OtherSources to the kept rows.
	for _, g := range groups {
		if len(g.seeds) > 1 {
			in[g.keepIdx].Seeds = medianInt(g.seeds)
			in[g.keepIdx].Peers = medianInt(g.peers)
			in[g.keepIdx].OtherSources = g.sources
		}
	}

	out := in[:0]
	for i, r := range in {
		if keep[i] {
			out = append(out, r)
		}
	}

	// Re-sort: the median may have moved a row's effective seed count.
	sort.Slice(out, func(i, j int) bool {
		si, sj := out[i].QualityScore, out[j].QualityScore
		if si != sj {
			return si > sj
		}
		return out[i].Seeds > out[j].Seeds
	})
	return out
}

// dedupeKey returns a string that's stable across indexers reporting the
// same release.
//
// Primary key: (Size, normalized title). Reasoning:
//
//  1. The actual data Prism receives via Pulse's torznab proxy is
//     ALWAYS HTTP-redirect URLs — the upstream magnet/info_hash is
//     hidden behind the proxy. Pulse's Cardigann scrapers don't
//     surface an `infohash` torznab attr because deriving it would
//     require an extra detail-page fetch per result.
//  2. Different indexers reporting the same physical release agree on
//     byte size to the byte (it's encoded in the torrent). They differ
//     on title formatting (`.` vs ` ` vs `-`), which `normalizeTitle`
//     erases.
//  3. (Size, normalized-title) is strong enough in practice: two
//     physically-different releases with the same exact byte count AND
//     the same words in the title are vanishingly rare. Title-only is
//     too lossy; size-only is too coarse; both together threads the
//     needle.
//
// Fallback: r.InfoHash. Only used when size or title aren't usable
// (extremely rare — every torznab response carries size). This keeps
// us correct on the edge case where a custom plugin populates
// InfoHash but not size, without forcing a separate code path.
//
// Returns "" when nothing reliable is derivable — that row stays
// ungrouped in the output, which is the safe default (no false-positive
// merges).
func dedupeKey(r SearchResult) string {
	if r.Size > 0 && r.Title != "" {
		nt := normalizeTitle(r.Title)
		if nt != "" {
			return "st:" + strconv.FormatInt(r.Size, 10) + "|" + nt
		}
	}
	if h := r.InfoHash; h != "" {
		return "ih:" + strings.ToLower(strings.TrimSpace(h))
	}
	if h := extractInfoHash(r.DownloadURL); h != "" {
		return "ih:" + h
	}
	return ""
}

// normalizeTitle collapses release-title formatting differences so that
// "John Wick 2014 PROPER UHD BluRay 2160p ... REMUX-FraMeSToR" and
// "John.Wick.2014.PROPER.UHD.BluRay.2160p...REMUX.FraMeSToR" hash to the
// same string.
//
//   - lowercase
//   - replace .  _  -  +  (  )  [  ]  with a space
//   - collapse runs of whitespace
//   - trim
//
// Critically does NOT strip release group, language, year, or quality
// markers — those are signal that distinguishes physically-different
// releases. The intent is to neutralize separator style, not to reduce
// titles to a fuzzy fingerprint.
var titleSeparatorRe = regexp.MustCompile(`[._\-+()\[\]]+`)

func normalizeTitle(t string) string {
	t = strings.ToLower(t)
	t = titleSeparatorRe.ReplaceAllString(t, " ")
	t = strings.Join(strings.Fields(t), " ")
	return t
}

// extractInfoHash pulls the BitTorrent info hash from a magnet URI or any
// URL whose query carries `xt=urn:btih:<hash>`. Returns the lowercased
// 40-char hex form, or "" when the hash can't be determined (HTTP
// .torrent URLs, malformed magnets, etc).
//
// Supports:
//   - magnet:?xt=urn:btih:<HEX40>...
//   - magnet:?xt=urn:btih:<BASE32_32>...   (legacy form, rare)
//   - any URL whose query has the same xt=urn:btih:... pattern
//   - the literal data: scheme used by the Pulse plugin pipeline carries
//     base64 .torrent bytes, NOT a hash — those return "" (kept ungrouped).
func extractInfoHash(downloadURL string) string {
	if downloadURL == "" {
		return ""
	}
	// magnet: parse the query directly. url.Parse handles magnet: but
	// puts everything in RawQuery — Query() works.
	u, err := url.Parse(downloadURL)
	if err != nil {
		return ""
	}
	for _, xt := range u.Query()["xt"] {
		const prefix = "urn:btih:"
		if !strings.HasPrefix(xt, prefix) {
			continue
		}
		hash := strings.TrimPrefix(xt, prefix)
		hash = strings.ToLower(hash)
		// 40-char hex is the v1 form. 32-char base32 is legacy. v2 hashes
		// (urn:btmh:) are intentionally ignored — they're a different key.
		switch len(hash) {
		case 40:
			if isHex(hash) {
				return hash
			}
		case 32:
			// Base32 form — accept as-is for grouping purposes; we don't
			// need to convert because two indexers reporting the same
			// torrent in base32 will produce the same string anyway.
			return hash
		}
	}
	return ""
}

func isHex(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

// medianInt returns the median of a non-empty slice of ints. For
// even-length inputs we pick the LOWER of the two middle values — the
// more conservative choice, since we'd rather under-promise seed counts
// than over-promise them. Mutates the input (sorts it in place); pass a
// copy if the caller cares.
func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sort.Ints(values)
	return values[(len(values)-1)/2]
}
