// +build !android

// Copyright Jeremy Price - Praeceptamachinae.com
//
//Released under the artistic license 2.0

package throfflib
import (
	"strconv"
	"fmt"
	"bytes"
	"runtime"
	"github.com/gordonklaus/portaudio"
	"text/template"
	"math/rand"
)

const sampleRate = 44100

var AudioCallbackState *Engine

func processAudio(out [][]float32) {
	AudioCallbackState.dataStack = pushStack(AudioCallbackState.dataStack, NewWrapper(out))
	AudioCallbackState = AudioCallbackState.RunString(fmt.Sprintf("CLEARSTACK  AUDIOCALLBACK %v", len(out[0])), "audio callback")
	for i := range out[0] {
		out[0][i] = rand.Float32() //float32(math.Sin(2 * math.Pi * g.phaseL))
		//_, g.phaseL = math.Modf(g.phaseL + g.stepL)
		//out[1][i] = float32(math.Sin(2 * math.Pi * g.phaseR))
		//_, g.phaseR = math.Modf(g.phaseR + g.stepR)
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}


func init() {
    // This is needed to arrange that main() runs on main thread.
    // See documentation for functions that are only allowed to be called from the main thread.
    runtime.LockOSThread()
}


//Creates a new engine and populates it with the core functions
func LoadAudio(e *Engine) *Engine{


	e=add(e, "portaudio.init", NewCode("portaudio.init", 0, func (e *Engine,c *Thingy) *Engine {
		portaudio.Initialize()
		return e}))

	e=add(e, "portaudio.terminate", NewCode("portaudio.terminate", 0, func (e *Engine,c *Thingy) *Engine {
		defer portaudio.Terminate()
		return e}))
	
	e=add(e, "portaudio.setData",  NewCode("setData", 3, func (ne *Engine,c *Thingy) *Engine {
            var posString, buffWrap, valString *Thingy
            posString, ne.dataStack = popStack(ne.dataStack)
			valString, ne.dataStack = popStack(ne.dataStack)
            buffWrap, ne.dataStack = popStack(ne.dataStack)

			var pos,_ = strconv.ParseInt( posString.getSource() , 10, 32 )
			var val,_ = strconv.ParseFloat( valString.getSource() , 32 )
			out := buffWrap._structVal.([][]float32)
			out[0][pos] = float32( val)
			
		    return ne
        }))
	
	e=add(e, "portaudio.start", NewCode("portaudio.start", -1, func (e *Engine,c *Thingy) *Engine {
		//AudioCallbackState = e.RunString("CLEARSTACK", "setup audio callback")
		AudioCallbackState = e
		stream, err := portaudio.OpenDefaultStream(0, 2, sampleRate, 0, processAudio)
		chk(err)
		stream.Start()
		chk(err)
		e.dataStack = pushStack(e.dataStack, NewWrapper(stream))
	chk(err)
	return e}))

	e=add(e, "portaudio.HostApis", NewCode("portaudio.HostApis", 0, func (e *Engine,c *Thingy) *Engine {
		var tmpl = template.Must(template.New("").Parse(
	`[ {{range .}}
	
	H[ 
		Name                   [ {{.Name}} ]
		{{if .DefaultInputDevice}}DefaultInput   [ {{.DefaultInputDevice.Name}} ] {{end}}
		{{if .DefaultOutputDevice}}DefaultOutput  [ {{.DefaultOutputDevice.Name}} ] {{end}}
		Devices [ {{range .Devices}}
					H[	
						Name                      [ {{.Name}} ]
						MaxInputChannels          [ {{.MaxInputChannels}} ]
						MaxOutputChannels         [ {{.MaxOutputChannels}} ]
						DefaultLowInputLatency    [ {{.DefaultLowInputLatency}} ]
						DefaultLowOutputLatency   [ {{.DefaultLowOutputLatency}} ]
						DefaultHighInputLatency   [  {{.DefaultHighInputLatency}} ]
						DefaultHighOutputLatency  [  {{.DefaultHighOutputLatency}} ]
						DefaultSampleRate         [  {{.DefaultSampleRate}} ]
					]H
				{{end}}
				]
	]H
{{end}}
]`,
))
		var b bytes.Buffer
		hs, err := portaudio.HostApis()
		chk(err)
		err = tmpl.Execute(&b, hs)
		chk(err)
		s := b.String()
		fmt.Println(s)
		return e}))

return e
}