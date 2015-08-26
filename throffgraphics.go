// Copyright Jeremy Price - Praeceptamachinae.com
//
//Released under the artistic license 2.0

package throfflib
import (    

 "runtime"
 "strconv"
 "github.com/go-gl/glfw/v3.1/glfw"
)



	func init() {
    // This is needed to arrange that main() runs on main thread.
    // See documentation for functions that are only allowed to be called from the main thread.
    runtime.LockOSThread()
}

		
		
//Creates a new engine and populates it with the core functions
func LoadGraphics(e *Engine) *Engine{
	
	
		e=add(e, "glfw.Init", NewCode("glfw.Init", 0, func (e *Engine,c *Thingy) *Engine {
			    err := glfw.Init()
				if err != nil {
					panic(err)
				}
		return e}))
		
		e=add(e, "GLFWEVENTLOOP", NewCode("GLFWEVENTLOOP", 0, func (e *Engine,c *Thingy) *Engine {
			var el1 *Thingy
			el1, e.dataStack = popStack(e.dataStack)
			window := el1._structVal.(*glfw.Window)
			window.MakeContextCurrent()

    for !window.ShouldClose() {
        // Do OpenGL stuff.
        window.SwapBuffers()
        glfw.PollEvents()
    }

		return e}))
		
	e=add(e, "glfw.CreateWindow",  NewCode("glfw.CreateWindow", -1, func (ne *Engine,c *Thingy) *Engine {
		var x,y,title *Thingy
		
		x, ne.dataStack = popStack(ne.dataStack)
		y, ne.dataStack = popStack(ne.dataStack)
		title, ne.dataStack = popStack(ne.dataStack)
		
		xx, _ := strconv.ParseInt( x.getString(), 10, 32 )
		yy, _ := strconv.ParseInt( y.getString(), 10, 32 )
		
		
		    window, err := glfw.CreateWindow(int(xx), int(yy), title.getString(), nil, nil)
    if err != nil {
        panic(err)
    }
		
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(window))
		return ne}))
return e	
}