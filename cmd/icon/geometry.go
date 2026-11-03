package main

import "math"

// pt 是画布上的一点，坐标单位为像素，原点在左上角。
type pt struct{ x, y float64 }

// poly 是一个闭合的多边形。
type poly []pt

// squircle 是以 (cx, cy) 为中心、半边长 r 的连续圆角方块（超椭圆），与 macOS
// 图标的轮廓相近。
func squircle(cx, cy, r float64) poly {
	const steps = 1440
	p := make(poly, steps)
	for i := range p {
		s, c := math.Sincos(2 * math.Pi * float64(i) / steps)
		p[i] = pt{
			cx + r*math.Copysign(math.Pow(math.Abs(c), 0.4), c),
			cy + r*math.Copysign(math.Pow(math.Abs(s), 0.4), s),
		}
	}
	return p
}

// path 记录一条轮廓，保留曲线段，交给光栅器自行细分。
type path []segment

// segment 是一段轮廓：起点已由上段终点给出。
type segment struct {
	op   byte // 'L' 直线、'Q' 二次曲线、'C' 三次曲线
	args [6]float64
}

// builder 保存轮廓的起点和曲线段。
type builder struct {
	start pt
	seg   path
}

func (b *builder) line(x, y float64) {
	b.seg = append(b.seg, segment{op: 'L', args: [6]float64{x, y}})
}

func (b *builder) quad(cx, cy, x, y float64) {
	b.seg = append(b.seg, segment{op: 'Q', args: [6]float64{cx, cy, x, y}})
}

func (b *builder) cube(c1x, c1y, c2x, c2y, x, y float64) {
	b.seg = append(b.seg, segment{op: 'C', args: [6]float64{c1x, c1y, c2x, c2y, x, y}})
}

// scale 返回缩放并平移到目标框内的轮廓。
func (b builder) scale(sx, sy, dx, dy float64) builder {
	move := func(x, y float64) (float64, float64) { return x*sx + dx, y*sy + dy }
	startX, startY := move(b.start.x, b.start.y)
	out := builder{start: pt{startX, startY}}
	for _, s := range b.seg {
		var q segment
		q.op = s.op
		n := 1
		switch s.op {
		case 'Q':
			n = 2
		case 'C':
			n = 3
		}
		for i := 0; i < n; i++ {
			q.args[2*i], q.args[2*i+1] = move(s.args[2*i], s.args[2*i+1])
		}
		out.seg = append(out.seg, q)
	}
	return out
}

// bounds 返回轮廓控制点的包围盒，用于把图形放进画布。
func (b builder) bounds() (minX, minY, maxX, maxY float64) {
	minX, minY, maxX, maxY = b.start.x, b.start.y, b.start.x, b.start.y
	for _, s := range b.seg {
		n := 1
		switch s.op {
		case 'Q':
			n = 2
		case 'C':
			n = 3
		}
		for i := 0; i < n; i++ {
			x, y := s.args[2*i], s.args[2*i+1]
			minX, minY = math.Min(minX, x), math.Min(minY, y)
			maxX, maxY = math.Max(maxX, x), math.Max(maxY, y)
		}
	}
	return minX, minY, maxX, maxY
}
