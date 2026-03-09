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
	// ── Code hosting / package registries ──
	"github.com": true, "gitlab.com": true, "codeberg.org": true,
	"bitbucket.org": true, "sr.ht": true, "npmjs.com": true,
	"pypi.org": true, "crates.io": true, "pkg.go.dev": true,
	"sourceforge.net": true,

	// ── Hosting platforms ──
	"github.io": true, "vercel.app": true, "netlify.app": true,
	"pages.dev": true, "web.app": true, "firebaseapp.com": true,
	"fly.dev": true, "render.com": true, "herokuapp.com": true,
	"supabase.co": true, "railway.app": true, "deno.dev": true,
	"readthedocs.io": true, "readthedocs.org": true,
	"substack.com": true,

	// ── Google services ──
	"google.com": true, "youtube.com": true, "googleapis.com": true,
	"gstatic.com": true, "googlevideo.com": true,

	// ── Apple services ──
	"apple.com": true, "icloud.com": true, "mzstatic.com": true,

	// ── Microsoft services ──
	"microsoft.com": true, "live.com": true, "outlook.com": true,
	"office.com": true, "onedrive.com": true,

	// ── Amazon / AWS / CDN infrastructure ──
	"amazon.com": true, "amazonaws.com": true, "cloudfront.net": true,
	"akamaized.net": true, "akamai.net": true, "fastly.net": true,
	"cloudflare.com": true, "cdn77.org": true,

	// ── Social / messaging ──
	"reddit.com": true, "redd.it": true,
	"discord.com": true, "discord.gg": true, "discordapp.com": true,
	"twitter.com": true, "x.com": true,
	"facebook.com": true, "instagram.com": true,
	"telegram.org": true, "telegram.me": true, "telegram.dog": true, "telegram.im": true, "t.me": true,
	"whatsapp.com": true, "signal.org": true, "slack.com": true,
	"matrix.org": true, "matrix.to": true, "element.io": true,
	"vk.com": true, "ok.ru": true,
	"libera.chat": true, "rizon.net": true, "tilde.chat": true,
	"convos.chat": true, "thelounge.chat": true,
	"boards.4chan.org": true,
	"linktr.ee": true, "beacons.ai": true,

	// ── Legal streaming / video platforms ──
	"twitch.tv": true, "vimeo.com": true, "dailymotion.com": true,
	"netflix.com": true, "disneyplus.com": true, "hulu.com": true,
	"hbomax.com": true, "max.com": true, "peacocktv.com": true,
	"paramountplus.com": true, "primevideo.com": true,
	"crunchyroll.com": true, "funimation.com": true,
	"pluto.tv": true, "tubitv.com": true, "corporate.tubitv.com": true,
	"plex.tv": true, "www.plex.tv": true, "watch.plex.tv": true,
	"roku.com": true, "channelstore.roku.com": true, "therokuchannel.roku.com": true,
	"www.arte.tv": true, "distro.tv": true, "fawesome.tv": true,
	"kanopy.com": true, "www.hidive.com": true,
	"www.dcuniverseinfinite.com": true, "www.vudu.com": true,
	"athome.fandango.com": true, "disneynow.com": true,
	"watch.sling.com": true, "play.history.com": true,
	"play.xumo.com": true, "shout-tv.com": true,
	"www.retrocrush.tv": true, "www.hoopladigital.com": true,
	"www.gizmoplex.com": true,
	"nfb.ca": true, "www.nfb.ca": true,
	"odysee.com": true, "rumble.com": true,
	"www.bilibili.com": true, "www.bilibili.tv": true,
	"www.nicovideo.jp": true,
	"www.viddsee.com": true, "www.shortoftheweek.com": true,
	"showroom-live.com": true,
	"www.ondemandchina.com": true, "www.asiancrush.com": true,
	"www3.nhk.or.jp": true,
	"www.adultswim.com": true,
	"cytu.be": true, "joinpeertube.org": true,
	"puffer.stanford.edu": true, "globalshakespeares.mit.edu": true,
	"classics.nascar.com": true,

	// ── Music (legal) ──
	"spotify.com": true, "spotify-dedup.com": true,
	"deezer.com": true, "tidal.com": true, "soundcloud.com": true,
	"bandcamp.com": true, "get.bandcamp.help": true,
	"music.amazon.com": true, "music.apple.com": true,
	"last.fm": true, "www.last.fm": true,
	"rateyourmusic.com": true, "musicbrainz.org": true,
	"listenbrainz.org": true, "libre.fm": true,
	"mora.jp": true, "ototoy.jp": true, "recochoku.jp": true,
	"www.qobuz.com": true, "play.qobuz.com": true,
	"www.discogs.com": true,
	"stats.fm": true, "volt.fm": true,
	"www.songkick.com": true, "radio.garden": true,
	"everynoise.com": true, "kworb.net": true,
	"www.billboard-japan.com": true,
	"www.oricon.co.jp": true, "us.oricon-group.com": true,
	"discoverquickly.com": true, "exportify.app": true,
	"spotifyplaylistarchive.com": true,
	"www.playlistsorter.com": true, "www.spotlistr.com": true,

	// ── Gaming platforms ──
	"steampowered.com": true, "store.steampowered.com": true, "steam.com": true,
	"gog.com": true, "epicgames.com": true,
	"playstation.com": true, "xbox.com": true, "nintendo.com": true,

	// ── Media databases / tracking ──
	"imdb.com": true, "www.imdb.com": true,
	"anilist.co": true, "myanimelist.net": true,
	"letterboxd.com": true, "trakt.tv": true,
	"themoviedb.org": true, "www.themoviedb.org": true,
	"thetvdb.com": true, "www.thetvdb.com": true,
	"anidb.net": true, "kitsu.app": true, "kitsu.io": true,
	"simkl.com": true, "www.anime-planet.com": true,
	"www.livechart.me": true, "www.mangaupdates.com": true,
	"www.anisearch.com": true, "www.anisearch.de": true,
	"vndb.org": true, "mydramalist.com": true,
	"www.goodreads.com": true, "bookmeter.com": true,
	"www.rottentomatoes.com": true, "www.boxofficemojo.com": true,
	"www.allmovie.com": true, "blu-ray.com": true, "www.blu-ray.com": true,
	"www.highdefdigest.com": true, "www.tvmaze.com": true,
	"www.justwatch.com": true, "comicvine.gamespot.com": true,
	"www.comics.org": true, "www.behindthevoiceactors.com": true,
	"www.commonsensemedia.org": true, "www.doesthedogdie.com": true,
	"www.unconsentingmedia.org": true,
	"imsdb.com": true, "tvtropes.org": true,
	"episodecalendar.com": true, "next-episode.net": true,
	"tastedive.com": true, "taste.io": true, "www.taste.io": true,
	"www.serializd.com": true,
	"www.whats-on-netflix.com": true,
	"runpee.com": true, "www.bechdeltest.com": true,
	"titantv.com": true, "tvark.org": true,

	// ── Legal manga / comics / light novel publishers ──
	"mangaplus.shueisha.co.jp": true, "j-novel.club": true,
	"global.bookwalker.jp": true,
	"www.viz.com": true, "yenpress.com": true,
	"kodansha.us": true, "kmanga.kodansha.com": true,
	"sevenseasentertainment.com": true,
	"www.darkhorse.com": true, "www.marvel.com": true,
	"tokyopop.com": true, "titan-comics.com": true,
	"tapas.io": true, "www.webtoons.com": true,
	"www.tappytoon.com": true, "globalcomix.com": true,
	"comics.inkr.com": true, "comikey.com": true,
	"www.mangamo.com": true, "www.lezhinus.com": true,
	"squareenixmangaandbooks.square-enix-games.com": true,
	"comic.pixiv.net": true, "global.manga-up.com": true,
	"www.gocomics.com": true, "www.comicsbeat.com": true,
	"leagueofcomicgeeks.com": true,
	"www.webnovel.com": true, "www.wuxiaworld.com": true,

	// ── Reading tools / apps ──
	"calibre-ebook.com": true, "www.audiobookshelf.org": true,
	"koreader.rocks": true, "www.sumatrapdfreader.org": true,
	"prologue.audio": true, "sigil-ebook.com": true,
	"mihon.app": true, "aniyomi.org": true,
	"aidoku.app": true, "tachimanga.app": true,
	"paperback.moe": true, "shosetsu.app": true,
	"www.lnreader.app": true, "readest.com": true,
	"librumreader.com": true, "opencomic.app": true,
	"www.yacreader.com": true,

	// ── Cloud storage ──
	"dropbox.com": true, "mega.nz": true, "mega.io": true,
	"mediafire.com": true, "drive.google.com": true,
	"www.4shared.com": true,

	// ── Reference / wiki ──
	"archive.org": true, "www.archive.org": true,
	"wikipedia.org": true, "en.wikipedia.org": true,
	"commons.wikimedia.org": true,
	"fandom.com": true, "www.wikiwand.com": true,
	"en.namu.wiki": true, "en.touhouwiki.net": true,
	"alternativeto.net": true, "www.openculture.com": true,
	"pastebin.com": true, "rentry.co": true, "rentry.org": true,
	"graph.org": true,

	// ── Subtitles ──
	"opensubtitles.org": true, "www.opensubtitles.org": true,
	"www.opensubtitles.com": true, "subscene.com": true,
	"subtitletools.com": true, "subdl.com": true,
	"subsource.net": true, "downsub.com": true,
	"freesubtitles.ai": true, "turboscribe.ai": true,
	"www.nikse.dk": true, "www.jubler.org": true,

	// ── News / media outlets ──
	"www.bbc.co.uk": true, "natalie.mu": true,
	"www.animenewsnetwork.com": true,
	"animationbusiness.info": true,

	// ── Education / government / archives ──
	"defense.gov": true, "www.loc.gov": true, "plus.nasa.gov": true,
	"www.nfsa.gov.au": true, "images.defence.gov.au": true,
	"www.defenceimagery.mod.uk": true,
	"www.iitk.ac.in": true, "diva.sfsu.edu": true,
	"digitaler-lesesaal.bundesarchiv.de": true,
	"www.dvidshub.net": true, "texasarchive.org": true,
	"wiki.archiveteam.org": true,
	"www.filmpreservation.org": true, "www.chicagofilmarchives.org": true,
	"www.sprocketschool.org": true, "www.softwareheritage.org": true,
	"movingimage.nls.uk": true, "www.nls.uk": true,
	"www.ngataonga.org.nz": true, "www.nzonscreen.com": true,
	"www.europeanfilmgateway.eu": true, "www.iwm.org.uk": true,
	"player.bfi.org.uk": true, "www.bfi.org.uk": true,
	"www.colonialfilm.org.uk": true,
	"www.britishpathe.com": true, "www.historicfilms.com": true,
	"www.huntleyarchives.com": true, "www.cinematheque.fr": true,
	"www.filmmuseum.at": true, "www.filmportal.de": true,
	"www.stumfilm.dk": true, "stiftung-imai.de": true,
	"ifiarchiveplayer.ie": true,
	"animation.filmarchives.jp": true, "meiji.filmarchives.jp": true,
	"publicdomainmovie.net": true, "classiccinemaonline.com": true,
	"cd.textfiles.com": true, "footagefarm.com": true,

	// ── Developer tools / open-source projects ──
	"ffmpeg.org": true, "mpv.io": true,
	"greasyfork.org": true, "openuserjs.org": true,
	"flathub.org": true, "apps.kde.org": true,
	"invent.kde.org": true, "okular.kde.org": true,
	"konversation.kde.org": true,
	"gitlab.freedesktop.org": true, "poppler.freedesktop.org": true,
	"directory.fsf.org": true, "fossies.org": true,
	"www.fosshub.com": true, "suckless.org": true,
	"webtorrent.io": true, "instant.io": true,
	"www.qbittorrent.org": true, "jdownloader.org": true,
	"www.foobar2000.org": true,
	"portableapps.com": true, "www.portablefreeware.com": true,
	"www.nirsoft.net": true, "www.majorgeeks.com": true, "oldergeeks.com": true,
	"xdaforums.com": true, "www.apkmirror.com": true,
	"revanced.app": true, "huggingface.co": true,
	"tailscale.com": true, "flexget.com": true,
	"ios.cfw.guide": true,
	"www.virustotal.com": true,
	"www.videohelp.com": true, "www.xpdfreader.com": true,
	"www.softpedia.com": true, "software.informer.com": true,
	"www.oldversion.com": true, "vetusware.com": true,
	"www.grc.com": true,
	"newreleases.io": true,
	"store.rg-adguard.net": true,
	"awesomeopensource.com": true, "opensource.builders": true,
	"openalternative.co": true, "www.opensourcealternative.to": true,
	"alternativeoss.com": true, "european-alternatives.eu": true,
	"isitreallyfoss.com": true,

	// ── Media servers / home media ──
	"jellyfin.org": true, "emby.media": true, "kodi.tv": true,
	"www.stremio.com": true, "web.stremio.com": true,
	"sonarr.tv": true, "radarr.video": true, "lidarr.audio": true,
	"www.bazarr.media": true,
	"sabnzbd.org": true, "nzbget.com": true,
	"pymedusa.com": true,
	"komga.org": true, "www.kavitareader.com": true,
	"spicetify.app": true,

	// ── Music tools / audio software ──
	"beets.io": true, "beets.readthedocs.io": true,
	"nicotine-plus.org": true, "slsknet.org": true, "www.slsknet.org": true,
	"mopidy.com": true, "koel.dev": true,
	"www.navidrome.org": true, "www.funkwhale.audio": true,
	"www.subsonic.org": true,
	"www.freac.org": true, "exactaudiocopy.de": true,
	"www.mp3tag.de": true, "cue.tools": true, "cuetools.net": true,
	"www.dbpoweramp.com": true, "www.spek.cc": true,
	"www.sonicvisualiser.org": true, "www.rarewares.org": true,
	"powerampapp.com": true, "symfonium.app": true,
	"www.ableton.com": true, "www.presonus.com": true,
	"www.steinberg.net": true, "www.image-line.com": true,
	"dreamtonics.com": true, "piaprostudio.com": true,
	"www.vocaloid.com": true, "cevio.jp": true,

	// ── Mozilla / browsers ──
	"mozilla.org": true, "addons.mozilla.org": true,
	"brave.com": true, "opera.com": true,
	"adiirc.com": true,

	// ── DNS / networking ──
	"nextdns.io": true, "adguard.com": true, "quad9.net": true,
	"opendns.com": true,

	// ── Hardware / misc vendors ──
	"samsung.com": true, "www.samsungtvplus.com": true,
	"www.webosbrew.org": true,

	// ── Video calls ──
	"zoom.us": true, "skype.com": true,

	// ── Anime / manga / JP media (official) ──
	"en.gundam.info": true, "gundam.info": true,
	"tsuburaya-prod.com": true, "www.ultramanconnection.com": true,
	"theapplewiki.com": true,

	// ── Demoscene / digital art ──
	"scene.org": true, "files.scene.org": true,
	"demozoo.org": true, "www.pouet.net": true,
	"defacto2.net": true,

	// ── Internet radio / podcasts ──
	"radio.co": true, "embed.radio.co": true, "streams.radio.co": true,
	"www.radio-browser.info": true,

	// ── Misc legitimate services ──
	"acestream.org": true, "teamup.com": true,
	"www.veed.io": true, "tv.naver.com": true,
	"sheet.zohopublic.com": true,
	"capture2text.sourceforge.net": true, "deadbeef.sourceforge.io": true,
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
