package acctest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/testcontainers/testcontainers-go"
)

// CustomComponent is a custom integration (one that HACS would install) that the container gets
// in /config/custom_components before Home Assistant starts. It is downloaded as a .tar.gz
// release archive and checked against its SHA-256.
type CustomComponent struct {
	// Domain is the integration's domain, the name of its directory in custom_components.
	Domain string
	// URL is the release archive.
	URL string
	// SHA256 is the archive's checksum, in hex.
	SHA256 string
	// Dir is the integration's directory inside the archive.
	Dir string
}

// Scheduler is the Scheduler integration (nielsfaber/scheduler-component, Apache-2.0), the
// backend of the scheduler card. The provider supports its 3.3 releases.
var Scheduler = CustomComponent{
	Domain: "scheduler",
	URL:    "https://github.com/nielsfaber/scheduler-component/archive/refs/tags/v3.3.9.tar.gz",
	SHA256: "6dc3fec716c03e44f924fb9a6fdffbd72dd9bbce90215172378dbf081532903e",
	Dir:    "scheduler-component-3.3.9/custom_components/scheduler",
}

// customComponents are installed into the package's container. Set them with Install.
var customComponents []CustomComponent

// Install makes the package's Home Assistant start with the custom integrations cs. Call it from
// TestMain, before Main. It only installs their files: a test sets the integration up itself,
// e.g. with a config flow. Packages that call it get a container of their own, like every package.
func Install(cs ...CustomComponent) {
	customComponents = append(customComponents, cs...)
}

// customComponentFiles downloads the custom components and returns their files, for the container
// request.
func customComponentFiles(ctx context.Context) ([]testcontainers.ContainerFile, error) {
	var files []testcontainers.ContainerFile
	for _, c := range customComponents {
		fs, err := c.files(ctx)
		if err != nil {
			return nil, fmt.Errorf("custom component %s: %w", c.Domain, err)
		}
		files = append(files, fs...)
	}
	return files, nil
}

func (c CustomComponent) files(ctx context.Context) ([]testcontainers.ContainerFile, error) {
	archive, err := download(ctx, c.URL)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != c.SHA256 {
		return nil, fmt.Errorf("%s has SHA-256 %s, want %s", c.URL, got, c.SHA256)
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	prefix := strings.TrimSuffix(c.Dir, "/") + "/"
	var files []testcontainers.ContainerFile
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		rel, ok := strings.CutPrefix(h.Name, prefix)
		if !ok || h.Typeflag != tar.TypeReg {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		files = append(files, testcontainers.ContainerFile{
			Reader:            bytes.NewReader(body),
			ContainerFilePath: path.Join("/config/custom_components", c.Domain, rel),
			FileMode:          0o644,
		})
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s has no files under %s", c.URL, c.Dir)
	}
	return files, nil
}

func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
