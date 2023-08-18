package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var chromePlatformMap = map[string]string{
	"linux-amd64":   "linux64",
	"darwin-amd64":  "mac-x64",
	"darwin-arm64":  "mac-arm64",
	"windows-amd64": "win64",
	"windows-386":   "win32",
}

func downloadChromeDriver(ctx context.Context, channel, platform, savePath, drvName string) error {
	platform, ok := chromePlatformMap[platform]
	if !ok {
		// chromedriver がサポートしていないプラットフォーム、というエラーを返す。
		return fmt.Errorf("%s is not a supported platform for chromedriver", platform)
	}

	slog.InfoContext(ctx, "download chrome driver", slog.String("platform", platform))

	doc, err := getDocument(ctx, "https://googlechromelabs.github.io/chrome-for-testing/")
	if err != nil {
		return err
	}

	var dlURL string
	doc.Find(fmt.Sprintf("#%s table tbody tr.status-ok", channel)).Each(func(i int, sel *goquery.Selection) {
		s := strings.TrimSpace(sel.Find("th:first-child > code").Text())

		slog.DebugContext(ctx, "found.", slog.String("Binary", s))

		if s != "chromedriver" {
			return
		}

		s = strings.TrimSpace(sel.Find("th:nth-child(2) > code").Text())

		slog.DebugContext(ctx, "found.", slog.String("Platform", s))

		if s != platform {
			return
		}

		s = strings.TrimSpace(sel.Find("td:nth-child(4)").Text())

		slog.DebugContext(ctx, "found.", slog.String("HTTP status", s))

		if s != "200" {
			return
		}

		s = strings.TrimSpace(sel.Find("td:nth-child(3)").Text())

		slog.DebugContext(ctx, "found.", slog.String("URL", s))

		if len(s) > 0 {
			dlURL = s
		}
	})

	if len(dlURL) == 0 {
		return errors.New("download URL not found")
	}

	slog.InfoContext(ctx, "found download URL", slog.String("URL", dlURL))

	target := "chromedriver"
	if strings.HasPrefix(platform, "win") {
		target += ".exe"
	}

	err = fetchAndSaveFile(ctx, dlURL, target, savePath, drvName)
	if err != nil {
		return err
	}

	return nil
}
