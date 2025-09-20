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
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
	
	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

// WindowsPreviewGenerator はWindows環境でのプレビュー生成実装
type WindowsPreviewGenerator struct {
	initialized bool
	powershellGen *PowerShellPreviewGenerator
}

// NewWindowsPreviewGenerator は新しいWindowsプレビュージェネレータを作成
func NewWindowsPreviewGenerator() *WindowsPreviewGenerator {
	return &WindowsPreviewGenerator{
		powershellGen: NewPowerShellPreviewGenerator(),
	}
}

// 必要なWindows API定義
var (
	shell32 = syscall.NewLazyDLL("shell32.dll")
	ole32   = syscall.NewLazyDLL("ole32.dll")
	gdi32   = syscall.NewLazyDLL("gdi32.dll")
	
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
	procDeleteObject                = gdi32.NewProc("DeleteObject")
)

// IShellItemImageFactory GUID
var (
	IID_IShellItemImageFactory = &windows.GUID{
		Data1: 0xbcc18b79,
		Data2: 0xba16,
		Data3: 0x442f,
		Data4: [8]byte{0x80, 0xc4, 0x8a, 0x59, 0xc3, 0x0c, 0x46, 0x3b},
	}
)

// SIZE 構造体
type SIZE struct {
	CX int32
	CY int32
}

// SIIGBF フラグ
const (
	SIIGBF_RESIZETOFIT   = 0x00000000
	SIIGBF_BIGGERSIZEOK  = 0x00000001
	SIIGBF_MEMORYONLY    = 0x00000002
	SIIGBF_ICONONLY      = 0x00000004
	SIIGBF_THUMBNAILONLY = 0x00000008
	SIIGBF_INCACHEONLY   = 0x00000010
)

// IShellItemImageFactory インターフェース
type IShellItemImageFactory struct {
	vtbl *IShellItemImageFactoryVtbl
}

type IShellItemImageFactoryVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetImage       uintptr
}

// GeneratePreview はWindowsのシェル機能を使用してプレビューを生成
func (g *WindowsPreviewGenerator) GeneratePreview(filePath string, width, height int) (image.Image, error) {
	// OSスレッドを固定（COMのSTAモデルに必要）
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	
	fmt.Printf("WindowsPreviewGenerator: Starting preview generation for %s\n", filePath)
	fmt.Printf("WindowsPreviewGenerator: Thread locked for COM STA\n")
	
	// ファイルの存在確認
	if _, err := os.Stat(filePath); err != nil {
		fmt.Printf("WindowsPreviewGenerator: File not found: %v\n", err)
		return nil, fmt.Errorf("file not found: %w", err)
	}
	
	// COM初期化（STAモデル）- 毎回初期化と解放を行う
	fmt.Printf("WindowsPreviewGenerator: Initializing COM with STA model\n")
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err != nil {
		fmt.Printf("WindowsPreviewGenerator: COM initialization failed: %v\n", err)
		return nil, fmt.Errorf("COM initialization failed: %w", err)
	}
	defer ole.CoUninitialize()
	
	// 絶対パスに変換
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Printf("WindowsPreviewGenerator: Failed to get absolute path: %v\n", err)
		return nil, err
	}
	fmt.Printf("WindowsPreviewGenerator: Using absolute path: %s\n", absPath)
	
	// IShellItemImageFactoryを使用してプレビュー生成を試みる
	fmt.Printf("WindowsPreviewGenerator: Attempting Shell API preview for extension: %s\n", filepath.Ext(filePath))
	img, err := g.getShellItemPreview(absPath, width, height)
	if err == nil {
		fmt.Printf("WindowsPreviewGenerator: Shell API preview successful\n")
		return img, nil
	}
	fmt.Printf("WindowsPreviewGenerator: Shell API failed: %v\n", err)
	
	// フォールバック: PowerShellまたはファイルタイプに応じた処理
	ext := strings.ToLower(filepath.Ext(filePath))
	fmt.Printf("WindowsPreviewGenerator: Falling back to alternative method for %s\n", ext)
	
	// PDFの場合はPowerShellジェネレータを試す
	if ext == ".pdf" {
		fmt.Printf("WindowsPreviewGenerator: Trying PowerShell generator for PDF\n")
		img, err := g.powershellGen.GeneratePreview(absPath, width, height)
		if err == nil {
			fmt.Printf("WindowsPreviewGenerator: PowerShell generator succeeded\n")
			return img, nil
		}
		fmt.Printf("WindowsPreviewGenerator: PowerShell generator failed: %v\n", err)
	}
	
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		return g.loadImageFile(absPath)
	case ".txt", ".log", ".md":
		return g.generateTextPreview(absPath)
	default:
		// デフォルトアイコンを生成
		fmt.Printf("WindowsPreviewGenerator: Generating default icon for %s\n", ext)
		return g.generateDefaultIcon(ext)
	}
}

// getShellItemPreview はIShellItemImageFactoryを使用してプレビューを取得
func (g *WindowsPreviewGenerator) getShellItemPreview(filePath string, width, height int) (image.Image, error) {
	fmt.Printf("getShellItemPreview: Starting for %s\n", filePath)
	
	// UTF16に変換
	pathPtr, err := syscall.UTF16PtrFromString(filePath)
	if err != nil {
		fmt.Printf("getShellItemPreview: UTF16 conversion failed: %v\n", err)
		return nil, err
	}
	
	// IShellItemを作成
	var shellItem uintptr
	hr, _, _ := procSHCreateItemFromParsingName.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(IID_IShellItemImageFactory)),
		uintptr(unsafe.Pointer(&shellItem)),
	)
	
	if hr != 0 {
		fmt.Printf("getShellItemPreview: SHCreateItemFromParsingName failed: 0x%x\n", hr)
		return nil, fmt.Errorf("SHCreateItemFromParsingName failed: 0x%x", hr)
	}
	fmt.Printf("getShellItemPreview: IShellItem created successfully\n")
	
	// IShellItemImageFactoryにキャスト
	factory := (*IShellItemImageFactory)(unsafe.Pointer(shellItem))
	defer g.releaseShellItem(factory)
	
	// サイズ設定 - 256以下に制限
	if width > 256 {
		width = 256
	}
	if height > 256 {
		height = 256
	}
	size := SIZE{CX: int32(width), CY: int32(height)}
	// 複数のフラグパターンを試す
	flagPatterns := []struct {
		flags uint
		name  string
	}{
		{SIIGBF_INCACHEONLY | SIIGBF_RESIZETOFIT, "INCACHEONLY"},
		{SIIGBF_THUMBNAILONLY | SIIGBF_RESIZETOFIT, "THUMBNAILONLY"},
		{SIIGBF_RESIZETOFIT, "RESIZETOFIT_ONLY"},
		{SIIGBF_MEMORYONLY | SIIGBF_RESIZETOFIT, "MEMORYONLY"},
		{SIIGBF_BIGGERSIZEOK | SIIGBF_RESIZETOFIT, "BIGGERSIZEOK"},
		{0, "NO_FLAGS"},
	}
	
	var hBitmap windows.Handle
	var lastHR uintptr
	var success bool
	
	// 各フラグパターンを順番に試す
	for i, pattern := range flagPatterns {
		fmt.Printf("getShellItemPreview: Trying pattern %d/%d - %s (flags: 0x%x)\n", i+1, len(flagPatterns), pattern.name, pattern.flags)
		
		hr, _, _ := syscall.Syscall6(
			(*factory.vtbl).GetImage,
			4,
			uintptr(unsafe.Pointer(factory)),
			uintptr(unsafe.Pointer(&size)),
			uintptr(pattern.flags),
			uintptr(unsafe.Pointer(&hBitmap)),
			0,
			0,
		)
		lastHR = hr
		
		if hr == 0 {
			fmt.Printf("getShellItemPreview: SUCCESS with %s pattern!\n", pattern.name)
			success = true
			break
		} else {
			fmt.Printf("getShellItemPreview: Pattern %s failed: 0x%x\n", pattern.name, hr)
		}
	}
	
	if !success {
		return nil, fmt.Errorf("GetImage failed with all patterns, last error: 0x%x", lastHR)
	}
	fmt.Printf("getShellItemPreview: Got HBITMAP handle: %v\n", hBitmap)
	defer deleteObject(hBitmap)
	
	// HBITMAPをimage.Imageに変換
	return g.convertHBitmapToImage(hBitmap)
}

// deleteObject はGDIオブジェクトを削除
func deleteObject(hObject windows.Handle) {
	procDeleteObject.Call(uintptr(hObject))
}

// releaseShellItem はCOMオブジェクトを解放
func (g *WindowsPreviewGenerator) releaseShellItem(factory *IShellItemImageFactory) {
	if factory != nil {
		syscall.Syscall((*factory.vtbl).Release, 1, uintptr(unsafe.Pointer(factory)), 0, 0)
	}
}

// convertHBitmapToImage はHBITMAPをimage.Imageに変換
func (g *WindowsPreviewGenerator) convertHBitmapToImage(hBitmap windows.Handle) (image.Image, error) {
	// デフォルトの画像を返す（一時的な実装）
	// TODO: 実際のHBITMAP変換を実装
	fmt.Printf("convertHBitmapToImage: Returning default image (TODO: implement HBITMAP conversion)\n")
	
	// とりあえず空の画像を作成
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))
	
	// 簡単な色で塗りつぶし（デバッグ用）
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			img.Set(x, y, image.White)
		}
	}
	
	return img, nil
}

// loadImageFile は画像ファイルを直接読み込む
func (g *WindowsPreviewGenerator) loadImageFile(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	img, _, err := image.Decode(file)
	return img, err
}

// generateTextPreview はテキストファイルのプレビューを生成
func (g *WindowsPreviewGenerator) generateTextPreview(filePath string) (image.Image, error) {
	// テキストファイルの先頭部分を読み込んで画像化
	// 実装は省略
	return image.NewRGBA(image.Rect(0, 0, 300, 300)), nil
}

// generateDefaultIcon はデフォルトアイコンを生成
func (g *WindowsPreviewGenerator) generateDefaultIcon(ext string) (image.Image, error) {
	// 拡張子に応じたデフォルトアイコンを生成
	// 実装は省略
	return image.NewRGBA(image.Rect(0, 0, 64, 64)), nil
}

// GeneratePreviewBytes はプレビュー画像をバイト配列として生成
func (g *WindowsPreviewGenerator) GeneratePreviewBytes(filePath string, width, height int, format string) ([]byte, string, error) {
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

// Cleanup はリソースをクリーンアップ
func (g *WindowsPreviewGenerator) Cleanup() {
	if g.initialized {
		ole.CoUninitialize()
		g.initialized = false
	}
}