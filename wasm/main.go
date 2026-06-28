package main

import (
	"strconv"
	"syscall/js"
)

var (
	showBlueHeads       = true
	overlap             = 0
	background          = "linear-gradient(#ccddee, #ffffff);"
	githubLinkColor     = "rgba(107,165,184,.5)"
	githubLinkOverColor = "rgba(107,165,184,1)"
	githubLink          = "http://github.com/tidwall/digitalrain"
	level1Cols          = 40
	level2Cols          = 60
)

var lowGlyphCanvases []*GlyphCanvas
var highGlyphCanvases []*GlyphCanvas
var backgrounds []string
var index = 1

func main() {
	sheet := js.Global().Get("document").Call("createElement", "style")
	sheet.Set("innerHTML",
		`html, body { 
			padding:0; margin:0; border:0; width:100%; height:100%; overflow:hidden;
		}
		html{
			background: black;
		}`)
	js.Global().Get("document").Get("head").Call("appendChild", sheet)
	js.Global().Get("document").Set("title", "whoa")

	initDigitalRain()
	select {}
}

func initDigitalRain() {
	lowGlyphCanvases = []*GlyphCanvas{NewGlyphCanvas("#6ba5b8"), NewGlyphCanvas("#3b806d")}
	highGlyphCanvases = []*GlyphCanvas{NewGlyphCanvas("#5b95a8"), NewGlyphCanvas("#5b9b9b")}
	backgrounds = []string{"#dcedfe", "#000000"}
	rain1, err := NewDigitalRain(js.Global().Get("document").Get("body"), level2Cols, 2, 8, 0.25)
	if err != nil {
		println(err.Error())
		return
	}
	cover := js.Global().Get("document").Call("createElement", "div")
	cover.Get("style").Set("height", "100%")
	cover.Get("style").Set("width", "100%")
	cover.Get("style").Set("background-image", "radial-gradient(ellipse farthest-corner at 45px 45px , #00FFFF 0%, rgba(0, 0, 255, 0) 50%, #0000FF 95%)")
	cover.Get("style").Set("opacity", "0.18")
	cover.Get("style").Set("position", "absolute")
	js.Global().Get("document").Get("body").Call("appendChild", cover)

	resizeFn1 := js.FuncOf(func(this js.Value, args []js.Value) any {
		rain1.layout()
		return nil
	})
	js.Global().Call("addEventListener", "resize", resizeFn1)

	rain2, err := NewDigitalRain(js.Global().Get("document").Get("body"), level1Cols, 2, 12, 1.0)
	if err != nil {
		println(err.Error())
		return
	}
	resizeFn2 := js.FuncOf(func(this js.Value, args []js.Value) any {
		rain2.layout()
		return nil
	})
	js.Global().Call("addEventListener", "resize", resizeFn2)
	rain2.Clicked = func() {
		return
		// unreachable below — kept for 100% fidelity
		index++
		rain1.lowGlyphCanvas = lowGlyphCanvases[index%2]
		rain1.highGlyphCanvas = highGlyphCanvases[index%2]
		rain2.lowGlyphCanvas = lowGlyphCanvases[index%2]
		rain2.highGlyphCanvas = highGlyphCanvases[index%2]
		js.Global().Get("document").Get("body").Get("style").Set("background", backgrounds[index%2])
	}

	// Keep the program running
	select {}
}

type Duration float64

const Second Duration = 1

func itoa(i int) string {
	return strconv.Itoa(i)
}
func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
func randi() int {
	return int(js.Global().Get("Math").Call("random").Float() * 2147483647.0)
}

type DigitalRain struct {
	parent, canvas  js.Value
	ctx             js.Value
	width, height   float64
	ratio           float64
	timestamp       Duration
	lowGlyphCanvas  *GlyphCanvas
	highGlyphCanvas *GlyphCanvas
	drops           []*waterDrop
	linkover        bool
	screenCols      int
	minSpeed        int
	maxSpeed        int
	brightness      float64
	Clicked         func()
}

func NewDigitalRain(parent js.Value, screenCols int, minSpeed int, maxSpeed int, brightness float64) (*DigitalRain, error) {
	rain := &DigitalRain{parent: parent}
	rain.screenCols = screenCols
	rain.minSpeed = minSpeed
	rain.maxSpeed = maxSpeed
	rain.brightness = brightness
	if err := rain.start(); err != nil {
		return nil, err
	}
	return rain, nil
}
func (r *DigitalRain) start() error {
	var raf string
	for _, s := range []string{"requestAnimationFrame", "webkitRequestAnimationFrame", "mozRequestAnimationFrame"} {
		if !js.Global().Get(s).IsUndefined() {
			raf = s
			break
		}
	}
	if raf == "" {
		panic("requestAnimationFrame is not available")
	}
	defer r.layout()
	var f js.Func
	f = js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Call(raf, f)
		r.loop(Duration(args[0].Float() / 1000))
		return nil
	})
	js.Global().Call(raf, f)
	return nil
}

func (r *DigitalRain) layout() {
	ratio := js.Global().Get("devicePixelRatio").Float()
	width := r.parent.Get("offsetWidth").Float() * ratio
	height := r.parent.Get("offsetHeight").Float() * ratio
	if !r.canvas.IsUndefined() && r.width == width && r.height == height && r.ratio == ratio {
		return
	}
	r.width, r.height, r.ratio = width, height, ratio
	if !r.canvas.IsUndefined() {
		r.parent.Call("removeChild", r.canvas)
	}
	r.canvas = js.Global().Get("document").Call("createElement", "canvas")
	r.ctx = r.canvas.Call("getContext", "2d")
	r.canvas.Set("width", r.width)
	r.canvas.Set("height", r.height)
	r.canvas.Get("style").Set("width", ftoa(r.width/r.ratio)+"px")
	r.canvas.Get("style").Set("height", ftoa(r.height/r.ratio)+"px")
	r.canvas.Get("style").Set("position", "absolute")
	r.parent.Call("appendChild", r.canvas)
	if r.highGlyphCanvas == nil {
		r.highGlyphCanvas = highGlyphCanvases[index%2]
	}
	if r.lowGlyphCanvas == nil {
		r.lowGlyphCanvas = lowGlyphCanvases[index%2]
	}

	clickFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		ev := args[0]
		x := ev.Get("x").Int()
		y := ev.Get("y").Int()
		if r.overLink(x, y) {
			js.Global().Set("location", githubLink)
		} else {
			if r.Clicked != nil {
				r.Clicked()
			}
		}
		return nil
	})
	r.canvas.Call("addEventListener", "click", clickFn)

	moveFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		ev := args[0]
		x := ev.Get("x").Int()
		y := ev.Get("y").Int()
		if r.overLink(x, y) {
			r.canvas.Get("style").Set("cursor", "pointer")
			r.linkover = true
		} else {
			r.canvas.Get("style").Set("cursor", "default")
			r.linkover = false
		}
		return nil
	})
	r.canvas.Call("addEventListener", "mousemove", moveFn)

	r.loop(r.timestamp)
}

func (r *DigitalRain) overLink(x int, y int) bool {
	return x > int(r.width/r.ratio)-320 && y > int(r.height/r.ratio)-50
}

type waterDrop struct {
	col     int
	row     float64
	start   float64
	speed   float64
	glyphs  []int
	spedup  bool
	created Duration
}

func (r *DigitalRain) dropWaterAtCol(col int, speed float64, length int, start float64, created Duration) {
	wd := waterDrop{}
	wd.col = col
	wd.speed = speed
	wd.glyphs = make([]int, length)
	for i := 0; i < length; i++ {
		wd.glyphs[i] = randi() % glyphsCount
	}
	r.drops = append(r.drops, &wd)
	wd.row = start
	wd.start = wd.row
	wd.created = created
}

func (r *DigitalRain) dropRandomWaterDrop(timestamp Duration) {
	col := randi() % r.screenCols
	colcnt := 0
	for _, drop := range r.drops {
		if drop.col == col && int(drop.row)-len(drop.glyphs) < 0 {
			colcnt++
			if colcnt > overlap {
				return
			}
		}
	}
	speed := float64(randi()%(r.maxSpeed-r.minSpeed) + r.minSpeed)
	length := randi()%30 + 10
	start := float64((randi() % r.maxRows()) - r.maxRows()/2)
	r.dropWaterAtCol(col, speed, length, start, timestamp)
}
func (r *DigitalRain) maxRows() int {
	cellSize := r.width / float64(r.screenCols)
	return int((r.height / cellSize) + 2)
}
func (r *DigitalRain) drawGlyphAt(nidx int, col int, row float64, brightness float64, head bool) {
	if col < 0 || col > r.screenCols || row < -1 || row > float64(r.maxRows()) {
		return
	}
	r.drawGlyphElAt(r.lowGlyphCanvas, nidx, col, row, brightness)
	if head {
		r.drawGlyphElAt(r.highGlyphCanvas, nidx, col, row, brightness)
	}
}
func (r *DigitalRain) drawGlyphElAt(glyphCanvas *GlyphCanvas, nidx int, col int, row float64, brightness float64) {
	if brightness <= 0.05 {
		return
	}
	if brightness > 1 {
		brightness = 1
	}
	cellSize := r.width / float64(r.screenCols)
	gy := int(nidx/glyphsCols) * glyphCellSize
	gx := int(nidx%glyphsCols) * glyphCellSize
	cx := cellSize*float64(col) + cellSize/2 - (cellSize*1.5)/2
	cy := cellSize * float64(row)
	glyph := glyphCanvas.Glyph(gx, gy)
	if !glyph.IsUndefined() {
		r.ctx.Call("save")
		r.ctx.Set("globalAlpha", brightness)
		r.ctx.Call("drawImage", glyph, cx, cy, cellSize*1.5, cellSize*1.5)
		r.ctx.Call("restore")
	}
}

func (r *DigitalRain) drawTitle(text string, color string, fontSize float64, y float64) float64 {
	ny := y + (fontSize * 1.5)
	pad := 15 * r.ratio
	x := r.width - pad
	yy := r.height - pad - y
	r.ctx.Call("save")
	r.ctx.Set("font", itoa(int(fontSize))+"px Menlo, Consolas, Monospace, Helvetica, Arial, Sans-Serif")
	r.ctx.Set("textAlign", "right")
	r.ctx.Set("lineWidth", 0)
	r.ctx.Set("shadowColor", color)
	r.ctx.Set("shadowBlur", float64(fontSize))
	r.ctx.Set("fillStyle", color)
	r.ctx.Call("fillText", text, x, yy)
	r.ctx.Call("restore")
	return ny
}

func shortLink(link string) string {
	for i := 0; i < len(link); i++ {
		if link[i] == ':' && i+2 < len(link) && link[i+1] == '/' && link[i+2] == '/' {
			return link[i+3:]
		}
	}
	return link
}

func (r *DigitalRain) drawTitles() {
	y := float64(0)
	if r.linkover {
		y = r.drawTitle(shortLink(githubLink), githubLinkOverColor, 15*r.ratio, y)
	} else {
		y = r.drawTitle(shortLink(githubLink), githubLinkColor, 15*r.ratio, y)
	}
}

func (r *DigitalRain) loop(timestamp Duration) {
	if timestamp == 0 || r.timestamp == 0 {
		r.timestamp = timestamp
		return
	}
	elapsed := timestamp - r.timestamp
	r.timestamp = timestamp

	r.dropRandomWaterDrop(timestamp)
	r.ctx.Call("clearRect", 0, 0, r.width, r.height)
	defer r.drawTitles()

	var drops []*waterDrop
	for _, drop := range r.drops {
		var ri = randi()
		if !drop.spedup {
			if ri%250 == 0 {
				drop.speed *= float64(ri%3) + 0.8
				drop.spedup = true
			}
		}
		gbrightness := r.brightness
		age := (timestamp - drop.created)
		if age < Second {
			gbrightness = float64(age / Second)
		}

		drop.row += float64(elapsed) / float64(Second) * drop.speed
		gl := len(drop.glyphs)

		if int(drop.row)-gl > r.maxRows() {
			continue
		}
		drops = append(drops, drop)
		gcount := int(drop.row - drop.start)
		if gcount > len(drop.glyphs) {
			gcount = len(drop.glyphs)
		}
		for i := 0; i < gcount; i++ {
			var ri = randi()
			glyph := drop.glyphs[i]
			brightness := 1 - (float64(i) / float64(gcount))
			if ri%50 == 0 {
				glyph = ri % glyphsCount
				drop.glyphs[i] = glyph
			}
			row := drop.row - float64(i)
			r.drawGlyphAt(glyph, drop.col, row, brightness*gbrightness, i == 0)
		}
	}
	r.drops = drops
}

const (
	glyphs        = "02345789ABCEGIJMNPRVXYZ:>+*~｡､イエカクコシセタツトニハフホミメヤラハヒルرعلحودסצשאיดฟวㅏㅓㅗㅜ-ㅣŁ"
	glyphsCols    = 18
	glyphsCount   = 72
	glyphCellSize = 100
	glyphFontSize = 86
)

func NewGlyphCanvas(color string) *GlyphCanvas {
	glyphCanvas := &GlyphCanvas{
		jso: js.Global().Get("document").Call("createElement", "canvas"),
	}
	glyphCanvas.jso.Set("width", int(glyphCellSize*glyphsCols+glyphCellSize))
	glyphCanvas.jso.Set("height", int(glyphCellSize*glyphsCount/glyphsCols+glyphCellSize))
	ctx := glyphCanvas.jso.Call("getContext", "2d")
	col := 0
	row := 1
	for i, c := range glyphs {
		if col == 18 {
			row++
			col = 0
		}
		cellSize := float64(glyphCellSize)
		fontSize := float64(glyphFontSize)
		if i <= 36 {
			fontSize *= .87
		}
		ctx.Call("save")
		ctx.Set("textAlign", "center")
		ctx.Set("font", itoa(int(fontSize))+"px Monaco, Helvetica, Arial, Sans-Serif")
		ctx.Set("shadowColor", "rgba(255,255,255,0.1)")
		ctx.Set("shadowBlur", float64(fontSize)*.50)
		switch c {
		default:
			ctx.Call("translate", cellSize*float64(col)+cellSize/2, cellSize*float64(row)+(fontSize-cellSize))
		case '2', '4', '9':
			ctx.Call("scale", -1, 1)
			ctx.Call("translate", -(cellSize*float64(col)+cellSize/2), cellSize*float64(row)+(fontSize-cellSize))
		}
		for i := 0; i < 3; i++ {
			ctx.Set("fillStyle", color)
			ctx.Call("fillText", string(c), 0, 0)
		}
		ctx.Call("restore")
		col++
	}
	return glyphCanvas
}

type GlyphCanvas struct {
	jso    js.Value
	glyphs map[int]map[int]js.Value
}

func (gc *GlyphCanvas) Glyph(gx int, gy int) js.Value {
	if gc.glyphs == nil {
		gc.glyphs = make(map[int]map[int]js.Value)
	}
	mx := gc.glyphs[gx]
	if mx == nil {
		mx = make(map[int]js.Value)
		gc.glyphs[gx] = mx
	}
	my := mx[gy]
	if my.IsUndefined() {
		my = js.Global().Get("document").Call("createElement", "canvas")
		my.Set("width", glyphCellSize)
		my.Set("height", glyphCellSize)
		ctx := my.Call("getContext", "2d")
		ctx.Call("drawImage", gc.jso, gx, gy, glyphCellSize, glyphCellSize, 0, 0, glyphCellSize, glyphCellSize)
		mx[gy] = my
	}
	return my
}
