package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var edgePlatformMap = map[string]string{
	"linux-amd64":   "linux64",
	"darwin-amd64":  "mac64",
	"darwin-arm64":  "mac64_m1",
	"windows-amd64": "win64",
	"windows-386":   "win32",
}

func downloadEdgeDriver(ctx context.Context, channel, platform, savePath, drvName string) error {
	labelPlatform, ok := edgePlatformMap[platform]
	if !ok {
		return fmt.Errorf("invalid platform: %s", platform)
	}

	pageUrl := "https://developer.microsoft.com/en-us/microsoft-edge/tools/webdriver/?form=MA13LH"

	slog.InfoContext(
		ctx,
		"get page",
		slog.String("target", "edge"),
		slog.String("platform", labelPlatform),
		slog.String("pageUrl", pageUrl),
	)

	doc, err := getDocument(ctx, pageUrl)
	if err != nil {
		return err
	}

	// ダウンロードファイル名の例
	// Windows Arm64:
	// edgedriver_arm64.zip
	//
	// Windows x64:
	// edgedriver_win64.zip
	//
	// Windows x86:
	// edgedriver_win32.zip
	//
	// Mac x64:
	// edgedriver_mac64.zip
	//
	// Mac arm64:
	// edgedriver_mac64_m1.zip
	//
	// Linux64:
	// edgedriver_linux64.zip

	parentSel := `div[data-fetch-key="block-web-driver:0"] .common-card-list__card`

	slog.DebugContext(ctx, "parent selector", slog.String("selector", parentSel))

	var childIdx int
	switch channel {
	case "stable":
		childIdx = 0
	case "beta":
		childIdx = 1
	case "dev":
		childIdx = 2
	case "canary":
		childIdx = 3
	}

	parent := doc.Find(parentSel).Eq(childIdx)
	if parent.Length() == 0 {
		return fmt.Errorf("not found parent element: %s %s", channel, platform)
	}

	sel := "div.block-web-driver__version-links > a"

	slog.DebugContext(ctx, "selector", slog.String("selector", sel))

	links := parent.Find(sel)

	slog.DebugContext(ctx, "links", slog.Int("length", links.Length()))

	if links.Length() == 0 {
		return fmt.Errorf("not found download link: %s %s", channel, platform)
	}

	suffix := fmt.Sprintf("edgedriver_%s.zip", labelPlatform)

	var dlURL string
	links.Each(func(i int, selection *goquery.Selection) {
		if len(dlURL) > 0 {
			return
		}

		href, ok := selection.Attr("href")
		if !ok {
			return
		}

		slog.DebugContext(ctx, "found href", slog.String("href", href))

		if strings.HasSuffix(href, suffix) {
			slog.DebugContext(ctx, "found download link", slog.String("href", href))
			dlURL = href
		}
	})

	if len(dlURL) == 0 {
		return fmt.Errorf("not found href attribute: %s %s", channel, platform)
	}

	slog.InfoContext(ctx, "download edge driver", slog.Group("target",
		slog.String("channel", channel),
		slog.String("platform", labelPlatform),
		slog.String("url", dlURL),
	))

	target := "msedgedriver"
	if strings.HasPrefix(platform, "win") {
		target += ".exe"
	}

	err = fetchAndSaveFile(ctx, dlURL, target, savePath, drvName)
	if err != nil {
		return err
	}

	return nil
}
