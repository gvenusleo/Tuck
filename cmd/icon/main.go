// Command icon 画出 Tuck 的应用图标与菜单栏图标。
//
//	go run ./cmd/icon
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// 图标只有一套配色：白色底板配主题色的小写 t。
const (
	plateColor = 0xFFFFFF
	markColor  = 0x2177B8
)

func rgb(v uint32) color.RGBA { return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255} }

func main() {
	out := flag.String("o", "resources", "输出目录")
	tray := flag.String("tray", "internal/app/resources/tray.png", "菜单栏图标的路径")
	flag.Parse()

	if err := write(filepath.Join(*out, "icon.png")); err != nil {
		fail(err)
	}
	if err := writeTray(*tray); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "icon:", err)
	os.Exit(1)
}

// write 画出应用图标并写到 path。
func write(path string) error {
	c := newCanvas()
	c.plate(rgb(plateColor))
	c.mark(rgb(markColor))
	return save(path, c.img)
}

func save(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
