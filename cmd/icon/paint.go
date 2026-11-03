package main

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/vector"
)

// size 是画布边长，也是输出图片的像素数。
const size = 1024

// canvas 是一张画布，连同往上填形状用的光栅器。
type canvas struct {
	img *image.RGBA
	z   *vector.Rasterizer
}

func newCanvas() *canvas {
	return &canvas{
		img: image.NewRGBA(image.Rect(0, 0, size, size)),
		z:   vector.NewRasterizer(size, size),
	}
}

// trace 把轮廓加进光栅器。逆时针的轮廓与顺时针相抵，可用于挖孔。
func (c *canvas) trace(b builder, hole bool) {
	c.z.MoveTo(float32(b.start.x), float32(b.start.y))
	for _, s := range b.seg {
		a := s.args
		switch s.op {
		case 'L':
			c.z.LineTo(float32(a[0]), float32(a[1]))
		case 'Q':
			c.z.QuadTo(float32(a[0]), float32(a[1]), float32(a[2]), float32(a[3]))
		case 'C':
			c.z.CubeTo(float32(a[0]), float32(a[1]), float32(a[2]), float32(a[3]), float32(a[4]), float32(a[5]))
		}
	}
	c.z.ClosePath()
	_ = hole
}

// fill 用 src 填充轮廓。
func (c *canvas) fill(src image.Image, b builder) {
	c.z.Reset(size, size)
	c.trace(b, false)
	c.z.Draw(c.img, c.img.Bounds(), src, image.Point{})
}

// mask 把轮廓画成覆盖率遮罩。
func (c *canvas) mask(b builder) *image.Alpha {
	m := image.NewAlpha(c.img.Bounds())
	c.z.Reset(size, size)
	c.trace(b, false)
	c.z.Draw(m, m.Bounds(), image.NewUniform(color.Alpha{255}), image.Point{})
	return m
}

// shade 画轮廓平移 (dx, dy) 后投下的软影，模糊半径为 r。
func (c *canvas) shade(col color.NRGBA, dx, dy float64, r int, b builder) {
	m := c.mask(b.scale(1, 1, dx, dy))
	blur(m, r)
	for i, v := range m.Pix {
		m.Pix[i] = uint8(int(v) * int(col.A) / 255)
	}
	src := image.NewUniform(color.RGBA{col.R, col.G, col.B, 255})
	draw.DrawMask(c.img, c.img.Bounds(), src, image.Point{}, m, image.Point{}, draw.Over)
}

// blur 用三遍盒式模糊近似半径为 r 的高斯模糊。
func blur(a *image.Alpha, r int) {
	w, h := a.Bounds().Dx(), a.Bounds().Dy()
	src, tmp := make([]float32, w*h), make([]float32, w*h)
	for i, v := range a.Pix {
		src[i] = float32(v)
	}
	for range 3 {
		box(tmp, src, w, r, 1, h, w) // 横向
		box(src, tmp, h, r, w, w, 1) // 纵向
	}
	for i, v := range src {
		a.Pix[i] = uint8(v + 0.5)
	}
}

// box 对 lines 条线各做一遍宽 2r+1 的均值：每条线有 n 个样本，样本间隔 stride，
// 线与线间隔 lineStride；越界取边缘的值。
func box(dst, src []float32, n, r, stride, lines, lineStride int) {
	width := float32(2*r + 1)
	for l := range lines {
		base := l * lineStride
		at := func(i int) float32 { return src[base+min(max(i, 0), n-1)*stride] }
		var sum float32
		for i := -r; i <= r; i++ {
			sum += at(i)
		}
		for i := range n {
			dst[base+i*stride] = sum / width
			sum += at(i+r+1) - at(i-r)
		}
	}
}

// pixels 是由函数逐点给出颜色的图，用作渐变的来源。
type pixels func(x, y int) color.Color

func (pixels) ColorModel() color.Model   { return color.RGBAModel }
func (pixels) Bounds() image.Rectangle   { return image.Rect(0, 0, size, size) }
func (p pixels) At(x, y int) color.Color { return p(x, y) }

// plate 画白色底板：落在背景上的软影与顶部一层极淡的柔光。底板 824 见方，
// 四周各留 100 像素，是 macOS 图标的规格。
func (c *canvas) plate(fill color.RGBA) {
	body := squirclePath(size/2, size/2, 412)
	c.shade(color.NRGBA{0x2B, 0x2B, 0x3A, 46}, 0, 14, 10, body)
	c.fill(image.NewUniform(fill), body)
	c.fill(pixels(func(_, y int) color.Color {
		return color.NRGBA{255, 255, 255, uint8(56 * max(0, 1-float64(y-100)/494))}
	}), body)
}

// mark 画小写 t：先投一层与主体同色的软影，再填充主题色。图形按包围盒严格
// 居中于底板，不另作光学偏移。
func (c *canvas) mark(fill color.RGBA) {
	m := centered(markPath(), size/2, size/2)
	c.shade(color.NRGBA{fill.R, fill.G, fill.B, 40}, 0, 12, 12, m)
	c.fill(image.NewUniform(fill), m)
}

// centered 平移轮廓，使包围盒中心落在 (cx, cy)。
func centered(b builder, cx, cy float64) builder {
	minX, minY, maxX, maxY := b.bounds()
	return b.scale(1, 1, cx-(minX+maxX)/2, cy-(minY+maxY)/2)
}

// squirclePath 把超椭圆轮廓转成可填充的路径。
func squirclePath(cx, cy, r float64) builder {
	p := squircle(cx, cy, r)
	b := builder{start: p[0]}
	for _, v := range p[1:] {
		b.line(v.x, v.y)
	}
	return b
}
