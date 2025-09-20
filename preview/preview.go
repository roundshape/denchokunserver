package preview

import (
	"image"
)

// Generator はプレビュー画像を生成するインターフェース
type Generator interface {
	// GeneratePreview はファイルのプレビュー画像を生成する
	GeneratePreview(filePath string, width, height int) (image.Image, error)
	
	// GeneratePreviewBytes はプレビュー画像をバイト配列として生成する
	GeneratePreviewBytes(filePath string, width, height int, format string) ([]byte, string, error)
}

// Options はプレビュー生成のオプション
type Options struct {
	Width   int
	Height  int
	Page    int    // PDFなどのページ番号
	Quality int    // JPEG品質 (1-100)
	Format  string // 出力フォーマット (jpeg/png)
}

// DefaultOptions はデフォルトのオプション値を返す
func DefaultOptions() *Options {
	return &Options{
		Width:   300,
		Height:  300,
		Page:    1,
		Quality: 80,
		Format:  "jpeg",
	}
}