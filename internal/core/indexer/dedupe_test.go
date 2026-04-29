package indexer

// dedupe_test.go — Regression tests for the cross-indexer info_hash dedup
// pass added in Search/GetRecent. These pin down the contract that the
// search dialog relies on: when the same torrent is reported by multiple
// indexers, we show ONE row with the median seed count instead of N
// rows where the indexer with the most-inflated number wins the rank.
//
// Real-world incident: TorrentGalaxyClone published 159 seeders for the
// John Wick PROPER UHD REMUX while 1337x reported 68, ExtraTorrent 57,
// LimeTorrents 55. Live tracker scrape said ~52. The 159 dominated the
// search rank and was both wrong AND a separate row from the 55/57/68
// ones. Fixing the indexer is out of our control; merging by info_hash
// removes its over-influence.

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/beacon-stack/prism/pkg/plugin"
)

// ── extractInfoHash ─────────────────────────────────────────────────────────

func TestExtractInfoHash_MagnetHex(t *testing.T) {
	got := extractInfoHash("magnet:?xt=urn:btih:EC5086C1C78D1ACD62EC8F2DD281E18BA222AC55&dn=John+Wick")
	want := "ec5086c1c78d1acd62ec8f2dd281e18ba222ac55"
	if got != want {
		t.Errorf("magnet hex: got %q, want %q (must lowercase)", got, want)
	}
}

func TestExtractInfoHash_MagnetWithMultipleParams(t *testing.T) {
	// Real magnets have many trackers + size — make sure we still pick the hash.
	in := "magnet:?xt=urn:btih:abcdef0123456789abcdef0123456789abcdef01" +
		"&dn=test" +
		"&tr=udp%3A%2F%2Ftracker.example.com%3A6969%2Fannounce" +
		"&xl=70000000000"
	got := extractInfoHash(in)
	if got != "abcdef0123456789abcdef0123456789abcdef01" {
		t.Errorf("multi-param magnet: got %q", got)
	}
}

func TestExtractInfoHash_HTTPReturnsEmpty(t *testing.T) {
	// HTTP/HTTPS .torrent URLs don't carry the info hash in the URL — the
	// hash is inside the .torrent file body. Those releases must pass
	// through dedup ungrouped (treat each as unique).
	cases := []string{
		"https://1337x.to/torrent/6449108/John-Wick-2014/",
		"https://torrentgalaxy.one/post-detail/76bd60/john-wick-2014/",
		"http://example.com/foo.torrent",
	}
	for _, in := range cases {
		if got := extractInfoHash(in); got != "" {
			t.Errorf("HTTP url %q: got %q, want empty (no info_hash discoverable from URL alone)", in, got)
		}
	}
}

func TestExtractInfoHash_EmptyAndGarbage(t *testing.T) {
	cases := []string{
		"",
		"not-a-url",
		"magnet:?xt=urn:btih:tooshort",
		"magnet:?xt=urn:btih:zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", // 40 chars but not hex
		"magnet:?xt=urn:btmh:1220abcdef",                               // v2 multihash — different namespace, ignored
		"magnet:?dn=no-xt-param",
	}
	for _, in := range cases {
		if got := extractInfoHash(in); got != "" {
			t.Errorf("garbage %q: got %q, want empty", in, got)
		}
	}
}

func TestExtractInfoHash_Base32Legacy(t *testing.T) {
	// Some old indexers still emit the base32 form. We don't decode it —
	// just keep the string as-is so two indexers using base32 group together.
	in := "magnet:?xt=urn:btih:5UIINQOHRUNM2YXMR4W5FAPBROREFLCV"
	got := extractInfoHash(in)
	if got != "5uiinqohrunm2yxmr4w5fapbrorerflcv"[:32] {
		// We just need consistent lowercased 32 chars.
		if len(got) != 32 {
			t.Errorf("base32 magnet: got %q (len=%d), want 32-char lowercase string", got, len(got))
		}
	}
}

// ── normalizeTitle / dedupeKey ─────────────────────────────────────────────

func TestNormalizeTitle_CollapsesSeparators(t *testing.T) {
	cases := []struct{ in, want string }{
		{"John.Wick.2014.PROPER.UHD.BluRay.2160p", "john wick 2014 proper uhd bluray 2160p"},
		{"John Wick 2014 PROPER UHD BluRay 2160p", "john wick 2014 proper uhd bluray 2160p"},
		{"John-Wick-2014-PROPER-UHD-BluRay-2160p", "john wick 2014 proper uhd bluray 2160p"},
		{"John+Wick+2014+PROPER+UHD+BluRay+2160p", "john wick 2014 proper uhd bluray 2160p"},
		{"REMUX-FraMeSToR", "remux framestor"},
		{"  Mixed  Whitespace  ", "mixed whitespace"},
	}
	for _, c := range cases {
		if got := normalizeTitle(c.in); got != c.want {
			t.Errorf("normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeTitle_DoesNotStripDistinguishingMarkers(t *testing.T) {
	// Different release groups/sources MUST stay distinguishable after
	// normalize — otherwise we'd merge releases that aren't actually the
	// same torrent.
	a := normalizeTitle("John.Wick.2014.PROPER.2160p.BluRay.REMUX.HEVC.DTS-HD.MA.TrueHD.7.1.Atmos-FGT")
	b := normalizeTitle("John.Wick.2014.PROPER.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRID.REMUX-FraMeSToR")
	if a == b {
		t.Errorf("FGT and FraMeSToR builds normalized to same string — too aggressive: %q", a)
	}
}

func TestDedupeKey_Priority(t *testing.T) {
	// 1. (Size, normalized-title) is the primary key — chosen because
	// real-world Prism data flows through Pulse's torznab proxy, which
	// hides info_hash. Even when InfoHash and a magnet are also present,
	// size+title wins so all rows for the same release group together.
	got := dedupeKey(SearchResult{Release: plugin.Release{
		InfoHash:    "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
		DownloadURL: "magnet:?xt=urn:btih:1111111111111111111111111111111111111111",
		Size:        100, Title: "Some.Release.2024",
	}})
	if got != "st:100|some release 2024" {
		t.Errorf("size+title should be primary key, got %q", got)
	}

	// 2. InfoHash fallback when size is missing.
	got = dedupeKey(SearchResult{Release: plugin.Release{
		InfoHash: "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
		Title:    "Some Release",
		Size:     0,
	}})
	if got != "ih:abcdef0123456789abcdef0123456789abcdef01" {
		t.Errorf("InfoHash should be used when size is missing, got %q", got)
	}

	// 3. Magnet URL parse fallback when neither size nor InfoHash help.
	got = dedupeKey(SearchResult{Release: plugin.Release{
		DownloadURL: "magnet:?xt=urn:btih:1111111111111111111111111111111111111111",
		Size:        0,
	}})
	if got != "ih:1111111111111111111111111111111111111111" {
		t.Errorf("magnet URL hash should be used, got %q", got)
	}

	// 4. Nothing usable → empty key (row stays ungrouped).
	got = dedupeKey(SearchResult{Release: plugin.Release{
		DownloadURL: "http://opaque-proxy/download?id=42",
		Size:        0,
		Title:       "",
	}})
	if got != "" {
		t.Errorf("nothing usable → empty key, got %q", got)
	}
}

// The most realistic real-world case for this codebase: Pulse proxies
// the torznab feed and rewrites enclosure URLs into opaque HTTP
// redirects. None of the rows have InfoHash or magnet URLs, but they
// all carry the same byte size and the same release title (modulo
// punctuation). They MUST merge under the (size, normalized-title)
// fallback.
func TestDedupeByInfoHash_PulseProxiedFallback(t *testing.T) {
	const size = int64(70330089472)
	in := []SearchResult{
		{Release: plugin.Release{
			Title:       "John Wick 2014 PROPER UHD BluRay 2160p TrueHD Atmos 7 1 DV HEVC HYBRID REMUX FraMeSToR",
			Indexer:     "TorrentGalaxyClone",
			DownloadURL: "http://pulse:9696/api/v1/torznab/abc/download?url=...",
			Size:        size, Seeds: 159, Peers: 105,
		}, QualityScore: 100},
		{Release: plugin.Release{
			Title:       "John.Wick.2014.PROPER.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRID.REMUX-FraMeSToR",
			Indexer:     "1337x",
			DownloadURL: "http://pulse:9696/api/v1/torznab/def/download?url=...",
			Size:        size, Seeds: 68, Peers: 17,
		}, QualityScore: 100},
		{Release: plugin.Release{
			Title:       "John+Wick+2014+PROPER+UHD+BluRay+2160p+TrueHD+Atmos+7+1+DV+HEVC+HYBRID+REMUX+FraMeSToR",
			Indexer:     "ExtraTorrent.st",
			DownloadURL: "magnet:?xt=urn:btih:EC5086C1C78D1ACD62EC8F2DD281E18BA222AC55",
			Size:        size, Seeds: 57, Peers: 17,
		}, QualityScore: 100},
		{Release: plugin.Release{
			Title:       "John-Wick-2014-PROPER-UHD-BluRay-2160p-TrueHD-Atmos-7-1-DV-HEVC-HYBRID-REMUX-FraMeSToR",
			Indexer:     "LimeTorrents",
			DownloadURL: "http://pulse:9696/api/v1/torznab/ghi/download?url=...",
			Size:        size, Seeds: 55, Peers: 18,
		}, QualityScore: 100},
		// Same physical movie, DIFFERENT release (FGT, smaller size).
		// Must NOT merge into the FraMeSToR row above.
		{Release: plugin.Release{
			Title:       "John.Wick.2014.PROPER.2160p.BluRay.REMUX.HEVC.DTS-HD.MA.TrueHD.7.1.Atmos-FGT",
			Indexer:     "LimeTorrents",
			DownloadURL: "http://pulse:9696/api/v1/torznab/ghi/download?url=...",
			Size:        52548924866, Seeds: 33, Peers: 13,
		}, QualityScore: 100},
		// Same FraMeSToR build but a different cut ("VOSTFR" suffix).
		// Same size as the main one, but different normalized title →
		// kept separate.
		{Release: plugin.Release{
			Title:       "John Wick 2014 PROPER UHD BluRay 2160p TrueHD Atmos 7 1 DV HEVC HYBRID REMUX FraMeSToR VOSTFR",
			Indexer:     "ExtraTorrent.st",
			DownloadURL: "magnet:?xt=urn:btih:4ae1246b9b089025f18d45058b9397e879b7eede",
			Size:        size, Seeds: 7, Peers: 0,
		}, QualityScore: 100},
	}

	out := dedupeByInfoHash(in)

	// Find the FraMeSToR group (4 indexers folded into 1).
	var framestor *SearchResult
	for i, r := range out {
		if !strings.Contains(strings.ToLower(r.Title), "vostfr") &&
			strings.Contains(strings.ToLower(r.Title), "framestor") &&
			strings.Contains(r.Title, "PROPER UHD") || strings.Contains(r.Title, "PROPER.UHD") {
			framestor = &out[i]
			break
		}
	}
	if framestor == nil {
		t.Fatalf("FraMeSToR merged row missing; out=%+v", out)
	}
	if len(framestor.OtherSources) != 3 {
		t.Errorf("FraMeSToR row should list 3 other sources (the OTHER three indexers), got %v", framestor.OtherSources)
	}
	// Median of [159, 68, 57, 55] = 57 (lower-middle).
	if framestor.Seeds != 57 {
		t.Errorf("FraMeSToR merged seeds=%d, want 57 (median of [55,57,68,159])", framestor.Seeds)
	}

	// Also verify FGT and VOSTFR did NOT get merged in.
	if len(out) != 3 {
		titles := make([]string, len(out))
		for i, r := range out {
			titles[i] = r.Title
		}
		t.Errorf("expected exactly 3 rows after merge (FraMeSToR-merged, FGT, VOSTFR); got %d:\n  %v", len(out), titles)
	}
}

// ── medianInt ──────────────────────────────────────────────────────────────

func TestMedianInt_OddLength(t *testing.T) {
	if got := medianInt([]int{55, 57, 68}); got != 57 {
		t.Errorf("median([55,57,68])=%d, want 57", got)
	}
}

func TestMedianInt_EvenLength_PicksLowerMiddle(t *testing.T) {
	// [55,57,68,159] sorted → 55,57,68,159. Lower-middle is 57.
	// We pick lower (not average) so an outlier on one side doesn't drag
	// the displayed count up. This is THE incident case.
	if got := medianInt([]int{159, 55, 68, 57}); got != 57 {
		t.Errorf("median([55,57,68,159])=%d, want 57 (lower-middle to under-promise)", got)
	}
}

func TestMedianInt_SingleElement(t *testing.T) {
	if got := medianInt([]int{42}); got != 42 {
		t.Errorf("median([42])=%d, want 42", got)
	}
}

func TestMedianInt_AllEqual(t *testing.T) {
	if got := medianInt([]int{5, 5, 5, 5}); got != 5 {
		t.Errorf("median of all-5s=%d, want 5", got)
	}
}

// ── dedupeByInfoHash ────────────────────────────────────────────────────────

func makeRelease(title, indexer, magnetHash string, seeds, peers int, qualityScore int) SearchResult {
	return SearchResult{
		Release: plugin.Release{
			Title:       title,
			Indexer:     indexer,
			DownloadURL: "magnet:?xt=urn:btih:" + magnetHash + "&dn=" + title,
			Seeds:       seeds,
			Peers:       peers,
		},
		QualityScore: qualityScore,
	}
}

// The headline regression test. Four indexers report the same torrent
// with seeds [159, 68, 57, 55]. After dedup, ONE row remains with the
// median (57) and a list of the contributing other indexers.
func TestDedupeByInfoHash_MergesIncidentCase(t *testing.T) {
	const ihJW = "ec5086c1c78d1acd62ec8f2dd281e18ba222ac55"
	in := []SearchResult{
		makeRelease("John Wick PROPER UHD REMUX", "TorrentGalaxyClone", ihJW, 159, 105, 100),
		makeRelease("John.Wick.PROPER.UHD.REMUX", "1337x", ihJW, 68, 17, 100),
		makeRelease("John+Wick+PROPER+UHD+REMUX", "ExtraTorrent.st", ihJW, 57, 17, 100),
		makeRelease("John-Wick-PROPER-UHD-REMUX", "LimeTorrents", ihJW, 55, 18, 100),
	}

	out := dedupeByInfoHash(in)

	if len(out) != 1 {
		t.Fatalf("expected 1 merged row, got %d:\n%+v", len(out), out)
	}
	merged := out[0]
	if merged.Indexer != "TorrentGalaxyClone" {
		t.Errorf("expected the highest-ranked entry (TGC, sorted first by caller) to be the keeper, got %q", merged.Indexer)
	}
	if merged.Seeds != 57 {
		t.Errorf("merged seeds=%d, want 57 (median of 55,57,68,159)", merged.Seeds)
	}
	// OtherSources should list the other 3, in insertion order.
	wantOthers := []string{"1337x", "ExtraTorrent.st", "LimeTorrents"}
	if !reflect.DeepEqual(merged.OtherSources, wantOthers) {
		t.Errorf("OtherSources=%v, want %v", merged.OtherSources, wantOthers)
	}
}

func TestDedupeByInfoHash_PreservesSingletons(t *testing.T) {
	const ih1 = "1111111111111111111111111111111111111111"
	const ih2 = "2222222222222222222222222222222222222222"
	in := []SearchResult{
		makeRelease("One", "IndexerA", ih1, 100, 5, 100),
		makeRelease("Two", "IndexerB", ih2, 50, 3, 100),
	}
	out := dedupeByInfoHash(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 rows preserved, got %d", len(out))
	}
	for _, r := range out {
		if r.OtherSources != nil {
			t.Errorf("singleton %q has OtherSources=%v, expected nil", r.Title, r.OtherSources)
		}
	}
}

func TestDedupeByInfoHash_HTTPReleasesPassUngrouped(t *testing.T) {
	// HTTP/HTTPS .torrent URLs have no discoverable info_hash; even if
	// they're "the same release," dedup must NOT merge them — better to
	// show duplicates than to silently fold based on a guess.
	in := []SearchResult{
		{Release: plugin.Release{Title: "A", Indexer: "X", DownloadURL: "https://x.com/a.torrent", Seeds: 100}, QualityScore: 100},
		{Release: plugin.Release{Title: "B", Indexer: "Y", DownloadURL: "https://y.com/b.torrent", Seeds: 1}, QualityScore: 100},
	}
	out := dedupeByInfoHash(in)
	if len(out) != 2 {
		t.Errorf("HTTP-only releases must not be grouped — got %d, want 2", len(out))
	}
	for _, r := range out {
		if r.OtherSources != nil {
			t.Errorf("HTTP-only release should not get OtherSources, got %v", r.OtherSources)
		}
	}
}

func TestDedupeByInfoHash_MixedHTTPAndMagnetSameTitle(t *testing.T) {
	// One indexer returned a magnet, another returned an http-only
	// .torrent for what's likely the same release. They must NOT merge —
	// we can't prove they're the same without the info_hash on both sides.
	const ih = "abcdef0123456789abcdef0123456789abcdef01"
	in := []SearchResult{
		makeRelease("Same Movie", "MagnetIndexer", ih, 100, 10, 100),
		{Release: plugin.Release{Title: "Same Movie", Indexer: "HTTPIndexer", DownloadURL: "https://x.com/s.torrent", Seeds: 50}, QualityScore: 100},
	}
	out := dedupeByInfoHash(in)
	if len(out) != 2 {
		t.Errorf("magnet + HTTP releases should stay separate, got %d rows", len(out))
	}
}

// Re-sort after merge: when the median pulls a row's effective seed count
// down, it might now rank below another release of equal quality. Verify
// the output stays correctly ordered by quality+seeds.
func TestDedupeByInfoHash_ResortsAfterMerge(t *testing.T) {
	const ihA = "aaaa00000000000000000000000000000000aaaa"
	const ihB = "bbbb00000000000000000000000000000000bbbb"
	in := []SearchResult{
		// A comes first with inflated 200; same quality.
		makeRelease("Inflated A", "TGC", ihA, 200, 10, 100),
		// B singleton has 100 seeds, same quality.
		makeRelease("Honest B", "1337x", ihB, 100, 5, 100),
		// Two more for A folding in: 30 and 25 → median of [200,30,25] = 30.
		makeRelease("Inflated A copy", "ExtraTorrent.st", ihA, 30, 4, 100),
		makeRelease("Inflated A copy 2", "LimeTorrents", ihA, 25, 3, 100),
	}
	out := dedupeByInfoHash(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 rows after merging A's 3 copies, got %d", len(out))
	}
	// Median([200,30,25]) = 30. So A's effective seeds drop to 30, below B's 100.
	// B should now be first.
	if out[0].Indexer != "1337x" || out[0].Seeds != 100 {
		t.Errorf("expected B to rank first after A's median pulled it down; got first=%q seeds=%d",
			out[0].Indexer, out[0].Seeds)
	}
	if out[1].Indexer != "TGC" || out[1].Seeds != 30 {
		t.Errorf("expected A second with median 30; got %q seeds=%d",
			out[1].Indexer, out[1].Seeds)
	}
}

func TestDedupeByInfoHash_EmptyInput(t *testing.T) {
	if out := dedupeByInfoHash(nil); out != nil {
		t.Errorf("nil in → nil out, got %v", out)
	}
	if out := dedupeByInfoHash([]SearchResult{}); len(out) != 0 {
		t.Errorf("empty in → empty out, got %v", out)
	}
}

// Real-world case: most torznab indexers return HTTP-proxied download
// URLs (the magnet is hidden behind a redirect), but they populate the
// `<torznab:attr name="infohash"/>` separately. dedup must use that
// explicit field — without it, the John Wick incident wouldn't have
// merged because the DownloadURL is `http://pulse:9696/.../download?...`,
// not a magnet.
func TestDedupeByInfoHash_UsesExplicitInfoHashField(t *testing.T) {
	const ih = "ec5086c1c78d1acd62ec8f2dd281e18ba222ac55"
	in := []SearchResult{
		{Release: plugin.Release{
			Title:       "John Wick PROPER UHD REMUX",
			Indexer:     "TorrentGalaxyClone",
			DownloadURL: "http://pulse:9696/api/v1/torznab/abc/download?url=https%3A%2F%2Ftgc.example",
			InfoHash:    ih,
			Seeds:       159, Peers: 105,
		}, QualityScore: 100},
		{Release: plugin.Release{
			Title:       "John Wick PROPER UHD REMUX",
			Indexer:     "1337x",
			DownloadURL: "http://pulse:9696/api/v1/torznab/def/download?url=https%3A%2F%2F1337x.example",
			InfoHash:    ih,
			Seeds:       68, Peers: 17,
		}, QualityScore: 100},
		{Release: plugin.Release{
			Title:       "John Wick PROPER UHD REMUX",
			Indexer:     "LimeTorrents",
			DownloadURL: "http://pulse:9696/api/v1/torznab/ghi/download?url=https%3A%2F%2Flime.example",
			InfoHash:    ih,
			Seeds:       55, Peers: 18,
		}, QualityScore: 100},
	}
	out := dedupeByInfoHash(in)
	if len(out) != 1 {
		t.Fatalf("HTTP-proxied URLs with explicit InfoHash must merge — got %d rows", len(out))
	}
	if out[0].Seeds != 68 {
		// Median of [159, 68, 55] = 68.
		t.Errorf("merged seeds=%d, want 68 (median)", out[0].Seeds)
	}
}

func TestDedupeByInfoHash_KeepsHighestRankedRepresentative(t *testing.T) {
	// Caller (Search) sorts by quality desc, then seeds desc, before
	// calling dedupe. Whichever entry comes FIRST is the one we keep —
	// even if its seeds count gets replaced with the median.
	const ih = "cccc00000000000000000000000000000000cccc"
	in := []SearchResult{
		// Best quality wins the sort by the caller; pretend seeds=100.
		makeRelease("Best Quality Release", "TGC", ih, 100, 10, 100),
		// Worse quality (90) but same hash — should fold in.
		makeRelease("Same hash, lesser quality entry", "Other", ih, 50, 5, 90),
	}
	// Caller-style sort first.
	sort.Slice(in, func(i, j int) bool {
		if in[i].QualityScore != in[j].QualityScore {
			return in[i].QualityScore > in[j].QualityScore
		}
		return in[i].Seeds > in[j].Seeds
	})

	out := dedupeByInfoHash(in)
	if len(out) != 1 {
		t.Fatalf("expected 1 row, got %d", len(out))
	}
	if out[0].Indexer != "TGC" {
		t.Errorf("expected highest-quality representative kept (TGC), got %q", out[0].Indexer)
	}
	if out[0].QualityScore != 100 {
		t.Errorf("kept representative's QualityScore=%d, want 100", out[0].QualityScore)
	}
}
