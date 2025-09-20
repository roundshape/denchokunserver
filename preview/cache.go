package preview

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cache はプレビュー画像のキャッシュを管理
type Cache struct {
	baseDir string
	mutex   sync.RWMutex
	items   map[string]*CacheItem
}

// CacheItem はキャッシュアイテムの情報
type CacheItem struct {
	Path      string
	CreatedAt time.Time
	Size      int64
}

// NewCache は新しいキャッシュマネージャーを作成
func NewCache(baseDir string) (*Cache, error) {
	// キャッシュディレクトリの作成
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	cache := &Cache{
		baseDir: baseDir,
		items:   make(map[string]*CacheItem),
	}
	
	// 既存のキャッシュファイルをスキャン
	cache.scanExistingCache()
	
	// 定期的なクリーンアップを開始
	go cache.startCleanupRoutine()
	
	return cache, nil
}

// generateCacheKey はキャッシュキーを生成
func (c *Cache) generateCacheKey(filePath string, width, height, page int) string {
	data := fmt.Sprintf("%s_%d_%d_%d", filePath, width, height, page)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Get はキャッシュから画像データを取得
func (c *Cache) Get(filePath string, width, height, page int) ([]byte, bool) {
	key := c.generateCacheKey(filePath, width, height, page)
	
	c.mutex.RLock()
	item, exists := c.items[key]
	c.mutex.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	// ファイルの更新時刻をチェック
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, false
	}
	
	// 元ファイルがキャッシュより新しい場合は無効
	if fileInfo.ModTime().After(item.CreatedAt) {
		c.Delete(key)
		return nil, false
	}
	
	// キャッシュファイルを読み込み
	data, err := os.ReadFile(item.Path)
	if err != nil {
		c.Delete(key)
		return nil, false
	}
	
	return data, true
}

// Put はキャッシュに画像データを保存
func (c *Cache) Put(filePath string, width, height, page int, data []byte) error {
	key := c.generateCacheKey(filePath, width, height, page)
	
	// キャッシュファイルのパスを生成
	cachePath := filepath.Join(c.baseDir, fmt.Sprintf("%s.cache", key))
	
	// ファイルに書き込み
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}
	
	// キャッシュ情報を更新
	c.mutex.Lock()
	c.items[key] = &CacheItem{
		Path:      cachePath,
		CreatedAt: time.Now(),
		Size:      int64(len(data)),
	}
	c.mutex.Unlock()
	
	return nil
}

// Delete はキャッシュからアイテムを削除
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if item, exists := c.items[key]; exists {
		os.Remove(item.Path)
		delete(c.items, key)
	}
}

// scanExistingCache は既存のキャッシュファイルをスキャン
func (c *Cache) scanExistingCache() {
	pattern := filepath.Join(c.baseDir, "*.cache")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		key := filepath.Base(file)
		key = key[:len(key)-6] // .cache を除去
		
		c.items[key] = &CacheItem{
			Path:      file,
			CreatedAt: info.ModTime(),
			Size:      info.Size(),
		}
	}
}

// startCleanupRoutine は定期的なクリーンアップを実行
func (c *Cache) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup は古いキャッシュを削除
func (c *Cache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	maxAge := 24 * time.Hour
	
	for key, item := range c.items {
		if now.Sub(item.CreatedAt) > maxAge {
			os.Remove(item.Path)
			delete(c.items, key)
		}
	}
}

// Clear はすべてのキャッシュをクリア
func (c *Cache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	for key, item := range c.items {
		if err := os.Remove(item.Path); err != nil {
			// エラーは無視して続行
		}
		delete(c.items, key)
	}
	
	return nil
}

// GetStats はキャッシュの統計情報を取得
func (c *Cache) GetStats() (count int, totalSize int64) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	count = len(c.items)
	for _, item := range c.items {
		totalSize += item.Size
	}
	
	return count, totalSize
}