// +build !android

// Copyright Jeremy Price - Praeceptamachinae.com
//
//Released under the artistic license 2.0

package throfflib
import (

"fmt"
 "strconv"
"github.com/magsoft-se/wg"
"image/color"
 "github.com/go-gl/glfw/v3.1/glfw"
 "github.com/go-gl/gl/v2.1/gl"
 	"image"
	"image/draw"
	_ "image/png"
 "github.com/llgcode/draw2d"
"github.com/llgcode/draw2d/draw2dimg"
"github.com/llgcode/draw2d/draw2dkit"
	"os"
	"runtime"
)
type Point struct {
        x int
        y int
}


var (
	texture   uint32
	rotationX float32
	rotationY float32
	gc *draw2dimg.GraphicContext
)


	func init() {
    // This is needed to arrange that main() runs on main thread.
    // See documentation for functions that are only allowed to be called from the main thread.
    runtime.LockOSThread()
}

var CallbackState *Engine

func WgCallback () {

CallbackState = CallbackState.RunString("WGCALLBACK", "wg callback")

}


func EnsureGC () {
	if (gc == nil) {
		dest := wg.GetImage()
		gc = draw2dimg.NewGraphicContext(dest)
	}
}

//Creates a new engine and populates it with the core functions
func LoadGraphics(e *Engine) *Engine{


	e=add(e, "glfw.Init", NewCode("glfw.Init", 0, func (e *Engine,c *Thingy) *Engine {
		    err := glfw.Init()
			if err != nil {
				panic(err)
			}
	return e}))

	e=add(e, "GLFWEVENTLOOP", NewCode("GLFWEVENTLOOP", 1, func (e *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		window := el1._structVal.(*glfw.Window)
		window.MakeContextCurrent()

		if err := gl.Init(); err != nil {
			panic(err)
		}

		texture = newTexture("square.png")
		defer gl.DeleteTextures(1, &texture)

		setupScene()
		for !window.ShouldClose() {
			drawScene()
			window.SwapBuffers()
			glfw.PollEvents()
		}


		return e}))


	e=add(e, "wg.GetStringBounds", NewCode("wg.GetStringBounds", 0, func (e *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		left, top, right, bottom := gc.GetStringBounds(el1.getString())
		e.dataStack = append(e.dataStack, NewString(fmt.Sprintf("%v",bottom), c))
		e.dataStack = append(e.dataStack, NewString(fmt.Sprintf("%v",right), c))
		e.dataStack = append(e.dataStack, NewString(fmt.Sprintf("%v",top), c))
		e.dataStack = append(e.dataStack, NewString(fmt.Sprintf("%v",left), c))
		return e}))


	e=add(e, "wg.SetFillColor", NewCode("wg.SetFillColor", 1, func (e *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		col := el1._structVal.(color.RGBA)
		EnsureGC()
		gc.SetFillColor(col)
	return e}))

	e=add(e, "wg.SetStrokeColor", NewCode("wg.SetStrokeColor", 1, func (e *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		col := el1._structVal.(color.RGBA)
		EnsureGC()
		gc.SetStrokeColor(col)
	return e}))

	e=add(e, "wg.CloseAndFill", NewCode("wg.CloseAndFill", 0, func (e *Engine,c *Thingy) *Engine {
		EnsureGC()
		gc.Close()
		gc.FillStroke()
	return e}))



	e=add(e, "wg.MoveTo", NewCode("wg.MoveTo", 2, func (e *Engine,c *Thingy) *Engine {
		var el1, el2 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		el2, e.dataStack = popStack(e.dataStack)
		x1, _ := strconv.ParseFloat( el1.getString(), 32 )
		x2, _ := strconv.ParseFloat( el2.getString(), 32 )
		EnsureGC()
		gc.MoveTo(float64(x1),float64(x2)) // should always be called first for a new path
	return e}))


	e=add(e, "wg.LineTo", NewCode("wg.LineTo", 2, func (e *Engine,c *Thingy) *Engine {
		var el1, el2 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		el2, e.dataStack = popStack(e.dataStack)
		x1, _ := strconv.ParseFloat( el1.getString(), 32 )
		x2, _ := strconv.ParseFloat( el2.getString(), 32 )
		EnsureGC()
		gc.LineTo(float64(x1),float64(x2)) // should always be called first for a new path
	return e}))

	e=add(e, "wg.RoundedRectangle", NewCode("wg.RoundedRectangle", 2, func (e *Engine,c *Thingy) *Engine {
		var el1, el2, el3, el4 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		el2, e.dataStack = popStack(e.dataStack)
		el3, e.dataStack = popStack(e.dataStack)
		el4, e.dataStack = popStack(e.dataStack)
		x1, _ := strconv.ParseFloat( el1.getString(), 32 )
		x2, _ := strconv.ParseFloat( el2.getString(), 32 )
		x3, _ := strconv.ParseFloat( el3.getString(), 32 )
		x4, _ := strconv.ParseFloat( el4.getString(), 32 )
		EnsureGC()
		draw2dkit.RoundedRectangle(gc, x1, x2, x3, x4, 10, 10)
		gc.FillStroke()
	return e}))


e=add(e, "wg.FillStringAt", NewCode("wg.FillStringAt", 3, func (e *Engine,c *Thingy) *Engine {
		var el1, el2, el3 *Thingy
		el3, e.dataStack = popStack(e.dataStack)
		el1, e.dataStack = popStack(e.dataStack)
		el2, e.dataStack = popStack(e.dataStack)
		x1, _ := strconv.ParseFloat( el1.getString(), 32 )
		x2, _ := strconv.ParseFloat( el2.getString(), 32 )
		EnsureGC()
		// Set the font luximbi.ttf
		gc.SetFontData(draw2d.FontData{Name: "luxi", Family: draw2d.FontFamilyMono, Style: draw2d.FontStyleBold | draw2d.FontStyleItalic})
		// Set the fill text color to black
		gc.SetFillColor(image.White)
		gc.SetFontSize(15)
		gc.FillStringAt(el3.getString(),float64(x1),float64(x2)) // should always be called first for a new path
	return e}))



	e=add(e, "WG.CLEAR", NewCode("WG.CLEAR", 1, func (e *Engine,c *Thingy) *Engine {
		var el3 *Thingy
		el3, e.dataStack = popStack(e.dataStack)
		col := el3._structVal.(color.RGBA)
		wg.ClearImage(col)
		return e}))

	e=add(e, "WG.TEST", NewCode("WG.TEST", 0, func (e *Engine,c *Thingy) *Engine {
		/*var el1, el2,el3 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		el2, e.dataStack = popStack(e.dataStack)
		el3, e.dataStack = popStack(e.dataStack)
		x1, _ := strconv.ParseInt( el1.getString(), 10, 32 )
		x2, _ := strconv.ParseInt( el2.getString(), 10, 32 )
		col := el3._structVal.(color.RGBA) */
		EnsureGC()
		// Set some properties
		gc.SetFillColor(color.RGBA{0x44, 0xff, 0x44, 0xff})
		gc.SetStrokeColor(color.RGBA{0x44, 0x44, 0x44, 0xff})
		gc.SetLineWidth(5)

		// Draw a closed shape
		gc.MoveTo(100, 100) // should always be called first for a new path
		gc.LineTo(600, 150)
		gc.QuadCurveTo(600, 10, 10, 10)
		gc.Close()
		gc.FillStroke()
		return e}))



	e=add(e, "WG.SETPOINT", NewCode("WG.SETPOINT", 3, func (e *Engine,c *Thingy) *Engine {
		var el1, el2,el3 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		el2, e.dataStack = popStack(e.dataStack)
		el3, e.dataStack = popStack(e.dataStack)
		x1, _ := strconv.ParseInt( el1.getString(), 10, 32 )
		x2, _ := strconv.ParseInt( el2.getString(), 10, 32 )
		col := el3._structVal.(color.RGBA)
		wg.GetImage().Set(int(x1),int(x2),col)
		return e}))


	e=add(e, "RGBA", NewCode("RGBA", 3, func (e *Engine,c *Thingy) *Engine {
		var el1, el2,el3,el4 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		el2, e.dataStack = popStack(e.dataStack)
		el3, e.dataStack = popStack(e.dataStack)
		el4, e.dataStack = popStack(e.dataStack)
		x1, _ := strconv.ParseInt( el1.getString(), 10, 32 )
		x2, _ := strconv.ParseInt( el2.getString(), 10, 32 )
		x3, _ := strconv.ParseInt( el3.getString(), 10, 32 )
		x4, _ := strconv.ParseInt( el4.getString(), 10, 32 )
		col := color.RGBA{uint8(x1),uint8(x2),uint8(x3),uint8(x4)}
		e.dataStack = append(e.dataStack, NewWrapper(col))
		return e}))

	e=add(e, "WG.GETKEY", NewCode("WG.GETKEY", 0, func (e *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, e.dataStack = popStack(e.dataStack)
		xx, _ := strconv.ParseInt( el1.getString(), 10, 32 )
		var ret *Thingy
		if wg.GetKey(int(xx)) {
			ret = NewBool(1)
		} else {
			ret = NewBool(0)
		}
		e.dataStack = append(e.dataStack, ret)
		return e}))
 	e=add(e, "WG.START", NewCode("WG.START", 1, func (ne *Engine,c *Thingy) *Engine {
		CallbackState = ne
    var port *Thingy
    port, ne.dataStack = popStack(ne.dataStack)
    portnum, _ := strconv.ParseInt( port.getString(), 10, 32 )
		wg.Start(800, 600, int(portnum), WgCallback)
		return ne}))



	e=add(e, "glfw.CreateWindow",  NewCode("glfw.CreateWindow", 2, func (ne *Engine,c *Thingy) *Engine {
		var x,y,title *Thingy

		x, ne.dataStack = popStack(ne.dataStack)
		y, ne.dataStack = popStack(ne.dataStack)
		title, ne.dataStack = popStack(ne.dataStack)

		xx, _ := strconv.ParseInt( x.getString(), 10, 32 )
		yy, _ := strconv.ParseInt( y.getString(), 10, 32 )


		glfw.WindowHint(glfw.Resizable, glfw.False)
		glfw.WindowHint(glfw.ContextVersionMajor, 2)
		glfw.WindowHint(glfw.ContextVersionMinor, 1)

		window, err := glfw.CreateWindow(int(xx), int(yy), title.getString(), nil, nil)
		if err != nil {
			panic(err)
		}

		ne.dataStack = pushStack(ne.dataStack, NewWrapper(window))
		return ne}))
return e
}


func newTexture(file string) uint32 {
	imgFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		panic(err)
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		panic("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.Enable(gl.TEXTURE_2D)
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture
}

func setupScene() {
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.LIGHTING)

	gl.ClearColor(0.5, 0.5, 0.5, 0.0)
	gl.ClearDepth(1)
	gl.DepthFunc(gl.LEQUAL)

	ambient := []float32{0.5, 0.5, 0.5, 1}
	diffuse := []float32{1, 1, 1, 1}
	lightPosition := []float32{-5, 5, 10, 0}
	gl.Lightfv(gl.LIGHT0, gl.AMBIENT, &ambient[0])
	gl.Lightfv(gl.LIGHT0, gl.DIFFUSE, &diffuse[0])
	gl.Lightfv(gl.LIGHT0, gl.POSITION, &lightPosition[0])
	gl.Enable(gl.LIGHT0)

	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Frustum(-1, 1, -1, 1, 1.0, 10.0)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
}

func destroyScene() {
}

func drawScene() {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Translatef(0, 0, -3.0)
	gl.Rotatef(rotationX, 1, 0, 0)
	gl.Rotatef(rotationY, 0, 1, 0)

	rotationX += 0.5
	rotationY += 0.5

	gl.BindTexture(gl.TEXTURE_2D, texture)

	gl.Color4f(1, 1, 1, 1)

	gl.Begin(gl.QUADS)

	gl.Normal3f(0, 0, 1)
	gl.TexCoord2f(0, 0)
	gl.Vertex3f(-1, -1, 1)
	gl.TexCoord2f(1, 0)
	gl.Vertex3f(1, -1, 1)
	gl.TexCoord2f(1, 1)
	gl.Vertex3f(1, 1, 1)
	gl.TexCoord2f(0, 1)
	gl.Vertex3f(-1, 1, 1)

	gl.Normal3f(0, 0, -1)
	gl.TexCoord2f(1, 0)
	gl.Vertex3f(-1, -1, -1)
	gl.TexCoord2f(1, 1)
	gl.Vertex3f(-1, 1, -1)
	gl.TexCoord2f(0, 1)
	gl.Vertex3f(1, 1, -1)
	gl.TexCoord2f(0, 0)
	gl.Vertex3f(1, -1, -1)

	gl.Normal3f(0, 1, 0)
	gl.TexCoord2f(0, 1)
	gl.Vertex3f(-1, 1, -1)
	gl.TexCoord2f(0, 0)
	gl.Vertex3f(-1, 1, 1)
	gl.TexCoord2f(1, 0)
	gl.Vertex3f(1, 1, 1)
	gl.TexCoord2f(1, 1)
	gl.Vertex3f(1, 1, -1)

	gl.Normal3f(0, -1, 0)
	gl.TexCoord2f(1, 1)
	gl.Vertex3f(-1, -1, -1)
	gl.TexCoord2f(0, 1)
	gl.Vertex3f(1, -1, -1)
	gl.TexCoord2f(0, 0)
	gl.Vertex3f(1, -1, 1)
	gl.TexCoord2f(1, 0)
	gl.Vertex3f(-1, -1, 1)

	gl.Normal3f(1, 0, 0)
	gl.TexCoord2f(1, 0)
	gl.Vertex3f(1, -1, -1)
	gl.TexCoord2f(1, 1)
	gl.Vertex3f(1, 1, -1)
	gl.TexCoord2f(0, 1)
	gl.Vertex3f(1, 1, 1)
	gl.TexCoord2f(0, 0)
	gl.Vertex3f(1, -1, 1)

	gl.Normal3f(-1, 0, 0)
	gl.TexCoord2f(0, 0)
	gl.Vertex3f(-1, -1, -1)
	gl.TexCoord2f(1, 0)
	gl.Vertex3f(-1, -1, 1)
	gl.TexCoord2f(1, 1)
	gl.Vertex3f(-1, 1, 1)
	gl.TexCoord2f(0, 1)
	gl.Vertex3f(-1, 1, -1)

	gl.End()
}
