# K2 — Promotion Plan & Content Plan

Hub document for promoting K2 (server monitoring, Go + htmx + PicoCSS, single static binary).

## 1. Positioning

K2 — a free, lightweight alternative to Netdata/Grafana for personal servers, VPS, and home hardware.

Key facts for content:

- Single static binary without CGO — zero dependencies, runs on minimal hardware (Raspberry Pi, old laptop, 512 MB VPS)
- Web UI built with htmx + PicoCSS — no node_modules, no frontend build step
- Monitors system (CPU/RAM/disk/network), processes, and Docker containers
- SQLite with retention and auto-purge
- One-command install: `curl -fsSL ... | sudo bash`
- Docker Compose support
- Auth with account lockout after failed attempts

## 2. Goal

- GitHub stars
- Real users (installs, issues, feedback)

## 3. Platform Matrix

### Primary (core)

| Platform | Role | Format | Frequency |
|---|---|---|---|
| **Website** | HUB: master articles, SEO, own domain, canonical origin | longreads RU (EN optional) | 1 article / 2 wks |
| **Habr** | RU spoke: adaptations of master articles, canonical to the site | longreads | 1 article / 2 wks |
| **X** | EN feed | teasers, feature threads, screenshots | 3–5 posts / wk |
| **dev.to** | EN spoke: adaptations of EN master articles, canonical to the site | longreads | 1 article / 2 wks |
| **YouTube** | RU video | screencasts + Shorts | 1 screencast + 3–4 Shorts / mo |
| **Telegram** | RU community, ties everything together | announcements, screenshots, changelog | 3–5 posts / wk |
| **GitHub** | showcase: README, GIF demos, releases | — | on every release |

### Excluded (deliberately)

| Platform | Reason |
|---|---|
| Reddit | author's decision |
| LinkedIn | requires native format, low ROI for a tech audience |
| Dzen | algorithm suppresses cross-posts, only works with native content |
| TJournal | editorial platform, reluctant to accept self-promotion |

## 4. Cross-Posting Rules

**Hub-and-spoke model:** 1 master article on the site → adaptations on the other platforms.

- **SEO platforms (site, Habr, dev.to):** Do NOT post identical copies without canonical. Master goes on the site, Habr (RU) and dev.to (EN) republish with `<link rel="canonical">` pointing to the original. (If the master lives on a platform — canonical from the other platforms to it.)
- **Feeds/social (X, Telegram):** 3–5 line teaser + link. Do not post full texts.
- **YouTube:** not a copy of the article, but repurposing (screencast, overview, tutorial).
- Full copies are acceptable only between non-indexed platforms.

## 5. Content Plan: 90 Days

### Phase 0 — Foundation (week 0)

1. README upgrade: GIF demo instead of static screenshots, badges (build / license / go version).
2. Site SEO audit: blog structure, meta tags, sitemap, canonical setup for master articles.
3. Create Telegram channel.
4. Prepare accounts: X, YouTube.

### Phase 1 — Launch Wave (weeks 1–6)

1. RU launch article on the site + Habr adaptation (canonical).
2. Show thread on X (EN): features, screenshots, link to the site.
3. YouTube screencast: 2-minute install + UI overview → 3–4 Shorts.
4. Telegram: announcements, screenshots, changelog.

Launch wave topics:

- "K2 — server monitoring with a single dependency-free binary"
- "Installing K2 on a VPS in 2 minutes" (video + text)

### Phase 2 — Deep Dive (weeks 7–12)

- "K2 vs Netdata and Grafana — what should a solo developer use"
- "Monitoring a Raspberry Pi / home server" (ace card: minimal hardware)
- "Docker container monitoring for free"
- "Building your own fork from source" (go install)
- Changelog posts for every release (project liveliness = stars)

## 6. Weekly Rhythm (10+ hrs/wk)

| Action | Volume |
|---|---|
| RU article on the site | 1 / 2 wks |
| Habr adaptation (canonical, RU) | 1 / 2 wks |
| dev.to adaptation (canonical, EN) | 1 / 2 wks |
| X posts | 3–5 / wk |
| Telegram posts | 3–5 / wk |
| YouTube screencast | 1 / mo |
| Shorts | 3–4 / mo |

## 7. Article Publication Checklist

1. Master article → site (RU; EN if the topic is international).
2. Habr adaptation (RU) + canonical to the site.
3. dev.to adaptation (EN) + canonical to the site (if the master article has an EN version).
4. Teaser post in Telegram with a link to the site.
5. Thread/teaser on X (EN version).
6. If there is video — screencast + Shorts on YouTube.

## 8. Release Checklist (GitHub)

1. README update (if features/install changed).
2. Release + changelog post.
3. Telegram: announcement.
4. X: announcement.

## 9. What Is Needed for Execution

- Site URL and structure (engine, how articles are added) — for SEO audit and canonical setup
- Access to the GitHub repo — for the README upgrade
- Drafts of the first articles (launch wave)