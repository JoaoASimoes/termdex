package dex

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Sprite downloads (or reads from the cache) the front sprite of a Pokémon.
func (d *Client) Sprite(name, url string) (image.Image, error) {
	if url == "" {
		return nil, ErrNotFound
	}
	dir := filepath.Join(d.cacheDir, "sprites")
	path := filepath.Join(dir, strings.ReplaceAll(name, "~", "_")+".png")
	raw, err := os.ReadFile(path)
	if err != nil || d.refresh {
		raw, err = d.getBytes(url)
		if err != nil {
			return nil, err
		}
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(path, raw, 0o644)
	}
	return png.Decode(bytes.NewReader(raw))
}

func (d *Client) getBytes(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "pokedex-cli/1.0")
	d.sem <- struct{}{}
	defer func() { <-d.sem }()
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sprite request returned %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
