//go:build windows
// +build windows

package preview

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PowerShellPreviewGenerator はPowerShellを使用してプレビューを生成
type PowerShellPreviewGenerator struct{}

// NewPowerShellPreviewGenerator は新しいPowerShellプレビュージェネレータを作成
func NewPowerShellPreviewGenerator() *PowerShellPreviewGenerator {
	return &PowerShellPreviewGenerator{}
}

// GeneratePreview はPowerShellを使用してプレビューを生成
func (g *PowerShellPreviewGenerator) GeneratePreview(filePath string, width, height int) (image.Image, error) {
	fmt.Printf("PowerShellPreviewGenerator: Starting preview generation for %s\n", filePath)
	
	// 一時ファイル名を生成
	tempDir := os.TempDir()
	timestamp := time.Now().Format("20060102150405")
	tempImagePath := filepath.Join(tempDir, fmt.Sprintf("thumbnail_%s.jpg", timestamp))
	defer os.Remove(tempImagePath) // 後始末
	
	// 絶対パスに変換
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	fmt.Printf("PowerShellPreviewGenerator: Using temp file: %s\n", tempImagePath)
	
	// PowerShellスクリプトを作成
	script := g.createThumbnailScript(absPath, tempImagePath, width, height)
	
	// PowerShellを実行
	fmt.Printf("PowerShellPreviewGenerator: Executing PowerShell script\n")
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("PowerShell execution failed: %w\nOutput: %s", err, string(output))
	}
	
	fmt.Printf("PowerShellPreviewGenerator: PowerShell output: %s\n", string(output))
	
	// 生成された画像ファイルを確認
	if _, err := os.Stat(tempImagePath); err != nil {
		return nil, fmt.Errorf("thumbnail file not created: %w", err)
	}
	
	// 画像ファイルを読み込み
	fmt.Printf("PowerShellPreviewGenerator: Loading generated image\n")
	file, err := os.Open(tempImagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open thumbnail: %w", err)
	}
	defer file.Close()
	
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode thumbnail: %w", err)
	}
	
	fmt.Printf("PowerShellPreviewGenerator: Successfully generated preview\n")
	return img, nil
}

// createThumbnailScript はPowerShellスクリプトを作成
func (g *PowerShellPreviewGenerator) createThumbnailScript(inputPath, outputPath string, width, height int) string {
	// パスを適切にエスケープ
	escapedInput := strings.ReplaceAll(inputPath, `'`, `''`)
	escapedOutput := strings.ReplaceAll(outputPath, `'`, `''`)
	
	script := fmt.Sprintf(`
try {
    # シンプルなShell.Application COMオブジェクトを使用
    $shell = New-Object -ComObject Shell.Application
    $folder = $shell.NameSpace((Split-Path '%s'))
    $item = $folder.ParseName((Split-Path '%s' -Leaf))
    
    if ($item) {
        # FolderItemのGetThumbnailメソッドを試す
        try {
            $thumb = $item.GetThumbnail(%d, 1)
            if ($thumb) {
                # サムネイルが取得できた場合は、一時的にクリップボード経由で保存
                [System.Windows.Forms.Clipboard]::SetImage($thumb)
                $image = [System.Windows.Forms.Clipboard]::GetImage()
                if ($image) {
                    $image.Save('%s', [System.Drawing.Imaging.ImageFormat]::Jpeg)
                    Write-Output "Thumbnail saved successfully"
                    exit 0
                }
            }
        } catch {
            Write-Output "GetThumbnail method failed: $_"
        }
    }
    
    # フォールバック: ファイルアイコンを取得
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    
    $icon = [System.Drawing.Icon]::ExtractAssociatedIcon('%s')
    if ($icon) {
        $bitmap = $icon.ToBitmap()
        $resized = New-Object System.Drawing.Bitmap($bitmap, %d, %d)
        $resized.Save('%s', [System.Drawing.Imaging.ImageFormat]::Jpeg)
        $bitmap.Dispose()
        $resized.Dispose()
        $icon.Dispose()
        Write-Output "Icon thumbnail saved successfully"
    } else {
        # 最終フォールバック: 簡単なプレースホルダー画像
        $bitmap = New-Object System.Drawing.Bitmap(%d, %d)
        $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
        $graphics.Clear([System.Drawing.Color]::LightGray)
        $font = New-Object System.Drawing.Font("Arial", 16)
        $brush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::Black)
        $graphics.DrawString("PDF", $font, $brush, 10, %d)
        $graphics.Dispose()
        $bitmap.Save('%s', [System.Drawing.Imaging.ImageFormat]::Jpeg)
        $bitmap.Dispose()
        $font.Dispose()
        $brush.Dispose()
        Write-Output "Placeholder thumbnail created"
    }
    
} catch {
    Write-Error "Script failed: $_"
    exit 1
}
`, escapedInput, escapedInput, width, escapedOutput, escapedInput, width, height, escapedOutput, width, height, height/2-10, escapedOutput)

	return script
}

// GeneratePreviewBytes はプレビュー画像をバイト配列として生成
func (g *PowerShellPreviewGenerator) GeneratePreviewBytes(filePath string, width, height int, format string) ([]byte, string, error) {
	img, err := g.GeneratePreview(filePath, width, height)
	if err != nil {
		return nil, "", err
	}
	
	var buf bytes.Buffer
	var contentType string
	
	switch strings.ToLower(format) {
	case "png":
		err = png.Encode(&buf, img)
		contentType = "image/png"
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
		contentType = "image/jpeg"
	default:
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
		contentType = "image/jpeg"
	}
	
	if err != nil {
		return nil, "", err
	}
	
	return buf.Bytes(), contentType, nil
}

// Cleanup はリソースをクリーンアップ（PowerShellでは特に何もしない）
func (g *PowerShellPreviewGenerator) Cleanup() {
	// PowerShellジェネレータでは特にクリーンアップは不要
}