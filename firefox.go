package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var geckoPlatformMap = map[string]string{
	"linux-amd64":   "linux64",
	"linux-arm64":   "linux-aarch64",
	"linux-386":     "linux32",
	"darwin-amd64":  "macos",
	"darwin-arm64":  "macos-aarch64",
	"windows-amd64": "win64",
	"windows-386":   "win32",
}

func downloadGeckoDriver(ctx context.Context, _, platform, savePath, drvName string) error {
	platform, ok := geckoPlatformMap[platform]
	if !ok {
		return fmt.Errorf("%s is not a supported platform for geckodriver", platform)
	}

	slog.InfoContext(ctx, "download gecko driver", slog.String("platform", platform))

	var ext string
	target := "geckodriver"
	if strings.HasPrefix(platform, "win") {
		ext = "zip"
		target += ".exe"
	} else {
		ext = "tar.gz"
	}

	doc, err := getDocument(ctx, "https://github.com/mozilla/geckodriver/releases")
	if err != nil {
		return err
	}

	suffix := fmt.Sprintf("-%s.%s", platform, ext)

	slog.DebugContext(ctx, "find file url", slog.Group(
		"search",
		slog.String("platform", platform),
		slog.String("ext", ext),
		slog.String("suffix", suffix),
	))

	sections := doc.Find("main section")
	slog.DebugContext(ctx, "found sections", slog.Int("length", sections.Length()))

	if sections.Length() == 0 {
		return fmt.Errorf("not found section")
	}

	ver := sections.First().Find("h2.sr-only").Text()

	slog.DebugContext(
		ctx,
		"found first section",
		slog.String("section", sections.First().AttrOr("aria-labelledby", "")),
		slog.Any("ver", ver),
	)

	doc, err = getDocument(ctx, "https://github.com/mozilla/geckodriver/releases/expanded_assets/v"+ver)
	if err != nil {
		return err
	}

	var dlURL string
	doc.Find("a.Truncate").Each(func(_ int, sel *goquery.Selection) {
		if len(dlURL) > 0 {
			return
		}

		var href string
		href, ok = sel.Attr("href")
		if !ok {
			return
		}

		slog.DebugContext(ctx, "found file url", slog.String("href", href))

		if strings.HasSuffix(href, suffix) {
			dlURL = href
			return
		}
	})

	if len(dlURL) == 0 {
		return fmt.Errorf("not found download link for %s", platform)
	}

	dlURL = "https://github.com" + dlURL
	slog.InfoContext(ctx, "found download URL", slog.String("url", dlURL))

	err = fetchAndSaveFile(ctx, dlURL, target, savePath, drvName)
	if err != nil {
		return err
	}

	return nil
}
