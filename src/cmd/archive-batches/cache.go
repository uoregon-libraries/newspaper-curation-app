package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const cacheVersion = 1

// cacheEntry records that a given batch's archive passed full BagIt
// validation. The tagmanifest fingerprint lets us detect archive drift: if the
// bag has changed since validation, the fingerprint won't match and we'll
// re-validate.
type cacheEntry struct {
	BatchID        int64     `json:"batch_id"`
	ArchivePath    string    `json:"archive_path"`
	TagFingerprint string    `json:"tagmanifest_fingerprint"`
	ValidatedAt    time.Time `json:"validated_at"`
}

// cacheFile is the on-disk format for the validation cache.
type cacheFile struct {
	Version int                    `json:"version"`
	Entries map[string]*cacheEntry `json:"entries"`
}

// validationCache wraps the on-disk cache with load/save/lookup helpers. A
// zero value is not usable; construct with loadCache.
type validationCache struct {
	path string
	file *cacheFile
}

// defaultCachePath returns the path we use when the user doesn't pass
// --cache-file. It follows the XDG Base Directory spec, falling back to
// ~/.cache when XDG_CACHE_HOME is unset.
func defaultCachePath() (string, error) {
	var base = os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		var home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "nca", "archive-batches-validation.json"), nil
}

// loadCache reads the cache file at path. A missing file is fine; a corrupt
// file logs a warning via the returned note and starts fresh (we'll overwrite
// on next save). Other I/O errors are returned.
func loadCache(path string) (*validationCache, string, error) {
	var c = &validationCache{path: path, file: &cacheFile{Version: cacheVersion, Entries: map[string]*cacheEntry{}}}

	var data, err = os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("reading cache %q: %w", path, err)
	}

	var loaded cacheFile
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		return c, fmt.Sprintf("cache file %q is unreadable (%s); starting fresh", path, err), nil
	}
	if loaded.Version != cacheVersion {
		return c, fmt.Sprintf("cache file %q is version %d, expected %d; starting fresh", path, loaded.Version, cacheVersion), nil
	}
	if loaded.Entries == nil {
		loaded.Entries = map[string]*cacheEntry{}
	}
	c.file = &loaded
	return c, "", nil
}

// lookup returns the cached entry for a batch FullName, or nil if none exists.
func (c *validationCache) lookup(fullName string) *cacheEntry {
	return c.file.Entries[fullName]
}

// record adds or replaces the cache entry for the given batch and immediately
// persists the cache to disk so partial runs aren't lost.
func (c *validationCache) record(fullName string, entry *cacheEntry) error {
	c.file.Entries[fullName] = entry
	return c.save()
}

// save writes the cache to disk, creating parent directories as needed.
func (c *validationCache) save() error {
	var err = os.MkdirAll(filepath.Dir(c.path), 0755)
	if err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	var data []byte
	data, err = json.MarshalIndent(c.file, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding cache: %w", err)
	}

	// Write via a temp file + rename so a crash mid-write can't corrupt the cache.
	var tmp = c.path + ".tmp"
	err = os.WriteFile(tmp, data, 0644)
	if err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}
	err = os.Rename(tmp, c.path)
	if err != nil {
		return fmt.Errorf("renaming cache: %w", err)
	}
	return nil
}
