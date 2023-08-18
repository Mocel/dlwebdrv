package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/m-mizutani/clog"
)

var (
	Version  string
	Revision string

	debug bool

	// optSavePath は、ダウンロードした WebDriver を保存するパスを表す。
	optSavePath string

	// optPlatform は、ダウンロードする WebDriver のプラットフォームを表す。
	optPlatform string
	// optChannel は、ダウンロードする WebDriver のチャンネルを表す。
	optChannel string
	// optDriverName は、保存する WebDriver のファイル名を別名にしたい場合に指定する。
	optDriverName string

	supportedBrowsers = []string{
		"chrome",
		"firefox",
		"edge",
	}

	supportedPlatforms = []string{
		"linux-amd64",
		"linux-arm64",
		"linux-386",
		"darwin-amd64",
		"darwin-arm64",
		"windows-amd64",
		"windows-386",
	}

	supportedChannels = []string{
		"stable",
		"beta",
		"dev",
		"canary",
	}

	httpClient *http.Client
)

func verString() string {
	if Version == "" {
		Version = "dev"
	}
	if Revision == "" {
		Revision = "unknown"
	}

	return fmt.Sprintf("version %s (%s)", Version, Revision)
}

func validateOpts(args []string) error {
	errMsg := make([]string, 0)

	if len(args) == 0 {
		errMsg = append(errMsg, "browser is required")
	}

	for _, arg := range args {
		if !slices.Contains(supportedBrowsers, arg) {
			errMsg = append(errMsg, fmt.Sprintf("browser %s is invalid, can use %s", arg, strings.Join(supportedBrowsers, ",")))
		}
	}

	if !slices.Contains(supportedChannels, optChannel) {
		errMsg = append(errMsg, fmt.Sprintf("channel %s is invalid, can use %s", optChannel, strings.Join(supportedChannels, ",")))
	}

	if !slices.Contains(supportedPlatforms, optPlatform) {
		return fmt.Errorf("platform %s is invalid, can use %s", optPlatform, strings.Join(supportedPlatforms, ","))
	}

	if len(errMsg) > 0 {
		return errors.New(strings.Join(errMsg, "\n"))
	}

	return nil
}

func getDocument(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func fetchBody(ctx context.Context, url string) (*bytes.Reader, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", res.Status)
	}

	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(res.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func saveToFile(ctx context.Context, reader io.Reader, saveFilename string, mode fs.FileMode) error {
	// saveFilename が存在していたら、上書きするログメッセージを出力する
	if _, err := os.Stat(saveFilename); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		slog.InfoContext(ctx, "overwrite file", slog.String("filename", saveFilename))
	}

	f, err := os.OpenFile(saveFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()

	bufW := bufio.NewWriter(f)
	defer bufW.Flush()

	_, err = bufW.ReadFrom(reader)
	if err != nil {
		return err
	}

	return nil
}

func fetchZIP(ctx context.Context, url, targetFilename, saveFilename string) (*time.Time, error) {
	buf, err := fetchBody(ctx, url)
	if err != nil {
		return nil, err
	}

	zipRd, err := zip.NewReader(buf, int64(buf.Len()))
	if err != nil {
		return nil, err
	}

	var zipFile *zip.File

	for _, f := range zipRd.File {
		if filepath.Base(f.Name) == targetFilename {
			zipFile = f

			break
		}
	}

	if zipFile == nil {
		return nil, fmt.Errorf("file not found: %s", targetFilename)
	}

	modTime := zipFile.Modified

	slog.DebugContext(ctx, "found file", slog.Group(
		"found",
		slog.String("filename", zipFile.Name),
		slog.Time("modTime", modTime),
	))

	rc, err := zipFile.Open()
	if err != nil {
		return nil, err
	}

	defer rc.Close()

	err = saveToFile(ctx, rc, saveFilename, zipFile.Mode())
	if err != nil {
		return nil, err
	}

	return &modTime, nil
}

func fetchTGZ(ctx context.Context, url, targetFilename, saveFilename string) (*time.Time, error) {
	slog.DebugContext(ctx, "fetchTGZ", slog.Group(
		"target",
		slog.String("url", url),
		slog.String("targetFilename", targetFilename),
		slog.String("saveFilename", saveFilename),
	))

	buf, err := fetchBody(ctx, url)
	if err != nil {
		return nil, err
	}

	gzRd, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gzRd.Close()

	tr := tar.NewReader(gzRd)

	var tarHeader *tar.Header
	for {
		th, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		if filepath.Base(th.Name) == targetFilename {
			tarHeader = th
			break
		}
	}

	if tarHeader == nil {
		return nil, fmt.Errorf("file %s is not found", targetFilename)
	}

	modTime := tarHeader.ModTime

	slog.DebugContext(ctx, "found file", slog.Group(
		"found",
		slog.String("filename", tarHeader.Name),
		slog.Time("modTime", modTime),
	))

	err = saveToFile(ctx, tr, saveFilename, tarHeader.FileInfo().Mode())
	if err != nil {
		return nil, err
	}

	return &modTime, nil
}

func fetchAndSaveFile(ctx context.Context, fileURL, target, savePath, drvName string) error {
	var saveFilename string
	if len(drvName) > 0 {
		saveFilename = filepath.Join(savePath, drvName)
	} else {
		saveFilename = filepath.Join(savePath, target)
	}

	slog.DebugContext(ctx, "fetchAndSaveFile", slog.Group(
		"target",
		slog.String("fileURL", fileURL),
		slog.String("saveFilename", saveFilename),
	))

	var err error
	var modTm *time.Time
	switch path.Ext(fileURL) {
	case ".gz", ".tgz":
		modTm, err = fetchTGZ(ctx, fileURL, target, saveFilename)

	case ".zip":
		modTm, err = fetchZIP(ctx, fileURL, target, saveFilename)

	default:
		err = fmt.Errorf("extension %s is not a valid file", path.Ext(fileURL))
	}
	if err != nil {
		return err
	}

	err = os.Chtimes(saveFilename, *modTm, *modTm)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "saved", slog.Group(
		"result",
		slog.String("fileUrl", fileURL),
		slog.String("saveFilename", saveFilename),
		slog.Time("modTime", *modTm),
	))

	return nil
}

func run(ctx context.Context, args []string) error {
	err := validateOpts(args)
	if err != nil {
		return err
	}

	savePath := optSavePath
	if len(savePath) == 0 {
		savePath, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	for _, arg := range args {
		slog.InfoContext(ctx, "start download", slog.Group(
			"target",
			slog.String("browser", arg),
			slog.String("platform", optPlatform),
			slog.String("channel", optChannel),
		))

		switch arg {
		case "chrome":
			err = downloadChromeDriver(ctx, optChannel, optPlatform, savePath, optDriverName)
			if err != nil {
				return err
			}

		case "edge":
			err = downloadEdgeDriver(ctx, optChannel, optPlatform, savePath, optDriverName)
			if err != nil {
				return err
			}

		case "firefox":
			err = downloadGeckoDriver(ctx, optChannel, optPlatform, savePath, optDriverName)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid browser: %s", arg)
		}
	}

	return nil
}

func init() {
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.StringVar(&optSavePath, "savepath", "", "save path")
	flag.StringVar(&optPlatform, "platform", strings.Join([]string{
		runtime.GOOS,
		runtime.GOARCH,
	}, "-"), "platform")
	flag.StringVar(&optChannel, "channel", "stable", "channel")
	flag.StringVar(&optDriverName, "drivername", "", "driver name")
}

func main() {
	v := flag.Bool("v", false, "show version")
	flag.Parse()

	if *v {
		_, _ = fmt.Fprintln(os.Stderr, verString())
		os.Exit(0)
	}

	var lvl slog.Level
	if debug {
		lvl = slog.LevelDebug
	} else {
		lvl = slog.LevelInfo
	}

	slog.SetDefault(slog.New(clog.New(
		clog.WithLevel(lvl),
		clog.WithSource(debug),
		clog.WithWriter(os.Stderr),
		clog.WithColor(true),
		clog.WithTimeFmt(time.RFC3339),
		clog.WithPrinter(clog.LinearPrinter),
	)))

	slog.Debug("debug mode")

	httpClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
		Timeout: 3 * time.Minute,
	}

	ctx := context.Background()

	err := run(ctx, flag.Args())
	if err != nil {
		slog.ErrorContext(ctx, "処理に失敗", slog.Any("err", err))
		os.Exit(1)
	}
}
