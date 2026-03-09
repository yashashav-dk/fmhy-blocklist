package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── Source definitions ──────────────────────────────────────────────────────

type Source struct {
	Name        string
	URL         string
	FallbackURL string
}

var sources = []Source{
	{
		Name:        "FMHY Streaming",
		URL:         "https://raw.githubusercontent.com/wiki/fmhy/FMHY/Streaming.md",
		FallbackURL: "https://raw.githubusercontent.com/fmhy/FMHY/main/docs/videopiracyguide.md",
	},
	{
		Name: "FMHY Downloading",
		URL:  "https://raw.githubusercontent.com/wiki/fmhy/FMHY/Downloading.md",
	},
	{
		Name: "Wotaku Websites",
		URL:  "https://raw.githubusercontent.com/wotakumoe/wotaku/main/docs/websites.md",
	},
	{
		Name: "Wotaku Music",
		URL:  "https://raw.githubusercontent.com/wotakumoe/wotaku/main/docs/music.md",
	},
	{
		Name: "Wotaku Software",
		URL:  "https://raw.githubusercontent.com/wotakumoe/wotaku/main/docs/software.md",
	},
	{
		Name: "Wotaku Non-English",
		URL:  "https://raw.githubusercontent.com/wotakumoe/wotaku/main/docs/nonen.md",
	},
	{
		Name: "Wotaku Misc",
		URL:  "https://raw.githubusercontent.com/wotakumoe/wotaku/main/docs/misc.md",
	},
	{
		Name: "Wotaku FAQ",
		URL:  "https://raw.githubusercontent.com/wotakumoe/wotaku/main/docs/faq.md",
	},
}

// ── Allowlist (never blocked) ───────────────────────────────────────────────

var allowlist = map[string]bool{
	// Code hosting / package registries
	"github.com": true, "gitlab.com": true, "codeberg.org": true,
	"bitbucket.org": true, "sr.ht": true, "npmjs.com": true,
	"pypi.org": true, "crates.io": true, "pkg.go.dev": true,

	// Hosting platforms (github.io, vercel, netlify, cloudflare pages, etc.)
	"github.io": true, "vercel.app": true, "netlify.app": true,
	"pages.dev": true, "web.app": true, "firebaseapp.com": true,
	"fly.dev": true, "render.com": true, "herokuapp.com": true,
	"supabase.co": true, "railway.app": true, "deno.dev": true,

	// Google services
	"google.com": true, "youtube.com": true, "googleapis.com": true,
	"gstatic.com": true, "googlevideo.com": true,

	// Apple services
	"apple.com": true, "icloud.com": true, "mzstatic.com": true,

	// Microsoft services
	"microsoft.com": true, "live.com": true, "outlook.com": true,
	"office.com": true, "onedrive.com": true,

	// Amazon / AWS / CDN infrastructure
	"amazon.com": true, "amazonaws.com": true, "cloudfront.net": true,
	"akamaized.net": true, "akamai.net": true, "fastly.net": true,
	"cloudflare.com": true, "cdn77.org": true,

	// Social / messaging
	"reddit.com": true, "www.reddit.com": true,
	"discord.com": true, "discord.gg": true, "discordapp.com": true,
	"twitter.com": true, "x.com": true,
	"facebook.com": true, "instagram.com": true,
	"telegram.org": true, "telegram.me": true, "telegram.dog": true, "telegram.im": true, "t.me": true,
	"whatsapp.com": true, "signal.org": true, "slack.com": true,
	"matrix.org": true, "element.io": true,

	// Streaming (legal)
	"www.youtube.com": true, "music.youtube.com": true,
	"twitch.tv": true, "www.twitch.tv": true,
	"vimeo.com": true, "dailymotion.com": true, "www.dailymotion.com": true,
	"netflix.com": true, "disneyplus.com": true, "hulu.com": true,
	"hbomax.com": true, "max.com": true, "peacocktv.com": true,
	"paramountplus.com": true, "primevideo.com": true,
	"crunchyroll.com": true, "www.crunchyroll.com": true, "funimation.com": true,
	"puffer.stanford.edu": true, "globalshakespeares.mit.edu": true,

	// Music (legal)
	"spotify.com": true, "support.spotify.com": true,
	"deezer.com": true, "www.deezer.com": true,
	"tidal.com": true, "soundcloud.com": true,
	"bandcamp.com": true, "get.bandcamp.help": true,
	"music.amazon.com": true, "music.apple.com": true,
	"www.last.fm": true, "rateyourmusic.com": true,
	"musicbrainz.org": true,

	// Gaming platforms
	"store.steampowered.com": true, "steampowered.com": true, "steam.com": true,
	"gog.com": true, "epicgames.com": true,
	"playstation.com": true, "xbox.com": true, "nintendo.com": true,

	// Media databases / trackers
	"imdb.com": true, "www.imdb.com": true,
	"anilist.co": true, "myanimelist.net": true,
	"letterboxd.com": true, "trakt.tv": true,
	"themoviedb.org": true, "www.themoviedb.org": true,
	"thetvdb.com": true, "www.thetvdb.com": true,

	// Legal manga / light novel
	"mangaplus.shueisha.co.jp": true, "j-novel.club": true,
	"global.bookwalker.jp": true,

	// Reading tools
	"calibre-ebook.com": true, "www.audiobookshelf.org": true,
	"koreader.rocks": true, "www.sumatrapdfreader.org": true,
	"prologue.audio": true,

	// Cloud storage
	"dropbox.com": true, "mega.nz": true, "mediafire.com": true,
	"drive.google.com": true,

	// Reference / wiki
	"archive.org": true, "www.archive.org": true,
	"en.wikipedia.org": true, "wikipedia.org": true,
	"fandom.com": true,

	// Subtitles
	"opensubtitles.org": true, "www.opensubtitles.org": true,
	"subscene.com": true,

	// Mozilla / browsers
	"mozilla.org": true, "addons.mozilla.org": true,
	"brave.com": true, "opera.com": true,
	"chromewebstore.google.com": true, "chrome.google.com": true,

	// DNS / networking
	"nextdns.io": true, "adguard.com": true, "quad9.net": true,
	"opendns.com": true,

	// Samsung / misc hardware
	"samsung.com": true, "www.samsungtvplus.com": true,

	// Zoom / video calls
	"zoom.us": true, "skype.com": true,
}

const etagFile = ".etags.json"

func loadETags() map[string]string {
	data, err := os.ReadFile(etagFile)
	if err != nil {
		return make(map[string]string)
	}
	var etags map[string]string
	if err := json.Unmarshal(data, &etags); err != nil {
		return make(map[string]string)
	}
	return etags
}

func saveETags(etags map[string]string) {
	data, _ := json.MarshalIndent(etags, "", "  ")
	os.WriteFile(etagFile, data, 0644)
}

type FetchResult struct {
	Body    string
	ETag    string
	Changed bool
}

func fetchWithETag(url, cachedETag string) (*FetchResult, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fmhy-blocklist/1.0")
	if cachedETag != "" {
		req.Header.Set("If-None-Match", cachedETag)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return &FetchResult{Changed: false, ETag: cachedETag}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &FetchResult{Body: string(body), ETag: resp.Header.Get("ETag"), Changed: true}, nil
}

var domainRegex = regexp.MustCompile(
	`(?:https?://)?([a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,})`,
)

func isAllowed(domain string) bool {
	if allowlist[domain] {
		return true
	}
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts)-1; i++ {
		parent := strings.Join(parts[i:], ".")
		if allowlist[parent] {
			return true
		}
	}
	return false
}

func extractDomains(body string) []string {
	matches := domainRegex.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	var domains []string
	for _, match := range matches {
		domain := strings.ToLower(match[1])
		if strings.HasSuffix(domain, ".png") || strings.HasSuffix(domain, ".jpg") ||
			strings.HasSuffix(domain, ".gif") || strings.HasSuffix(domain, ".svg") ||
			strings.HasSuffix(domain, ".webp") || strings.HasSuffix(domain, ".ico") ||
			strings.HasSuffix(domain, ".css") || strings.HasSuffix(domain, ".js") {
			continue
		}
		if len(domain) < 4 || !strings.Contains(domain, ".") {
			continue
		}
		if !seen[domain] && !isAllowed(domain) {
			seen[domain] = true
			domains = append(domains, domain)
		}
	}
	return domains
}

type ScrapeResult struct {
	Source  string
	Domains []string
	Err     error
}

type Stats struct {
	LastUpdated  string         `json:"last_updated"`
	TotalDomains int           `json:"total_domains"`
	Sources      map[string]int `json:"sources"`
}

func main() {
	fmt.Println("fmhy-blocklist: starting scrape...")
	start := time.Now()
	etags := loadETags()
	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([]ScrapeResult, len(sources))
	anyChanged := false

	for i, src := range sources {
		wg.Add(1)
		go func(idx int, s Source) {
			defer wg.Done()
			cachedETag := etags[s.URL]
			result, err := fetchWithETag(s.URL, cachedETag)
			if err != nil && s.FallbackURL != "" {
				fmt.Printf("  [%s] primary failed (%v), trying fallback...\n", s.Name, err)
				cachedETag = etags[s.FallbackURL]
				result, err = fetchWithETag(s.FallbackURL, cachedETag)
				if err == nil && result.ETag != "" {
					mu.Lock()
					etags[s.FallbackURL] = result.ETag
					mu.Unlock()
				}
			} else if err == nil && result.ETag != "" {
				mu.Lock()
				etags[s.URL] = result.ETag
				mu.Unlock()
			}
			if err != nil {
				results[idx] = ScrapeResult{Source: s.Name, Err: err}
				return
			}
			if !result.Changed {
				fmt.Printf("  [%s] unchanged (ETag match)\n", s.Name)
				results[idx] = ScrapeResult{Source: s.Name}
				return
			}
			mu.Lock()
			anyChanged = true
			mu.Unlock()
			domains := extractDomains(result.Body)
			fmt.Printf("  [%s] %d domains extracted\n", s.Name, len(domains))
			results[idx] = ScrapeResult{Source: s.Name, Domains: domains}
		}(i, src)
	}
	wg.Wait()

	if !anyChanged {
		if _, err := os.Stat("blocklist.txt"); err == nil {
			fmt.Printf("fmhy-blocklist: no changes detected, done in %s\n", time.Since(start).Round(time.Millisecond))
			saveETags(etags)
			return
		}
		fmt.Println("  output files missing, regenerating...")
	}

	seen := make(map[string]bool)
	var allDomains []string
	sourceCounts := make(map[string]int)
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  WARNING: [%s] failed: %v\n", r.Source, r.Err)
			continue
		}
		count := 0
		for _, d := range r.Domains {
			if !seen[d] {
				seen[d] = true
				allDomains = append(allDomains, d)
				count++
			}
		}
		sourceCounts[r.Source] = count
	}
	sort.Strings(allDomains)

	// Write blocklist.txt (uBlock format)
	var sb strings.Builder
	sb.WriteString("! Title: FMHY Binge Blocker\n")
	sb.WriteString(fmt.Sprintf("! Description: Auto-generated blocklist from FMHY + Wotaku (%d domains)\n", len(allDomains)))
	sb.WriteString(fmt.Sprintf("! Last updated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString("! Homepage: https://github.com/yashashav-dk/fmhy-blocklist\n")
	sb.WriteString("! License: MIT\n")
	sb.WriteString("!\n! ── uBlock Origin filters ──\n!\n")
	for _, d := range allDomains {
		sb.WriteString(fmt.Sprintf("||%s^\n", d))
	}
	sb.WriteString("!\n! ── /etc/hosts format (for apply-hosts.sh) ──\n!\n")
	for _, d := range allDomains {
		sb.WriteString(fmt.Sprintf("! 0.0.0.0 %s\n", d))
	}
	if err := os.WriteFile("blocklist.txt", []byte(sb.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write blocklist.txt: %v\n", err)
		os.Exit(1)
	}

	// Write domains.txt (plain domain list for NextDNS / AdGuard / Pi-hole)
	var db strings.Builder
	db.WriteString("# FMHY Binge Blocker — plain domain list\n")
	db.WriteString(fmt.Sprintf("# %d domains | updated %s\n", len(allDomains), time.Now().UTC().Format(time.RFC3339)))
	db.WriteString("# https://github.com/yashashav-dk/fmhy-blocklist\n#\n")
	for _, d := range allDomains {
		db.WriteString(d + "\n")
	}
	if err := os.WriteFile("domains.txt", []byte(db.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write domains.txt: %v\n", err)
		os.Exit(1)
	}

	// Write stats.json
	stats := Stats{
		LastUpdated:  time.Now().UTC().Format(time.RFC3339),
		TotalDomains: len(allDomains),
		Sources:      sourceCounts,
	}
	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	if err := os.WriteFile("stats.json", statsJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write stats.json: %v\n", err)
		os.Exit(1)
	}
	saveETags(etags)
	fmt.Printf("fmhy-blocklist: wrote %d domains to blocklist.txt + domains.txt in %s\n",
		len(allDomains), time.Since(start).Round(time.Millisecond))
}
