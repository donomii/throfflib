// Copyright Jeremy Price - Praeceptamachinae.com
//
//Released under the artistic license 2.0

package throfflib
import (    

 "strconv"
"github.com/magsoft-se/wg"
"image/color"
 "github.com/go-gl/glfw/v3.1/glfw"
 "github.com/go-gl/gl/v2.1/gl"
 	"image"
	"image/draw"
	_ "image/png"
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


	e=add(e, "WG.CLEAR", NewCode("WG.CLEAR", 1, func (e *Engine,c *Thingy) *Engine {
		var el3 *Thingy
		el3, e.dataStack = popStack(e.dataStack)
		col := el3._structVal.(color.RGBA)
		wg.ClearImage(col)
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
 	e=add(e, "WG.START", NewCode("WG.START", 1, func (e *Engine,c *Thingy) *Engine {
		CallbackState = e
		wg.Start(800, 600, 8500, WgCallback)
		return e}))


		
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
