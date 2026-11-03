package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"golang.org/x/image/vector"
)

// 菜单栏图标是 16 点见方、32 像素的模板图标：黑色加透明底，macOS 按菜单栏的
// 明暗给它上色。菜单栏里的图形比应用图标更需要存在感，因此单独放大，笔画粗细
// 仍与应用图标一致。
const (
	traySize   = 32
	traySupers = 4  // 超采样倍数，用来抗锯齿
	trayHeight = 26 // t 在 32 像素画布里的高度
)

func writeTray(path string) error {
	const n = traySize * traySupers
	z := vector.NewRasterizer(n, n)
	m := markPath()
	minX, minY, maxX, maxY := m.bounds()
	s := float64(trayHeight*traySupers) / (maxY - minY)
	dx := (float64(n) - (maxX-minX)*s) / 2
	dy := (float64(n) - (maxY-minY)*s) / 2
	fitted := m.scale(s, s, dx-minX*s, dy-minY*s)

	z.MoveTo(float32(fitted.start.x), float32(fitted.start.y))
	for _, seg := range fitted.seg {
		a := seg.args
		switch seg.op {
		case 'L':
			z.LineTo(float32(a[0]), float32(a[1]))
		case 'Q':
			z.QuadTo(float32(a[0]), float32(a[1]), float32(a[2]), float32(a[3]))
		case 'C':
			z.CubeTo(float32(a[0]), float32(a[1]), float32(a[2]), float32(a[3]), float32(a[4]), float32(a[5]))
		}
	}
	z.ClosePath()

	big := image.NewAlpha(image.Rect(0, 0, n, n))
	z.Draw(big, big.Bounds(), image.NewUniform(color.Alpha{255}), image.Point{})

	img := image.NewNRGBA(image.Rect(0, 0, traySize, traySize))
	for y := range traySize {
		for x := range traySize {
			var sum int
			for sy := range traySupers {
				for sx := range traySupers {
					sum += int(big.AlphaAt(x*traySupers+sx, y*traySupers+sy).A)
				}
			}
			img.SetNRGBA(x, y, color.NRGBA{A: uint8(sum / (traySupers * traySupers))})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
