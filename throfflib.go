package throfflib
// Copyright Jeremy Price - Praeceptamachinae.com
//
//Released under the artistic license 2.0

// #cgo CFLAGS: -IC:/Users/user/Desktop/Dropbox/goProjects/src/github.com/thinxer/go-tcc
/*
#include <stdlib.h>

typedef float (*jitfunc)(float a, float b);

static float call(jitfunc f,float a, float b) {
    return f(a, b);
}
*/
import "C"

import "unsafe"
import "github.com/thinxer/go-tcc"
import "math/big"
import "fmt"
import "time"
import "strings"
import "bufio"
import "os"
import "io/ioutil"
import "strconv"
import "sort"
import "math"
import "runtime"
import "github.com/edsrzf/mmap-go"
import "net/http"
import "net"
import "html"
import "net/rpc"
import "net/rpc/jsonrpc"
import "log"
import "io"
import "bytes"
import "database/sql"
import "github.com/mdlayher/arp"
import ( _ "github.com/mattn/go-sqlite3" )
import "golang.org/x/net/websocket"
import "github.com/chzyer/readline"

var precision uint = 256
var interpreter_debug = false
var interpreter_trace = false
var traceProg = false
var debug = false
var USE_FUNCTION_CACHE = false
var seqID = 0					//Every instruction in the program gets a unique number, used by the optimiser and similar tasks

type StepFunc func(*Engine, *Thingy) *Engine

// The fundamental unit of data.  In this engine, arrays, hashes, strings and numbers are all "Thingy"s
type Thingy struct {
	annotations 	map[string]Thingy
	_stub			StepFunc					//The native code function that will perform the next command in the program
	tiipe			string						//The tiipe is the user-visible type.  Tiipes are very flexible especially string<->token and code<->array
	subType			string						//The interpreter visible type.  Almost always "NATIVE" or "INTERPRETED"
	userType		*Thingy						//A purely user defined type.  Can be anything
	_source	string								//A string that, in theory, could be eval-ed to recreate the data structure.  Also used for if statement comparisons
	environment 	*Thingy  					//Functions carry a reference to their parent environment, which allows us to create a new environment for the function to run in each time it is called
    errorChain      stack                       //Error handlers are functions that are kept on a stack
	//_parentEnvironment *Thingy
	lock 			*Thingy
	_structVal		interface{}
	_stringVal		string
	_arrayVal		stack
	_engineVal		*Engine
	_hashVal		map[string]*Thingy
	_bytesVal		[]byte
	_note			string
	arity			int                     //The number of arguments this function pops off the stack, minus the number of return results`
	_intVal			int                     //Currently used for booleans, will also be used to cache string->int conversions
	_id				int                     //Every token gets a unique id number, this allows us to do JIT on the code, amoung other tricks
	_line			int                     //Line number of instruction
	_filename		string                  //The source code file this instruction was written in
	codeStackConsume	int                 //When the functions are cached, we need to know how long they are, and skip the build phase by popping this many Thingys off the code stack
	immutable		bool
	share_parent_environment	bool			//Instead of creating a new lexical pad (environment to store variables), use the surrounding lexical environment.  This allows e.g. if statements to affect variables in the surrounding function
	no_environment	bool						//AKA macro.  Instead of using its own lexical environment, this function will use the lexical environment that it is INVOKED in.  Required for doing complicated bind operations, probably a bad idea for anything else

}

type stack []*Thingy
type Engine struct {
	//previousEngine	*Engine
	environment 		*Thingy  			//The current lexical environment
	dataStack 			stack				//The argument stack
    dyn                 stack               //The current dynamic environment
	codeStack			stack				//The future of the program
	lexStack			stack
	_buildingFunc		bool				//do we run functions or just shift them to the data stack?
	_funcLevel			int
	_prevLevel			int
	_functionCache		map[int]*Thingy		//Cache built functions here to avoid expensive rebuilds
	_line				int
	_heatMap			map[string][]int
	_safeMode			bool
}

func (t *Thingy) setString (val string) {
	t._stringVal=val
	t._source=val
}
func (t *Thingy) setStub (val StepFunc) {
	t._stub=val
	t._source="A function added by setStub"
	t._note="A function added by setStub"
}


//Builds a string that, when EVALed, will recreate the data structure
//The string might not be an exact representation of the data, but should recreate it perfectly
func (t *Thingy) getSource () string {
	if t.tiipe == "ARRAY" {
		var accum string = "A[ "
		for _,el := range t._arrayVal { accum=fmt.Sprintf("%v STRING [ %v ]", accum, el._source)}
		return fmt.Sprintf("%v ]A", accum)
	}
	if t.tiipe == "CODE" || t.tiipe == "LAMBDA" {
		var accum string = "[ "
		for _,el := range t._arrayVal { accum=fmt.Sprintf("%v %v", accum, el._source)}
		return fmt.Sprintf("%v ]", accum)
	}

	if t.tiipe == "HASH"  {
		var accum string = "H[ "
		for k,v := range t._hashVal { accum=fmt.Sprintf("%s %s %s", accum, k, v.getSource())}
		return fmt.Sprintf("%v ]H", accum)
	}

// 		if t.tiipe == "STRING" {
// 		return fmt.Sprintf("STRING [ %v ]", t._stringVal)
// 	}

	return t._source
}

//Builds a string for display to the user.  Probably can't be used to re-create the original data structure
func (t *Thingy) getString () string {
	if t.tiipe == "ARRAY"  {
		var accum string = ""
		for i,el := range t._arrayVal {
		if i==0 {
				accum=fmt.Sprintf("%v",  el.getString())
		} else {
				 accum=fmt.Sprintf("%v  %v", accum, el.getString())
		}
		}
		return accum
	}

  	if t.tiipe == "CODE" || t.tiipe == "LAMBDA" {
  		var accum string = ""
  		for i,el := range t._arrayVal {
			if i==0 {
				accum=fmt.Sprintf("%v",  el.getString())
		} else {
				accum=fmt.Sprintf("%v %v", accum, el.getString())
		}
		}
  		return  accum
  	}

  	if t.tiipe == "HASH"  {
		var accum string = "{ "
		for k,v := range t._hashVal { accum=fmt.Sprintf("%s, %s -> %s", accum, k, v.getString())}
		return fmt.Sprintf("%v }", accum)
	}

	if t.tiipe == "BOOLEAN"  {
		if t._intVal == 0 {
			return "FALSE"
		} else {
			return "TRUE"
		}
	}

	return t._stringVal
}

func actualClone(t Thingy) *Thingy {
	return &t}

func clone(t *Thingy) *Thingy {
	//fmt.Printf("Cloning thingy %v\n\n", t )
	return actualClone(*t)
}


//The engine is cloned at each step
func actualCloneEngine(t Engine) *Engine {return &t}
func cloneEngine(t *Engine, immutable bool) *Engine {
	//fmt.Printf("Cloning engine %v\n\n", t )
	//engineDump(t)
	newt:= actualCloneEngine(*t)
	if immutable {
		newt.environment=cloneEnv(t.environment)
	} else {
		newt.environment=t.environment
	}
	//newt.previousEngine=t  //This is a memory leak

	return newt
}

func nameSpaceLookup(e *Engine, t *Thingy) (*Thingy, bool) {
	key := t.getString()
	val, ok := e.environment._hashVal[key]
	if interpreter_debug {
		emit(fmt.Sprintf("%p: Looking up: %v -> %v in %v\n", e.environment, key, val, e.environment))
	}
	if  !ok {
	var _,ok = strconv.ParseFloat( t.getSource() , 32 )		//Numbers don't need to be defined in the namespace
	if  ok != nil {
		if e._safeMode { emit(fmt.Sprintf("Warning: %v not defined at line %v\n", key, t._line)) }
	}
	}
	return val,ok
}


func cloneMap(m map[string]*Thingy) map[string]*Thingy{
	//fmt.Printf("Cloning map %v\n\n", m )
	var nm = make(map[string]*Thingy, 1000)
	for k,v :=range m { nm[k]=v }
	return nm
}

func cloneEnv(env *Thingy) *Thingy{
	newEnv := clone(env)
	newEnv._hashVal = cloneMap(env._hashVal)
	//fmt.Printf("Cloning map %v\n\n", m )
	return newEnv
}


func add(e *Engine, s string, t *Thingy) *Engine {
	ne := cloneEngine(e, false)
	t._note = s
	ne.environment._hashVal[s] = t
	//t.environment = ne.environment
	t.share_parent_environment=true
	t.no_environment=true
	return ne
}

func newThingy () *Thingy {
	t := new(Thingy)
	//The default action is to push the thing on to the data stack
	t._stub = func (e *Engine,c *Thingy) *Engine {
		ne := cloneEngine(e, false)
		ne.dataStack=pushStack(ne.dataStack, c)
		return ne
	}

	return t
}


//This is the native code called when the engine encounters a token.  It looks up the token and replaces it with the correct value.
//It also builds functions when it spots the function brackets [ ]
//This function got out of hand, and should be broken up
func tokenStepper (e *Engine,c *Thingy) *Engine {
		ne := cloneEngine(e, false)						//FIXME e should already be a clone?
		//Are we in function-building mode?
		ne._prevLevel=ne._funcLevel
		if(c.getSource()=="[") { ne._funcLevel-=1}		//Finish a (possibly nested) function
		if(c.getSource()=="]") {
			//fmt.Printf("Cached func: %v - %v, %v\n", c._id, ne._functionCache[c._id], c.getSource())
			if USE_FUNCTION_CACHE && ne._functionCache[c._id] != nil {

				f := ne._functionCache[c._id]
				//fmt.Printf("Cached func: %v - %v\n", c._id, f.getSource())
				//We have previously seen this function, and have it cached
				//fmt.Printf("Found cached function, skipping build...\n")
				//engineDump(ne)
				f.environment = ne.environment
				ne.dataStack = pushStack(ne.dataStack, f)
				var i int
				for i=0;i<= f.codeStackConsume;i++ {
					_,ne.codeStack = popStack(ne.codeStack)
					_,ne.lexStack = popStack(ne.lexStack)
				}
				//fmt.Printf("Skip complete\n")
				//engineDump(ne)
				return ne
			} else {
				ne._funcLevel+=1
			}
		} 		//Start a (possibly nested) function
		//fmt.Printf("TOKEN: in function level: %v\n", ne._funcLevel)
		if (ne._funcLevel<0) {
			emit(fmt.Sprintf ("Unmatched [ at line %v\n", c._line))
			engineDump(ne)
			panic(fmt.Sprintf ("Unmatched [ at line %v\n", c._line))

		}	//Too many close functions, not enough opens
		if (ne._funcLevel==0) {							//Either we are not building a function, or we just finished
			if(c.getSource()=="BuildFuncFromStack") {	//We move to phase 2, assembling the function from pieces on the data stack
				//fmt.Printf("debug: %v\n", c._source)
				ne._buildingFunc=false					//Switch out of phase 1 build mode
				ne._funcLevel+=1						//Start counting function brackets
				ne.dataStack = pushStack(ne.dataStack, NewString("[", e.environment))
				return c._stub(ne, c)
			} else {
				val,ok := nameSpaceLookup(ne, c)
				if (ok) {
					if val.tiipe == "CODE" {

						ne.codeStack = pushStack(ne.codeStack, val)
						ne.lexStack = pushStack(ne.lexStack, e.environment)
					} else {
						ne.dataStack = pushStack(ne.dataStack, val)
					}
				} else {
					var _,ok = strconv.ParseFloat( c.getSource() , 32 )		//Numbers don't need to be defined in the namespace
					if  ok != nil {
						//fmt.Printf("Warning:  %v not defined\n", c.getString())

					}
					ne.dataStack = pushStack(ne.dataStack, c)
				}
			}
		} else {
			ne.dataStack = pushStack(ne.dataStack, c)
		}
		//fmt.Printf("TokenStep: %v\n", c._source)
	return ne}


//Tokens cause a namespace lookup on their string value
//Whatever gets returned is pushed onto the code stack
//Then on the next step, that Thing gets activated, usually moving itself to the data stack, or running some code
//This can cause infinite loops if the token resolves to itself
func NewToken (aString string, env *Thingy) *Thingy {
	t := newThingy()
	t.tiipe="TOKEN"
	t.subType="NATIVE"
	t.setString(aString)
	t.environment = env
	t._stub = tokenStepper
	t.arity=0
	return t
}


//Raw byte representated.
func NewBytes (bytes []byte,  env *Thingy) *Thingy {
	t := newThingy()
	t.tiipe="BYTES"
	t.subType="NATIVE"
	t._bytesVal= bytes
	t.environment = nil
	t._stub = func (e *Engine,c *Thingy) *Engine {
		ne := cloneEngine(e, false)
		ne.dataStack = pushStack(ne.dataStack, c)
		//fmt.Printf("StringStep: %v\n", c._source)
	return ne}
	t.arity=-1
	return t
}

//Unicode strings.  Length etc might not be the same as its byte representation
func NewString (aString string,  env *Thingy) *Thingy {
	t := newThingy()
	t.tiipe="STRING"
	t.subType="NATIVE"
	t.setString(aString)
	t.environment = nil
	t._stub = func (e *Engine,c *Thingy) *Engine {
		ne := cloneEngine(e, false)
		ne.dataStack = pushStack(ne.dataStack, c)
		//fmt.Printf("StringStep: %v\n", c._source)
	return ne}
	t.arity=-1
	return t
}

//Stores a reference to the engine at the point where it was called.  When activated, execution continues at that point
func NewContinuation (e *Engine) *Thingy {
	t := newThingy()
	t.tiipe="CONTINUATION"
	t.subType="NATIVE"
	t.setString("Continuation")
	t._engineVal=e
	return t
}

//Holds any go structure, like a filehandle, network socket or database handle
func NewWrapper (s interface{}) *Thingy {
	t := newThingy()
	t._structVal = s
	t.tiipe="WRAPPER"
	t.subType="NATIVE"
	t.setString("Native structure wrapper")
	t.arity=-1
	return t
}

//Wraps a native go function
func NewCode (aName string, arity int, aFunc StepFunc) *Thingy {
	t := newThingy()
	t.tiipe="CODE"
	t.subType="NATIVE"
	t.setStub(aFunc)
	t.setString(aName)
	t.arity=arity
	return t
}


func NewArray (a stack) *Thingy {
	t := newThingy()
	t.tiipe="ARRAY"
	t.subType="INTERPRETED"
	t.setString("Array - add code to fill this in properly")
	t.arity=-1
	t._arrayVal = a
	return t
}

func NewHash () *Thingy {
	t := newThingy()
	t.tiipe="HASH"
	t.subType="NATIVE"
	t.setString("hash - add code to fill this in properly")
	t.arity=-1
	t._hashVal = make(map[string]*Thingy, 1000)
	return t
}

func NewBool (a int) *Thingy {
	t := newThingy()
	t.tiipe="BOOLEAN"
	t.subType="NATIVE"
	t.setString("BOOLEAN")
	t._intVal=a
	t.arity=-1
	return t
}
func NewEngine () *Engine{
	e:=new(Engine)
	e.environment = NewHash()
	e._functionCache = map[int]*Thingy{}
	e._heatMap = map[string][]int{}
	e._funcLevel=0
	e._safeMode = false
	return e
}
func NullStep(e *Engine) *Engine {
	return e
}
func pushStack (s stack, v *Thingy) stack {
	return append(s,v)
}
func popStack (s stack) (*Thingy, stack) {

	if len(s)>0 {
		v, sret := s[len(s)-1], s[:len(s)-1]
		return v, sret
	} else {

		panic("Attempted read past end of stack!")
	}
}

//neatly print out all the variables in scope
func dumpEnv ( e *Thingy ) {
		if e != nil {
		emit(fmt.Sprintf("===Env=== %p\n", e))
		keys :=make([]string,0,len(e._hashVal))
		for k, _ := range e._hashVal { keys = append(keys, k) }
		sort.Strings(keys)
		emit(fmt.Sprintln(keys))
		}
}


//The core of the interpreter.  Each step advances the program by one command
func doStep(e *Engine) (*Engine, bool) {
	if len(e.codeStack)>0 {											//If there are any instructions left
		var v, lex *Thingy
        var dyn stack
		//v, _ = popStack(e.codeStack)
		ne := cloneEngine(e, false)								//Clone the current engine state.  The false means "do not clone the lexical environment" i.e. it
																//will be common to this step and the previous step.  Otherwise we would be running in fully
																//immutable mode (to come)
		v, ne.codeStack = popStack(ne.codeStack)							//Pop an instruction off the instruction stack.  Usually a token, but could be data or native code
		lex, ne.lexStack = popStack(ne.lexStack)
        dyn = lex.errorChain
		if v._line !=0 && len(v._filename) > 1 {
				m := ne._heatMap[v._filename]
				if m == nil {
				m=make([]int, 1000000,1000000)}
				m[v._line] = m[v._line]+1
				ne._heatMap[v._filename] = m

		}

		if interpreter_debug && v.environment != nil {dumpEnv(v.environment)}


		if    v.tiipe == "CODE" && v.no_environment == true {							//Macros do not carry their own environment, they use the environment from the previous instruction
			if interpreter_debug {
				emit(fmt.Sprintf("Macro using invoked environment"))
			}
			lex = ne.environment						//Set the macro environment to the current environment i.e. of the previous instruction, which will usually be the token's environment
		} else {
			ne._line = v._line
			if lex != nil {
			ne.environment = lex								//Set the current environment to that carried by the current instruction
            ne.dyn = dyn
			}
		}

		//ne.environment = lex
		if traceProg {
			emit(fmt.Sprintf("%v:Step: %v(%v) - (%p) \n", v._line, v.getString(), v.tiipe, lex))
		}
		//fmt.Printf("Calling: '%v'\n", v.getString())
		if interpreter_debug {
			emit(fmt.Sprintf("Choosing environment %p for command %v(%v)\n", ne.environment, v.getString(), ne.environment))
		}

		oldlen:= len(ne.dataStack)								//Note the size of the data stack
		if interpreter_debug {
			emit(fmt.Sprintf("Using environment: %p for command : %v\n", ne.environment, v.getString()))
		}
		ne = v._stub(ne, v)										//Call the handler function for this instruction
		newlen := len(ne.dataStack)								//Note the new data stack size
		if ((v.arity > 9000) || (v.tiipe == "TOKEN")) {			//Tokens and some other instructions can have variable arity
			return ne, true
		} else {
			if (v.arity==(oldlen-newlen)) {						//Check the number of arguments to an instruction against the change in the stack size
				return ne, true
			} else {
				if v.tiipe == "CODE" {
				emit(fmt.Sprintln(fmt.Sprintf("Arity mismatch in native function! %v claimed %v, but actually took %v\n", v.getSource(), v.arity, (oldlen-newlen))))

				}
				return ne, true
			}
		}
	} else {
		//fmt.Printf("No code left to run!\n")
		return e, false
	}
}

func SlurpFile ( fname string ) (string) {
	content, _ := ioutil.ReadFile(fname)
	return string(content)
}


func tokenise (s string, filename string) stack {
	var line int = 1
	s=strings.Replace(s, "\n", " LINEBREAKHERE ", -1)
	s=strings.Replace(s, "\r", " ", -1)
	s=strings.Replace(s, "\t", " ", -1)
	stringBits := strings.Split(s, " ")
	var tokens  stack
	for _,v :=range stringBits {
		seqID=seqID+1
		if len(v)>0 {
			if v== "LINEBREAKHERE" {
				line = line+1
			} else {
			t := NewToken(v,NewHash())
			t._id=seqID
			t._line = line
			t._filename = filename
			//fmt.Printf("Token id: %v\n", i)
			tokens=pushStack(tokens, t)
			}
		}
		}
	return tokens
}

func StringsToTokens (stringBits []string) stack {
	var tokens  stack
	for i,v :=range stringBits {
		if len(v)>0 {
						t := NewToken(v,NewHash())
			t._id=i
			tokens=pushStack(tokens, t)
		}
	}
	return tokens
}

func engineDump (e *Engine) {
		emit(fmt.Sprintf("Stack: %v, Code: %v, Environment: %v items\n", len(e.dataStack), len(e.codeStack), len(e.environment._hashVal)))
		emit(fmt.Sprintf("---------------------------"))
		emit(fmt.Sprintf("|| code: "))
		stackDump(e.codeStack)
		emit(fmt.Sprintf("\n"))
		emit(fmt.Sprintf("|| data: "))
		stackDump(e.dataStack)
		emit(fmt.Sprintf("\n"))
		emit(fmt.Sprintf("----------------------------"))
}
func run (e *Engine) (*Engine, bool) {
	ok := true
	for ok {
		e, ok = doStep(e)
	}
	return e, ok
}

func (e *Engine) Run() (*Engine){
	e, _ = run(e)
	return e
}

func (e *Engine) LoadTokens (s stack)  {
	for _, elem := range s {
		elem.environment = e.environment
		e.lexStack = pushStack(e.lexStack, e.environment)
	}  //All tokens start off sharing the root environment
	e.codeStack = append(e.codeStack, s...)
	//e.codeStack = s
}


// Function constructor - constructs new function for listing given directory
func listFiles(path string) func(string) []string {
    return func(line string) []string {
        names := make([]string, 0)
        files, _ := ioutil.ReadDir(path)
        for _, f := range files {
            names = append(names, f.Name())
        }
        return names
    }
}

var completer = readline.NewPrefixCompleter(
    readline.PcItem("mode",
        readline.PcItem("vi"),
        readline.PcItem("emacs"),
    ),
    readline.PcItem("login"),
    readline.PcItem("say",
        readline.PcItemDynamic(listFiles("./"),
            readline.PcItem("with",
                readline.PcItem("following"),
                readline.PcItem("items"),
            ),
        ),
        readline.PcItem("hello"),
        readline.PcItem("bye"),
    ),
    readline.PcItem("setprompt"),
    readline.PcItem("setpassword"),
    readline.PcItem("bye"),
    readline.PcItem("help"),
    readline.PcItem("go",
        readline.PcItem("build", readline.PcItem("-o"), readline.PcItem("-v")),
        readline.PcItem("install",
            readline.PcItem("-v"),
            readline.PcItem("-vv"),
            readline.PcItem("-vvv"),
        ),
        readline.PcItem("test"),
    ),
    readline.PcItem("sleep"),
)

func Repl (e *Engine) *Engine {
    l, err := readline.NewEx(&readline.Config{
        Prompt:          "Throff \033[31m»\033[0m ",
        HistoryFile:     "/tmp/readline.tmp",
        AutoComplete:    completer,
        InterruptPrompt: "^C",
        EOFPrompt:       "exit",
    })
    if err != nil {
        panic(err)
    }
    defer l.Close()
    log.SetOutput(l.Stderr())

    return realRepl(e, l)
}

func realRepl (e *Engine, rl *readline.Instance) *Engine {
	//engineDump(e)
	emit(fmt.Sprintf("Ready> "))
	//reader := bufio.NewReader(os.Stdin)
	//text, _ := reader.ReadString('\n')
    line, err := rl.Readline()
        if err == readline.ErrInterrupt {
            if len(line) == 0 {
                return e
            } else {
                //continue
            }
        } else if err == io.EOF {
            return e
        }
    text := line
	if (len(text)>0) {
		e.LoadTokens(tokenise(text, "repl"))
		//stackDump(e.codeStack)
		e,_=run(e)
		emit(fmt.Sprintln(e.dataStack[len(e.dataStack)-1].getString()))
		realRepl(e,rl)
		return e
	} else {
		return e
	}
}

var outputBuff string = ""
//All output to STDOUT should go through this function, so we can resend it to network ports, logfiles etc
func emit (s string) {
    outputBuff = fmt.Sprintf("%s%s", outputBuff, s)
    fmt.Printf("%s", s)
}

func clearOutput () string {
    var temp = outputBuff
    outputBuff = ""
    return temp
}

func (e *Engine) RunString (s string, source string) (*Engine) {
	e.LoadTokens(tokenise(s, source))
	e, _ = run(e)
	return e
}

func (e *Engine) RunFile (s string) (*Engine) {
	codeString := SlurpFile(s)
	//println(codeString)
	return e.RunString(codeString, s)
}

func stackDump (s stack) {
	emit(fmt.Sprintf("\nStack: "))
	for i, _:= range s {
		if  i< 20 {
			emit(fmt.Sprintf(":%v(%v):", s[len(s)-1-i].getSource(), s[len(s)-1-i].tiipe))
        }
	}
}

//This is the code that is called when a function is activated
func buildFuncStepper (e *Engine,c *Thingy) *Engine {
			ne:=cloneEngine(e, false)
			//fmt.Printf("Loading code\n")
			var  lexical_env *Thingy
			//fmt.Printf("Share parent env: %v\n", c.share_parent_environment)
			if c.tiipe == "CODE" {
				if c.share_parent_environment == true {
					//fmt.Println("Sharing parent environment\n")
					lexical_env = c.environment
					if c.no_environment == true {
						lexical_env = e.environment
					}
				} else {
					//fmt.Println("Creating new environment for function activation\n")
					lexical_env = cloneEnv(c.environment)  //Copy the lexical environment to make a new environment for this function (e.g. think of recursion)
                    lexical_env.errorChain = e.dyn
				}
				//var newArr stack
				//copy(newArr,c._arrayVal)
				for _,ee := range c._arrayVal {
					//t := clone(ee)
					//t.environment = lexical_env
					ne.codeStack = pushStack(ne.codeStack, ee)
					ne.lexStack = pushStack(ne.lexStack, lexical_env)
				} //Make a string->token function?
			} else {
				ne.dataStack = pushStack(ne.dataStack, c)
			}
			return ne}

//Recurse down the stack, picking up words and collecting them in the stack f. f will become our function.
func buildFunc(e *Engine, f stack) *Engine {
	var v *Thingy
	ne:=cloneEngine(e, false)
	v,ne.dataStack=popStack(ne.dataStack)
	if(v.getSource()=="[") { ne._funcLevel+=1}
	if(v.getSource()=="]") { ne._funcLevel-=1}
	//fmt.Printf("BUILDFUNC: in function level: %v\n", ne._funcLevel)
	//fmt.Printf("Considering %v\n", v.getSource())
	if ( v.getSource()=="]" && ne._funcLevel==0) {
		//fmt.Printf("fINISHING FUNCTION\n")
		//This code is called when the newly-built function is activated
		newFunc:=NewCode("InterpretedCode",0, buildFuncStepper)
		newFunc.tiipe = "LAMBDA"
		newFunc.subType="INTERPRETED"
		newFunc._arrayVal=f
		newFunc._line = e._line
		ne.dataStack=pushStack(ne.dataStack,newFunc)
		if USE_FUNCTION_CACHE && v._id > 0 {
		newFunc.codeStackConsume = len(f)
		ne._functionCache[v._id] = newFunc

		//fmt.Printf("Caching new function of length %v, id %v\ncache %v", newFunc.codeStackConsume, v._id, ne._functionCache)
		}
		return ne
	} else {
		//fmt.Printf("Building FUNCTION\n")
		v.environment = e.environment
		f=pushStack(f, v)
		ne=buildFunc(ne,f)
	}
	//fmt.Printf("Returning\n")
	return ne
}


	func (t *TagResponder) Eval(args *Args, reply *StatusReply) error {

			code := args.A

			var en = MakeEngine()
			en = en.RunFile("bootstrap.lib")
			emit(fmt.Sprintf("code: %v\n", code))

			en = en.RunString(code, "eval")
			var ret, _ = popStack(en.dataStack)

			var rethash = map[string]string{}
			rethash["test"]="worked"
			rethash["retval"]=ret.getSource()
			//fmt.Printf("return hash: %v\n", ret)
			for k,v := range ret._hashVal { rethash[k]=v.getString() }



	reply.Answer = rethash
	//reply.TagsToFilesHisto, reply.TopTags = sumariseDatabase()
	emit(fmt.Sprintln("Status handler complete"))
	return nil
}
type Args struct {
	A     string
	Limit int
}

type StatusReply struct {
	Answer map[string]string
}



type TagResponder int

// NewRPCRequest returns a new rpcRequest.

// rpcRequest represents a RPC request.
// rpcRequest implements the io.ReadWriteCloser interface.
type rpcRequest struct {
	r    io.Reader     // holds the JSON formated RPC request
	rw   io.ReadWriter // holds the JSON formated RPC response
	done chan bool     // signals then end of the RPC request
}

func NewRPCRequest(r io.Reader) *rpcRequest {
	var buf bytes.Buffer
	done := make(chan bool)
	return &rpcRequest{r, &buf, done}
}


// Read implements the io.ReadWriteCloser Read method.
func (r *rpcRequest) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

// Write implements the io.ReadWriteCloser Write method.
func (r *rpcRequest) Write(p []byte) (n int, err error) {
	r.done <- true
	return r.rw.Write(p)
}

// Close implements the io.ReadWriteCloser Close method.
func (r *rpcRequest) Close() error {
	return nil
}

// Call invokes the RPC request, waits for it to complete, and returns the results.
func (r *rpcRequest) Call() io.Reader {
	if debug {
		emit(fmt.Sprintf("Processing json rpc request\n"))
	}
	arith := new(TagResponder)

	server := rpc.NewServer()
	server.Register(arith)

	//server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	go server.ServeCodec(jsonrpc.NewServerCodec(r))
	//go jsonrpc.ServeConn(r)
	<-r.done
	//b := []byte{}
	//_, _ = r.rw.Read(b)
	if debug {
	    emit(fmt.Sprintln("Returning"))
    }
	return r.rw
}

func rpc_server(serverAddress string) {
	arith := new(TagResponder)

	server := rpc.NewServer()
	server.Register(arith)

	//server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	l, e := net.Listen("tcp", serverAddress)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	http.HandleFunc("/rpc", func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		res := NewRPCRequest(req.Body).Call()
		io.Copy(w, res)
	})

	cwd, _ := os.Getwd()
	emit(fmt.Sprintf("Serving /files/ from:%s\n", cwd))

	http.Handle("/", http.FileServer(http.Dir("throffweb")))
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(cwd))))


	go http.ListenAndServe(":80", nil)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		//log.Println("Got connection")
		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
		//log.Println("Sent response, probably")
	}

}



//Creates a new engine and populates it with the core functions
func MakeEngine() *Engine{

	e:=NewEngine()

		e=add(e, "IDEBUGOFF", NewCode("IDEBUGOFF", 0, func (e *Engine,c *Thingy) *Engine {
			interpreter_debug = false
		return e}))

		e=add(e, "CLEAROUTPUT", NewCode("CLEAROUTPUT", 0, func (e *Engine,c *Thingy) *Engine {
		    e.dataStack=pushStack(e.dataStack,NewString(clearOutput(), e.environment))
		return e}))


		e=add(e, "IDEBUGON", NewCode("IDEBUGON", 0, func (e *Engine,c *Thingy) *Engine {
			interpreter_debug = true
		return e}))

		e=add(e, "FORCEGC", NewCode("FORCEGC", 0, func (e *Engine,c *Thingy) *Engine {
			runtime.GC()
		return e}))


		e=add(e, "DEBUGOFF", NewCode("DEBUGOFF", 0, func (e *Engine,c *Thingy) *Engine {
			debug = false
		return e}))

		e=add(e, "DEBUGON", NewCode("DEBUGOFF", 0, func (e *Engine,c *Thingy) *Engine {
			debug = true
		return e}))


		e=add(e, "TROFF", NewCode("TROFF", 0, func (e *Engine,c *Thingy) *Engine {
			traceProg = false
		return e}))

		e=add(e, "TRON", NewCode("TRONS", 0, func (e *Engine,c *Thingy) *Engine {
			traceProg = true
		return e}))

		e=add(e, "ITROFF", NewCode("ITROFF", 0, func (e *Engine,c *Thingy) *Engine {
			interpreter_trace = false
		return e}))

		e=add(e, "ITRON", NewCode("ITROFF", 0, func (e *Engine,c *Thingy) *Engine {
			interpreter_trace = true
		return e}))

	e=add(e, "NULLSTEP", NewCode("NullStep", 0, func (e *Engine,c *Thingy) *Engine {
		emit(fmt.Sprintf("NullStep\n"))
		return e}))

	e=add(e, "DROP", NewCode("DROP", 1, func (ne *Engine,c *Thingy) *Engine {
		_, ne.dataStack = popStack(ne.dataStack)
		return ne}))

		e=add(e, "ZERO",  NewCode("ZERO", -1, func (ne *Engine,c *Thingy) *Engine {
		ne.dataStack=pushStack(ne.dataStack,NewString("0", e.environment))
		return ne}))

	e=add(e, "ROLL",  NewCode("ROLL", 1, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		var n,_ = strconv.ParseInt( el1.getSource(), 10, 32 )
		n = int64(len(ne.dataStack))-n-1
		v := ne.dataStack[n]
		ne.dataStack = append( ne.dataStack[:n],ne.dataStack[n+1:]...)
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne}))

	e=add(e, "PICK",  NewCode("PICK", 0, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		var n,_ = strconv.ParseInt( el1.getSource() , 10, 32 )
		n = int64(len(ne.dataStack))-n-1
		v := ne.dataStack[n]
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne}))

	e=add(e, "NUM2CHAR",  NewCode("NUM2CHAR", 0, func (ne *Engine,c *Thingy) *Engine {
		var v, el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		var n,_ = strconv.ParseInt( el1.getSource() , 10, 32 )
		v = NewString(fmt.Sprintf("%c", n), e.environment)
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne}))

	e=add(e, "GETLINE",  NewCode("GETLINE", 0, func (ne *Engine,c *Thingy) *Engine {
		var v *Thingy
		bio := bufio.NewReader(os.Stdin)
		line, _, _ := bio.ReadLine()
		v = NewString(string(line), e.environment)
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne}))

	e=add(e, "OPENFILE",  NewCode("OPENFILE", -1, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy

		el1, ne.dataStack = popStack(ne.dataStack)
		f,err := os.Open(el1.getString())
        if ( ! (err == nil) ) {
            return ne.RunString(fmt.Sprintf("THROW [ Could not open file %v: %v ] ", el1.getString(), err), "Internal Error")
        }

		reader := bufio.NewReaderSize(f, 999999)
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(f))
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(reader))
        return ne}))

	e=add(e, "OPENSQLITE",  NewCode("OPENSQLITE", 0, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy

		el1, ne.dataStack = popStack(ne.dataStack)
		db, err := sql.Open("sqlite3", el1.getString())
		if err != nil {
			log.Fatal(err)
		}

		ne.dataStack = pushStack(ne.dataStack, NewWrapper(db))
		return ne}))




	e=add(e, "QUERY",  NewCode("QUERY", 1, func (ne *Engine,c *Thingy) *Engine {
		var el1, querystring *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		querystring, ne.dataStack = popStack(ne.dataStack)
		db := el1._structVal.(*sql.DB)
		str := querystring.getString()

		rows, err := db.Query(str)

		if err != nil {
			emit(fmt.Sprintf("Error: Reading from table %v", err))
		}
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(rows))
		return ne}))

		e=add(e, "EXEC",  NewCode("EXEC", 3, func (ne *Engine,c *Thingy) *Engine {
		var el1, querystring,wrappedArgs *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		querystring, ne.dataStack = popStack(ne.dataStack)
		wrappedArgs, ne.dataStack = popStack(ne.dataStack)
		db := el1._structVal.(*sql.DB)
		stringArgs := []interface{}{}
		for _,v := range wrappedArgs._arrayVal {
			stringArgs = append(stringArgs, v.getString())
		}

		_,err := db.Exec(querystring.getString(), stringArgs...)

		if err != nil {
			emit(fmt.Sprintf("Error: exec failed: %v", err))
		}
		return ne}))

		e=add(e, "NEXTROW",  NewCode("NEXTROW", 0, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		rows := el1._structVal.(*sql.Rows)

		rows.Next()
			var name string
			rows.Scan(&name)

			    cols, err := rows.Columns()
    if err != nil {
        emit(fmt.Sprintln("Failed to get columns", err))
    }

    // Result is your slice string.
    rawResult := make([][]byte, len(cols))
    result := make([]string, len(cols))

    dest := make([]interface{}, len(cols)) // A temporary interface{} slice
    for i, _ := range rawResult {
        dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
    }

    rows.Next()
        err = rows.Scan(dest...)
        if err != nil {
            emit(fmt.Sprintln("Failed to scan row", err))

        }

        for i, raw := range rawResult {
            if raw == nil {
                result[i] = "\\N"
            } else {
                result[i] = string(raw)
            }
        }
			h:= NewHash()
			for i,v := range result {
				h._hashVal[cols[i]] = NewString(v,e.environment)

			}

			ne.dataStack = pushStack(ne.dataStack, h)
		return ne}))




	e=add(e, "CLOSEFILE",  NewCode("CLOSEFILE", 1, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy

		el1, ne.dataStack = popStack(ne.dataStack)
		f := el1._structVal.(*os.File)
		f.Close()
		return ne}))

	e=add(e, "MMAPFILE",  NewCode("MMAPFILE", -1, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy

		el1, ne.dataStack = popStack(ne.dataStack)
		f,_ := os.OpenFile(el1.getString(), os.O_RDWR, 0644)
		//info, _ :=os.Lstat(el1.getString())
		b, err := mmap.Map(f, mmap.RDWR, 0)
		if err != nil {
			emit(fmt.Sprintf("mmap failed: %v\n", err))

		}
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(f))
		bt := NewBytes(b, el1.environment)
		bt._structVal = f
		ne.dataStack = pushStack(ne.dataStack, bt)
		return ne}))



	e=add(e, "RUNSTRING",  NewCode("RUNSTRING", 9001, func (ne *Engine,c *Thingy) *Engine {
		var el1,env  *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		env, ne.dataStack = popStack(ne.dataStack)
		ne.environment = env
		ne = ne.RunString(el1.getString(), "runstring")
		return ne}))

		e=add(e, "READFILELINE",  NewCode("READFILELINE", 0, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		reader := el1._structVal.(*bufio.Reader)
		buff, _, ok  :=reader.ReadLine()
		var v *Thingy
		if ok == nil {
			v = NewString(string(buff), ne.environment)
		} else {
			v= NewBool(0)
		}
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne}))


		e=add(e, "THIN",  NewCode("THIN", 0, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		el2 := clone(el1)
		el2.share_parent_environment = true
		ne.dataStack = pushStack(ne.dataStack, el2)
		return ne}))

		e=add(e, "MACRO",  NewCode("MACRO", 0, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		el2 := clone(el1)
		el2.no_environment = true
		el2.share_parent_environment = true
		el2.environment = nil
		ne.dataStack = pushStack(ne.dataStack, el2)
		return ne}))

		e=add(e, "CALL",  NewCode("CALL", 1, func (ne *Engine,c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		el2 := clone(el1)
		if el2.tiipe =="LAMBDA" {
			el2.tiipe = "CODE"
		}
		ne.codeStack = pushStack(ne.codeStack, el2)
		ne.lexStack = pushStack(ne.lexStack, ne.environment)
		//engineDump(ne)
		return ne}))


	e=add(e, "EMIT",  NewCode("EMIT", 1, func (ne *Engine,c *Thingy) *Engine {
		var v *Thingy
		v, ne.dataStack = popStack(ne.dataStack)
		emit(fmt.Sprintf("%v", v.getString()))
		return ne}))


	e=add(e, "PRINTLN",  NewCode("PRINTLN", 1, func (ne *Engine,c *Thingy) *Engine {
		var v *Thingy
		v, ne.dataStack = popStack(ne.dataStack)
		//fmt.Printf("printing type: %v\n", v.tiipe)
		emit(fmt.Sprintf("%v\n", v.getString()))
		return ne}))


	e=add(e, "]",  NewCode("StartFunctionDef", 0, func (ne *Engine,c *Thingy) *Engine {
		ne._buildingFunc=true
		ne.dataStack=pushStack(ne.dataStack,c)
		return ne}))


	e=add(e, "[",  NewCode("BuildFuncFromStack", 9001, func (ne *Engine,c *Thingy) *Engine {
		ne._funcLevel+=1
		var f stack
		ne=buildFunc(ne,f)
		newFunc, _ := popStack(ne.dataStack)
		newFunc.environment = ne.environment
		return ne}))


	e=add(e, "DIRECTORY-LIST",  NewCode("DIRECTORY-LIST", 0, func (ne *Engine,c *Thingy) *Engine {
		var dir []os.FileInfo
        var aDir *Thingy
		aDir, ne.dataStack =popStack(ne.dataStack)
		dir,_ = ioutil.ReadDir(aDir.getString())
		var f stack
		for _,el := range dir { f=pushStack(f,NewString(el.Name(), e.environment))}
		c=NewArray(f)
		ne.dataStack=pushStack(ne.dataStack, c)
		return ne}))

	e=add(e, "SPLIT",  NewCode("SPLIT", 2, func (ne *Engine,c *Thingy) *Engine {
		var aString, aSeparator, aCount *Thingy
		aString, ne.dataStack =popStack(ne.dataStack)
		aSeparator, ne.dataStack =popStack(ne.dataStack)
		aCount, ne.dataStack =popStack(ne.dataStack)
		n, _ := strconv.ParseInt( aCount.getString(), 10, 32 )
		bits := strings.SplitN(aString.getString(), aSeparator.getString(), int(n))
		var f stack
		for _,el := range bits { f=pushStack(f,NewString(el, e.environment))}
		c=NewArray(f)
		ne.dataStack=pushStack(ne.dataStack, c)
		return ne}))

	e=add(e, "SAFETYON", NewCode("SAFETYON", 2, func (ne *Engine,c *Thingy) *Engine {
			ne._safeMode= true
		return ne}))

	e=add(e, ":", NewCode(":", 2, func (ne *Engine,c *Thingy) *Engine {
		var aName, aVal *Thingy
				defer func() {
        if r := recover(); r != nil {
            emit(fmt.Sprintln("Unable to set variable ",aName.getSource(), " because ",  r))
			engineDump(ne)
			os.Exit(1)
        }
    }()
		aName, ne.dataStack =popStack(ne.dataStack)
		aVal, ne.dataStack =popStack(ne.dataStack)
		env:=ne.environment
		if interpreter_debug {
			emit(fmt.Sprintf("Environment: %p - Storing %v in %v\n", env, aVal.getString(), aName.getString()))
		}

		prev, ok :=env._hashVal[aName.getString()]
		if ok  {
			if e._safeMode {
				emit(fmt.Sprintf("Warning:  mutating binding %v in %v at line %v(previous value %v)\n", aName.getString(), aName._filename, aName._line, prev.getString()))
				os.Exit(1)
			}
		}
		env._hashVal[aName.getString()] = aVal
		checkVal, checkOk :=env._hashVal[aName.getString()]
		if interpreter_debug {
		emit(fmt.Sprintf("Checked var %v, value is %v, in environment %p - %v\n",aName.getString(), checkVal, env, env ))
		}
		if !checkOk {panic("bind name failed!")}
		if ! (checkVal == aVal) {panic("bind name failed!")}
		if checkVal == nil {panic("bind name failed!")}
		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}

		return ne}))

	e=add(e, "REBIND", NewCode(":", 2, func (ne *Engine,c *Thingy) *Engine {
		var aName, aVal *Thingy
				defer func() {
        if r := recover(); r != nil {
            emit(fmt.Sprintln("Unable to set variable ",aName.getSource(), " because ",  r))
			engineDump(ne)
			os.Exit(1)
        }
    }()

		aName, ne.dataStack =popStack(ne.dataStack)
		aVal, ne.dataStack =popStack(ne.dataStack)
		env:=aName.environment
		if interpreter_debug {
			emit(fmt.Sprintf("Environment: %p - Storing %v in %v\n", env, aVal.getString(), aName.getString()))
		}

		_, ok :=env._hashVal[aName.getString()]
		if !ok  {
			if e._safeMode {
				emit(fmt.Sprintf("Warning:  Could not mutate: binding %v not found at line %v\n", aName.getString(), aName._line))
				os.Exit(1)
			}
		}
		env._hashVal[aName.getString()] = aVal
		_, ok =env._hashVal[aName.getString()]
		if !ok {panic("key not found")}
		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne}))


	e=add(e, "ENVIRONMENTOF", NewCode("ENVIRONMENTOF", 0, func (ne *Engine,c *Thingy) *Engine {
		var aName, aVal *Thingy
		aName, ne.dataStack =popStack(ne.dataStack)
		if interpreter_debug {
			emit(fmt.Sprintf("Environment: %p - Storing %v in %v\n", aName.environment, aVal.getString(), aName.getString()))
		}
		ne.dataStack=pushStack(ne.dataStack, ne.environment)

		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne}))


	e=add(e, "LOCATIONOF", NewCode("LOCATIONOF", 0, func (ne *Engine,c *Thingy) *Engine {
		var  aVal *Thingy
		aVal, ne.dataStack =popStack(ne.dataStack)
		if interpreter_debug {
			emit(fmt.Sprintf("Location: %v\n", aVal._line))
		}
		H := NewString(fmt.Sprintf("%v", aVal._line), c.environment)
		ne.dataStack=pushStack(ne.dataStack, H)

		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne}))


	e=add(e, "SETLEX", NewCode("SETLEX", 2, func (ne *Engine,c *Thingy) *Engine {
		var aName, aVal *Thingy
		aName, ne.dataStack =popStack(ne.dataStack)
		aVal, ne.dataStack =popStack(ne.dataStack)
		env := ne.environment
		//fmt.Printf("Storing %v in %v\n", aVal._source, aName._source)
		env._hashVal[aName.getString()] = aVal
		_, ok :=env._hashVal[aName.getString()]
		if !ok {panic("key not found in environment after set")}
		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne}))

	e=add(e, "SCRUBLEX", NewCode("SCRUBLEX", 1, func (ne *Engine,c *Thingy) *Engine {
		var aName *Thingy
		aName, ne.dataStack =popStack(ne.dataStack)
		env := ne.environment
		if interpreter_debug {
			emit(fmt.Sprintf("Scrubbing %v from %v\n", aName.getString(), env))
		}
		delete(env._hashVal, aName.getString())
		_, ok :=env._hashVal[aName.getString()]
		if ok {panic("key found in environment after set")}
		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne}))



	e=add(e, "GETLEX", NewCode("GETLEX", 0, func (ne *Engine,c *Thingy) *Engine {
		var aName *Thingy
		aName, ne.dataStack =popStack(ne.dataStack)
		//fmt.Printf("Fetching %v\n", aName.getString())

		aVal, ok :=ne.environment._hashVal[aName.getString()]
		if !ok {
			for k,v := range ne.environment._hashVal {emit(fmt.Sprintf("%v: %v\n", k,v))}
			emit(fmt.Sprintln("key not found "   , aName.getString()))
			panic("Key not found error")
		}

		ne.dataStack = pushStack(ne.dataStack, aVal)
		return ne}))


	e=add(e, "EQUAL", NewCode("EQUAL", 1, func (ne *Engine,c *Thingy) *Engine {
		var aVal,bVal *Thingy
		aVal, ne.dataStack =popStack(ne.dataStack)
		bVal, ne.dataStack =popStack(ne.dataStack)

		if (aVal.getString() == bVal.getString()) {
			ne.dataStack = pushStack(ne.dataStack, NewBool(1))
		} else {
			ne.dataStack = pushStack(ne.dataStack, NewBool(0))
		}
		return ne}))


	e=add(e, "IF", NewCode("IF", 3, func (ne *Engine,c *Thingy) *Engine {
		var testVal, trueBranch, falseBranch *Thingy
		testVal, ne.dataStack =popStack(ne.dataStack)
		trueBranch, ne.dataStack =popStack(ne.dataStack)
		falseBranch, ne.dataStack =popStack(ne.dataStack)

		ne.codeStack = pushStack(ne.codeStack, NewToken("CALL", nil))
		ne.lexStack= pushStack(ne.lexStack, ne.environment)

		if (testVal._intVal==1) {
			ne.codeStack = pushStack(ne.codeStack, trueBranch)
			ne.lexStack= pushStack(ne.lexStack, ne.environment)
		} else {
			ne.codeStack = pushStack(ne.codeStack, falseBranch)
			ne.lexStack= pushStack(ne.lexStack, ne.environment)
		}
		//engineDump(ne)
		return ne}))

	e=add(e, "NOT", NewCode("NOT", 0, func (ne *Engine,c *Thingy) *Engine {
		var aVal *Thingy
		aVal, ne.dataStack = popStack(ne.dataStack)
		aVal = clone(aVal)

		if aVal._intVal == 0 {
			aVal._intVal = 1
		} else {
			aVal._intVal = 0
		}
		ne.dataStack = pushStack(ne.dataStack, aVal)
		return ne}))

	e=add(e, "LESSTHAN", NewCode("LESSTHAN", 1, func (ne *Engine,c *Thingy) *Engine {
		var aVal,bVal *Thingy
		aVal, ne.dataStack =popStack(ne.dataStack)
		bVal, ne.dataStack =popStack(ne.dataStack)

		var a,_ = strconv.ParseFloat( aVal.getSource() , 32 )
		var b,_ = strconv.ParseFloat( bVal.getSource() , 32 )
		if (a < b) {
			ne.dataStack = pushStack(ne.dataStack, NewBool(1))
		} else {
			ne.dataStack = pushStack(ne.dataStack, NewBool(0))
		}
		return ne}))


	e=add(e, "THREAD", NewCode("THREAD", 1, func (ne *Engine,c *Thingy) *Engine {

		var  threadBranch *Thingy
		threadBranch, ne.dataStack =popStack(ne.dataStack)


		ne2:=cloneEngine(ne, true)
		ne2.codeStack=stack{}
		ne2.lexStack=stack{}
		ne2.dataStack=stack{}

		ne2.codeStack = pushStack(ne2.codeStack, NewToken("CALL", ne.environment))
		ne2.lexStack = pushStack(ne2.lexStack, ne.environment)

		ne2.codeStack = pushStack(ne2.codeStack, threadBranch)
		ne2.lexStack = pushStack(ne2.lexStack, ne.environment)
		go func () {run(ne2)}()

		return ne}))


	e=add(e, "SLEEP", NewCode("SLEEP", 1, func (ne *Engine,c *Thingy) *Engine {
			var el1 *Thingy
			el1, ne.dataStack = popStack(ne.dataStack)
			n, _ := strconv.ParseInt( el1.getSource(), 10, 64 )
			time.Sleep(time.Duration(n) * time.Millisecond)
		return ne}))


	e=add(e, "GETTYPE",  NewCode("GETTYPE", 0, func (ne *Engine,c *Thingy) *Engine {
		var v *Thingy
		v, ne.dataStack = popStack(ne.dataStack)
		ne.dataStack = pushStack( ne.dataStack, NewString ( v.tiipe, e.environment ) )
		return ne}))

	e=add(e, "SETTYPE",  NewCode("SETTYPE", 1, func (ne *Engine,c *Thingy) *Engine {
		var t, el *Thingy
		t, ne.dataStack = popStack(ne.dataStack)
		el, ne.dataStack = popStack(ne.dataStack)

		targetType := t.getString()
		el = clone(el)
		if targetType == "STRING" && ( el.tiipe == "CODE" || el.tiipe == "LAMBDA"){
			el._stringVal = el.getString() //Calculate the string representation of the array before we change the type
			el._source = el.getSource() //Calculate the string representation of the array before we change the type
		}
		if targetType == "STRING" && ( el.tiipe == "BOOLEAN"){
			el._stringVal = el.getString() //Calculate the string representation of the array before we change the type
			el._source = el.getSource() //Calculate the string representation of the array before we change the type
		}
		if targetType == "STRING" && ( el.tiipe == "ARRAY"){
			el._stringVal = el.getString() //Calculate the string representation of the array before we change the type
			el._source = el.getSource() //Calculate the string representation of the array before we change the type
		}
		if targetType == "CODE" && ( el.tiipe == "CODE" || el.tiipe == "LAMBDA"){
			el.arity = 0
		}
		el.tiipe = targetType
		ne.dataStack = pushStack(ne.dataStack, el  )
		return ne}))

	e=add(e, "->BYTES",  NewCode("->BTES", 0, func (ne *Engine,c *Thingy) *Engine {
		var t,el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)

		 t = NewBytes([]byte(el.getString()), el.environment)

		ne.dataStack = pushStack(ne.dataStack, t  )
		return ne}))

	e=add(e, "SPACE",  NewCode("SPACE", -1, func (ne *Engine,c *Thingy) *Engine {
		ne.dataStack=pushStack(ne.dataStack,NewString(" ", e.environment))
		return ne}))


	e=add(e, ".S",  NewCode(".S", 0, func (e *Engine,c *Thingy) *Engine {
		stackDump(e.dataStack)
		return e}))

	e=add(e, ".C",  NewCode(".C", 0, func (e *Engine,c *Thingy) *Engine {
		stackDump(e.codeStack)
		return e}))

	e=add(e, ".L",  NewCode(".S", 0, func (e *Engine,c *Thingy) *Engine {
		emit(fmt.Sprintln())
		//stackDump(e.codeStack)
		emit(fmt.Sprintln("lexstack"))
		stackDump(e.lexStack)
		emit(fmt.Sprintln())
		return e}))

	e=add(e, ".E",  NewCode(".S", 0, func (e *Engine,c *Thingy) *Engine {
		dumpEnv(e.environment)
		dumpEnv(c.environment)
		return e}))


	e=add(e, "ARRAYPUSH",  NewCode("ARRAYPUSH", 1, func (ne *Engine,c *Thingy) *Engine {
		var arr, el *Thingy
		arr, ne.dataStack = popStack(ne.dataStack)
		el, ne.dataStack = popStack(ne.dataStack)
		newarr := clone(arr)
		newarr._arrayVal = append(arr._arrayVal, el)

		ne.dataStack = pushStack(ne.dataStack, newarr  )
		return ne}))

	e=add(e, "NEWARRAY",  NewCode("NEWARRAY", -1, func (ne *Engine,c *Thingy) *Engine {
		var arr *Thingy
		arr = NewArray(stack{})
		ne.dataStack = pushStack(ne.dataStack, arr  )
		return ne}))



    e=add(e, "POPARRAY",  NewCode("POPARRAY", -1, func (ne *Engine,c *Thingy) *Engine {
		var arr, el *Thingy
		arr, ne.dataStack = popStack(ne.dataStack)
		var newarr *Thingy = clone(arr)
		newarr._arrayVal = nil
		el, newarr._arrayVal = popStack(arr._arrayVal)
		ne.dataStack = pushStack(ne.dataStack, newarr)
		ne.dataStack = pushStack(ne.dataStack, el)
		return ne}))

	e=add(e, "SHIFTARRAY",  NewCode("SHIFTARRAY", -1, func (ne *Engine,c *Thingy) *Engine {
		var arr*Thingy
		arr, ne.dataStack = popStack(ne.dataStack)
		el := arr._arrayVal[0]
		newarr := clone(arr)
		newarr._arrayVal = nil
		newarr._arrayVal = append(stack{}, arr._arrayVal[1:]...)
		ne.dataStack = pushStack(ne.dataStack, newarr)
		ne.dataStack = pushStack(ne.dataStack, el)
		return ne}))


	e=add(e, "UNSHIFTARRAY",  NewCode("UNSHIFTARRAY", 1, func (ne *Engine,c *Thingy) *Engine {
		var arr, el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		arr._arrayVal = append(stack{el},arr._arrayVal...)

		ne.dataStack = pushStack(ne.dataStack, arr  )
		return ne}))

	e=add(e, "GETARRAY",  NewCode("GETARRAY", 1, func (ne *Engine,c *Thingy) *Engine {
		var arr, el *Thingy

		defer func() {
        if r := recover(); r != nil {
            emit(fmt.Sprintf("Array out of bounds in getarray: index %v\n", arr, el.getSource()))
			engineDump(ne)
			os.Exit(1)
        }
    }()
		el, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		var n,_ = strconv.ParseInt( el.getSource(), 10, 64 )
		ret := arr._arrayVal[n]

		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne}))

	e=add(e, "GETBYTE",  NewCode("GETBYTE", 1, func (ne *Engine,c *Thingy) *Engine {
		var arr, el *Thingy

		el, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		var n,_ = strconv.ParseInt( el.getSource(), 10, 32 )
		ret := arr._bytesVal[n]

		ne.dataStack = pushStack(ne.dataStack, NewString(fmt.Sprintf("%c", ret), el.environment))
		return ne}))

	e=add(e, "GETSTRING",  NewCode("GETSTRING", 1, func (ne *Engine,c *Thingy) *Engine {
		var arr, el *Thingy

		el, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		var n,_ = strconv.ParseInt( el.getSource(), 10, 32 )
		s := arr.getString()
		var s1 string
		for index, r := range s {
			if index == int(n) {
				s1 = fmt.Sprintf("%c", r)
			}
		}
		ret := NewString(s1, arr.environment)
		ne.dataStack = pushStack(ne.dataStack, ret )
		return ne}))
	/*
	e=add(e, "SETSTRING",  NewCode("SETSTRING", 1, func (ne *Engine,c *Thingy) *Engine {
		var arr, el, val *Thingy

		el, ne.dataStack = popStack(ne.dataStack)
		val, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		var n,_ = strconv.ParseInt( el.getSource(), 10, 32 )
		s := arr.getString()
		s[n] = val.getString()
		ret := NewString(s, arr.environment)
		ne.dataStack = pushStack(ne.dataStack, ret )
		return ne}))
	*/
	e=add(e, "SETARRAY",  NewCode("SETARRAY", 2, func (ne *Engine,c *Thingy) *Engine {
		var arr, index, value *Thingy
		index, ne.dataStack = popStack(ne.dataStack)
		value, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		var n,_ = strconv.ParseInt( index.getSource(), 10, 32 )

		newarr := clone(arr)
		newarr._arrayVal = make(stack,len(arr._arrayVal),len(arr._arrayVal))
		copy(newarr._arrayVal, arr._arrayVal)

		newarr._arrayVal[n] = value

		ne.dataStack = pushStack(ne.dataStack, newarr)
		return ne}))

	e=add(e, "KEYVALS",  NewCode("KEYVALS", 0, func (ne *Engine,c *Thingy) *Engine {
		var arr, hash *Thingy
		hash, ne.dataStack = popStack(ne.dataStack)
		arr = NewArray(stack{})
		for k,v := range hash._hashVal {
			arr._arrayVal = append(arr._arrayVal, NewString(k, ne.environment), v)
		}

		ne.dataStack = pushStack(ne.dataStack, arr  )
		return ne}))





	e=add(e, "STRING-CONCATENATE",  NewCode("STRING-CONCATENATE", 1, func (ne *Engine,c *Thingy) *Engine {
		var s1, s2 *Thingy
		s1, ne.dataStack = popStack(ne.dataStack)
		s2, ne.dataStack = popStack(ne.dataStack)
		s3  := NewString(fmt.Sprintf("%s%s", s1.getString(),s2.getString()), ne.environment)

		ne.dataStack = pushStack(ne.dataStack, s3  )
		return ne}))

	/*e=add(e, "SWAP",  NewCode("SWAP", 1, func (ne *Engine,c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		ne.dataStack = pushStack(ne.dataStack, el  )
		ne.dataStack = pushStack(ne.dataStack, el1  )
		return ne}))
		*/

	e=add(e, "ADD",  NewCode("ADD", 1, func (ne *Engine,c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
        var v1, v2  *big.Float
        var v3 big.Float
         v1,_,_ = big.ParseFloat( el.getSource(), 10, precision, big.ToZero  )
        v1=v1.SetPrec(precision)
        v2,_,_ = big.ParseFloat( el1.getSource(), 10, precision, big.ToZero )
        v2=v2.SetPrec(precision)
        v3.SetPrec(0)
        v3=*v3.Add(v1,v2)

		var t *Thingy = NewString(fmt.Sprintf("%v", v3.Text('g', int(precision))), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t  )
		return ne}))

	e=add(e, "FLOOR",  NewCode("FLOOR", 0, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)

		var v1,_ = strconv.ParseFloat( el.getSource() , 32 )

		var t *Thingy = NewString(fmt.Sprintf("%v", math.Floor(v1)), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t  )
		return ne}))


	e=add(e, "SETHASH",  NewCode("SETHASH", 2, func (ne *Engine,c *Thingy) *Engine {
		var key, val, hash, newhash *Thingy
		key, ne.dataStack = popStack(ne.dataStack)
		val, ne.dataStack = popStack(ne.dataStack)
		hash, ne.dataStack = popStack(ne.dataStack)
		newhash = clone(hash)
		newhash._hashVal = cloneMap(hash._hashVal)
		newhash._hashVal[key.getString()]=val
		ne.dataStack = pushStack(ne.dataStack, newhash  )
		return ne}))

	e=add(e, "GETHASH",  NewCode("GETHASH", 1, func (ne *Engine,c *Thingy) *Engine {
		var key, val, hash *Thingy
		key, ne.dataStack = popStack(ne.dataStack)
		hash, ne.dataStack = popStack(ne.dataStack)
		val = hash._hashVal[key.getString()]
		if val == nil {
			emit(fmt.Sprintf("Warning: %v not found in hash%v\n\nCreating empty value\n", key.getString(), hash._hashVal))
			val = NewString(fmt.Sprintf("UNDEFINED:%v", key.getString()), ne.environment)
		}
		ne.dataStack = pushStack(ne.dataStack, val  )
		return ne}))

	e=add(e, "NEWHASH",  NewCode("NEWHASH", -1, func (ne *Engine,c *Thingy) *Engine {
		var hash *Thingy = NewHash()
		ne.dataStack = pushStack(ne.dataStack, hash  )
		return ne}))





	e=add(e, "SUB",  NewCode("SUB", 1, func (ne *Engine,c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		var v1,_ = strconv.ParseFloat( el.getSource() , 64 )
		var v2,_ = strconv.ParseFloat( el1.getSource() , 64 )
		var t *Thingy = NewString(fmt.Sprintf("%v", v1-v2), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t  )
		return ne}))

	e=add(e, "MULT",  NewCode("MULT", 1, func (ne *Engine,c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		var v1,_ = strconv.ParseFloat( el.getSource() , 32 )
		var v2,_ = strconv.ParseFloat( el1.getSource() , 32 )
		var t *Thingy = NewString(fmt.Sprintf("%v", v1*v2), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t  )
		return ne}))

	e=add(e, "MODULO",  NewCode("MODULO", 1, func (ne *Engine,c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		var v1,_ = strconv.ParseFloat( el.getSource() , 32 )
		var v2,_ = strconv.ParseFloat( el1.getSource() , 32 )
		var t *Thingy = NewString(fmt.Sprintf("%v", math.Mod(v1,v2)), el.environment)
		ne.dataStack = pushStack(ne.dataStack, t  )
		return ne}))


	e=add(e, "LN",  NewCode("LN", 0, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		var v1,_ = strconv.ParseFloat( el.getSource() , 32 )
		var t *Thingy = NewString(fmt.Sprintf("%v", math.Log2(v1)), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne}))



		e=add(e, "DIVIDE",  NewCode("DIVIDE", 1, func (ne *Engine,c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
        var v1, v2  *big.Float
        var v3 big.Float
         v1,_,_ = big.ParseFloat( el.getSource(), 10, precision, big.ToZero  )
        v1=v1.SetPrec(precision)
        v2,_,_ = big.ParseFloat( el1.getSource(), 10, precision, big.ToZero )
        v2=v2.SetPrec(precision)
        v3.SetPrec(0)
        v3=*v3.Quo(v1,v2)
		var t *Thingy = NewString(fmt.Sprintf("%v", v3.Text('g', int(precision))), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t  )
		return ne}))

	e=add(e, "TIMESEC",  NewCode("TIMESEC", -1, func (ne *Engine,c *Thingy) *Engine {
		var t *Thingy = NewString(fmt.Sprintf("%v", int32(time.Now().Unix())), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t  )
		return ne}))

		e=add(e, "TOK",  NewCode("TOK", -1, func (ne *Engine,c *Thingy) *Engine {
		var el,lex *Thingy
		el, ne.codeStack = popStack(ne.codeStack)
		lex, ne.lexStack= popStack(ne.lexStack)
		el1 := clone(el)
		el1.environment = lex
		ne.dataStack = pushStack(ne.dataStack, el1)
		return ne}))


		e=add(e, "GETFUNCTION",  NewCode("GETFUNCTION", 0, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		val, ok := nameSpaceLookup(ne, el)
		if ok {
			ne.dataStack = pushStack(ne.dataStack, val)
		} else {
			ne.dataStack = pushStack(ne.dataStack, NewToken("FALSE", ne.environment))
		}
		//stackDump(ne.dataStack)
		return ne}))





	e=add(e, "RPCSERVER",  NewCode("RPCSERVER", 0, func (ne *Engine,c *Thingy) *Engine {
		rpc_server("127.0.0.1:80")
		return ne}))

		e=add(e, "TCPSERVER",  NewCode("TCPSERVER", 1, func (ne *Engine,c *Thingy) *Engine {
			var server, port *Thingy
			server, ne.dataStack = popStack(ne.dataStack)
			port, ne.dataStack = popStack(ne.dataStack)
// Listen on TCP port 2000 on all interfaces.
    l, err := net.Listen("tcp", fmt.Sprintf("%s:%s",server.getString(), port.getString()))
    if err != nil {
        log.Fatal(err)
    }
    defer l.Close()
    for {
        // Wait for a connection.
        conn, err := l.Accept()
        if err != nil {
            log.Fatal(err)
        }
        // Handle the connection in a new goroutine.
        // The loop then returns to accepting, so that
        // multiple connections may be served concurrently.
        go func(c net.Conn) {
			t := NewWrapper(c)
			ne.dataStack = pushStack(ne.dataStack, t)
            ne.RunString("CALL SWAP", "TCPSERVER provided handler")
            // Echo all incoming data.
            //io.Copy(c, c)
            // Shut down the connection.
            c.Close()
        }(conn)
    }
		return e}))
		e=add(e, "OPENSOCKET",  NewCode("OPENSOCKET", 1, func (ne *Engine,c *Thingy) *Engine {
			var server, port *Thingy
			server, ne.dataStack = popStack(ne.dataStack)
			port, ne.dataStack = popStack(ne.dataStack)
      conn, err := net.Dial("tcp", fmt.Sprintf("%v:%v", server.getString(), port.getString()))
			emit(fmt.Sprintf("%v",err))
			t := NewWrapper(conn)
			ne.dataStack = pushStack(ne.dataStack, t)
			return ne
}))

e=add(e, "PRINTSOCKET",  NewCode("PRINTSOCKET", 2, func (ne *Engine,c *Thingy) *Engine {
	var message, conn *Thingy
	message, ne.dataStack = popStack(ne.dataStack)
	conn, ne.dataStack = popStack(ne.dataStack)
    fmt.Fprintf(conn._structVal.(io.Writer), message.getString())
	return ne
}))


e=add(e, "READSOCKETLINE",  NewCode("READSOCKETLINE", 0, func (ne *Engine,c *Thingy) *Engine {
	var server *Thingy
	server, ne.dataStack = popStack(ne.dataStack)
  var conn net.Conn
	conn = server._structVal.(net.Conn)
  message, _ := bufio.NewReader(conn).ReadString('\n')
	ret := NewString(message,ne.environment)
	ne.dataStack = pushStack(ne.dataStack, ret)
	return ne
}))

	e=add(e, "HTTPSERVER",  NewCode("HTTPSERVER", 0, func (ne *Engine,c *Thingy) *Engine {
		var path, callback *Thingy
		path, ne.dataStack = popStack(ne.dataStack)
		callback, ne.dataStack = popStack(ne.dataStack)

		http.HandleFunc(path.getString(), func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			var code = r.Form["code"]
			var en = MakeEngine()
			en = en.RunFile("bootstrap.lib")
			emit(fmt.Sprintf("code: %v\n", strings.Join(code, "")))
			en.dataStack = pushStack(en.dataStack, NewString(strings.Join(code, ""), en.environment))
			en.codeStack = pushStack(en.codeStack, callback)
			en.lexStack = pushStack(en.lexStack, ne.environment)
			en = en.Run()
			var ret, _ = popStack(en.dataStack)
			fmt.Fprintf(w, "Hello, %q, %q, %v",  callback.getString(), html.EscapeString(r.URL.Path), ret.getString())
		})
			cwd, _ := os.Getwd()
		emit(fmt.Sprintf("Serving /resources/ from:%s\n", cwd))
		http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir(cwd))))
		http.ListenAndServe(":80", nil)
		return ne}))

	e=add(e, "WEBSOCKETCLIENT",  NewCode("WEBSOCKETCLIENT", 0, func (ne *Engine,c *Thingy) *Engine {
        var url, protocol, origin *Thingy
		var q1, q2  *Thingy
		url, ne.dataStack = popStack(ne.dataStack)
		protocol, ne.dataStack = popStack(ne.dataStack)
		origin, ne.dataStack = popStack(ne.dataStack)
		q1, ne.dataStack = popStack(ne.dataStack)
		q2, ne.dataStack = popStack(ne.dataStack)

        wqueue := q1._structVal.(chan *Thingy)
        rqueue := q2._structVal.(chan *Thingy)

    ws, err := websocket.Dial(url.getString(), protocol.getString(), origin.getString())
    if err != nil {
        log.Fatal(err)
    }
        go func () {
            for {
                var msg = make([]byte, 512)
                var n int
                if n, err = ws.Read(msg); err != nil {
                    log.Fatal(err)
                }
                //fmt.Printf("Received: %s.\n", msg[:n])
                rqueue<-NewBytes(msg[:n], ne.environment)
        }}()

        go func () {
            for {
                    msg := <- wqueue
                        emit(fmt.Sprintf("Sending %v\n", msg.getString()))
                        if _, err := ws.Write([]byte(msg.getString())); err != nil {
                    log.Fatal(err)
                }
            }
        }()

        ne.dataStack = pushStack(ne.dataStack, NewString("", e.environment))
		return ne}))


		e=add(e, "RPCSERVER",  NewCode("RPCSERVER", 0, func (ne *Engine,c *Thingy) *Engine {
		rpc_server("127.0.0.1:80")
		return ne}))


	e=add(e, "GETWWW",  NewCode("GETWWW", 0, func (ne *Engine,c *Thingy) (re *Engine) {
		var path *Thingy
		path, ne.dataStack = popStack(ne.dataStack)
		defer func() {
        if r := recover(); r != nil {
            emit(fmt.Sprintln("Failed to retrieve ",path.getSource(), " because ",  r))
			ne.dataStack = pushStack(ne.dataStack, NewString("", e.environment))
			re=ne
        }
    }()
		res, err := http.Get(path.getString())
	if err != nil {
		log.Println(err)
	}
	robots, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Println(err)
	}

	ne.dataStack = pushStack(ne.dataStack, NewString(string(robots), ne.environment))

		return ne}))





	e=add(e, "EXIT",  NewCode("EXIT", 0, func (ne *Engine,c *Thingy) *Engine {
		/*for f, m := range ne._heatMap {
			emit(fmt.Sprintln("Hotspots in file ", f))
			for i, v := range m {
				emit(fmt.Sprintf("%d: %d\n", i,v))
			}
		}*/
		emit(fmt.Sprintln("Goodbye"))
		os.Exit(0)
		return e}))


		e=add(e, "LENGTH",  NewCode("LENGTH", 0, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		var val int
		if el.tiipe =="ARRAY" || el.tiipe =="LAMBDA" || el.tiipe =="CODE" {
			val = len(el._arrayVal)
		}
		if el.tiipe =="STRING" {
			val = len(el.getString())
		}
		if el.tiipe =="BYTES" {
			val = len(el._bytesVal)
		}
		ne.dataStack = pushStack(ne.dataStack, NewString(fmt.Sprintf("%v", val), ne.environment))
		//stackDump(ne.dataStack)
		return ne}))


		e=add(e, "NEWQUEUE",  NewCode("NEWQUEUE", -1, func (ne *Engine,c *Thingy) *Engine {
		q := make(chan *Thingy, 1000)
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(q))
		//stackDump(ne.dataStack)
		return ne}))

		e=add(e, "WRITEQ",  NewCode("WRITEQ", 2, func (ne *Engine,c *Thingy) *Engine {
			var el,el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)

		q := el._structVal.(chan *Thingy)

		q <- el1
		//ne.dataStack = pushStack(ne.dataStack, NewWrapper(q))
		//stackDump(ne.dataStack)
		return ne}))

		e=add(e, "READQ",  NewCode("READQ", 0, func (ne *Engine,c *Thingy) *Engine {
			var el,el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)


		q := el._structVal.(chan *Thingy)
		el1 = <- q

		ne.dataStack = pushStack(ne.dataStack, el1)
		//stackDump(ne.dataStack)
		return ne}))

		e=add(e, "ARP",  NewCode("ARP", 0, func (ne *Engine,c *Thingy) *Engine {
			var el,el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)


		ifi,_ := net.InterfaceByName(el.getString())
		emit(fmt.Sprintf("%v\n", ifi))
		arp,_ := arp.NewClient(ifi)
		s := net.ParseIP(el1.getString())
		emit(fmt.Sprintf("%V\n", s))
		addr, _ := arp.Resolve(s)

		ne.dataStack = pushStack(ne.dataStack, NewString(string(addr), nil))
		return ne}))

		e=add(e, "DNS.CNAME",  NewCode("DNS.CNAME", 0, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		r,_ := net.LookupCNAME(el.getString())
		ne.dataStack = pushStack(ne.dataStack, NewString(string(r), nil))
		return ne}))

		e=add(e, "DNS.HOST",  NewCode("DNS.HOST", 0, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		r,_ := net.LookupHost(el.getString())
		a := fmt.Sprintf("->ARRAY [ %v  ]", strings.Join(r, " "))
		ne = ne.RunString(a, "DNS.HOST")
		//ne.dataStack = pushStack(ne.dataStack, NewString(string(a), nil))
		return ne}))

		e=add(e, "DNS.TXT",  NewCode("DNS.TXT", 0, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		r,_ := net.LookupTXT(el.getString())
		a := fmt.Sprintf("->ARRAY [ %v  ]", strings.Join(r, " "))
		ne = ne.RunString(a, "DNS.TXT")
		//ne.dataStack = pushStack(ne.dataStack, NewString(string(a), nil))
		return ne}))

		e=add(e, "DNS.REVERSE",  NewCode("DNS.REVERSE", 0, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		r,_ := net.LookupAddr(el.getString())
		a := fmt.Sprintf("->ARRAY [ %v  ]", strings.Join(r, " "))
		ne = ne.RunString(a, "DNS.REVERSE")
		//ne.dataStack = pushStack(ne.dataStack, NewString(string(a), nil))
		return ne}))


		e=add(e, "CALL/CC",  NewCode("CALL/CC", -1, func (ne *Engine,c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		cc := NewWrapper(ne)
		cc._engineVal=ne

		ne = cloneEngine(ne, true)
		ne.codeStack = pushStack(ne.codeStack, NewToken("CALL", nil))
		ne.lexStack= pushStack(ne.lexStack, ne.environment)

		ne.dataStack = pushStack(ne.dataStack, cc)
		ne.dataStack = pushStack(ne.dataStack, el)


		return ne}))

		e=add(e, "ACTIVATE/CC",  NewCode("ACTIVATE/CC", 9999, func (ne *Engine,c *Thingy) *Engine {
		var el, el1 *Thingy

		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		ne = el._structVal.(*Engine)

		ne.dataStack = pushStack(ne.dataStack, el1)
		return ne}))


		e=add(e, "INSTALLDYNA",  NewCode("INSTALLDYNA", 2, func (ne *Engine,c *Thingy) *Engine {
            var el, err *Thingy
            err, ne.dataStack = popStack(ne.dataStack)
            el, ne.dataStack = popStack(ne.dataStack)

            var new_env = ne.environment
            var errStack = append(ne.dyn, err)
            new_env.errorChain = errStack
            ne.lexStack= pushStack(ne.lexStack, new_env)
            ne.codeStack = pushStack(ne.codeStack, el)
		    return ne
        }))
		e=add(e, "ERRORHANDLER",  NewCode("ERRORHANDLER", -1, func (ne *Engine,c *Thingy) *Engine {
            var errHandler *Thingy
            var new_env = ne.environment
            errHandler , _ = popStack(new_env.errorChain)
            ne.dataStack = pushStack(ne.dataStack, errHandler)
		    return ne
        }))
		e=add(e, "DUMP",  NewCode("DUMP", 1, func (ne *Engine,c *Thingy) *Engine {
            var el *Thingy
            el, ne.dataStack = popStack(ne.dataStack)

            ret := NewString(fmt.Sprintf("%v", el), nil)
            ne.dataStack = pushStack(ne.dataStack, ret)
		    return ne
        }))
        e=add(e, "STARTPROCESS",  NewCode("STARTPROCESS", 1, func (ne *Engine,c *Thingy) *Engine {
            var el, el_arr *Thingy
            el, ne.dataStack = popStack(ne.dataStack)
            el_arr, ne.dataStack = popStack(ne.dataStack)

            var argv = []string{el.getString()}
            //fmt.Printf("$V", el_arr._arrayVal)
            for _,v := range el_arr._arrayVal {
                argv = append(argv, v.getString())
            }
            //fmt.Printf("$V", argv)
            attr :=  os.ProcAttr{
                Files:  []*os.File{os.Stdin, os.Stdout, os.Stderr},
            }
            proc, err := os.StartProcess(el.getString(), argv, &attr)
            running := NewWrapper(proc)

            var ret *Thingy
            if err == nil {
                ret = running
            } else {
                ret = NewBool(0)
            }

            ne.dataStack = pushStack(ne.dataStack, ret)
            return ne
        }))
        e=add(e, "KILLPROC",  NewCode("KILLPROC", 1, func (ne *Engine,c *Thingy) *Engine {
            var el *Thingy
            el, ne.dataStack = popStack(ne.dataStack)
            proc := el._structVal.(*os.Process)
            err := proc.Kill()
            var ret *Thingy

            if (err == nil) {
                ret = NewBool(0)
            } else {
                ret = NewString(fmt.Sprintf("%v", err), nil)
            }

            ne.dataStack = pushStack(ne.dataStack, ret)
            return ne
        }))
        e=add(e, "RELEASEPROC",  NewCode("RELEASEPROC", 1, func (ne *Engine,c *Thingy) *Engine {
            var el *Thingy
            el, ne.dataStack = popStack(ne.dataStack)
            proc := el._structVal.(*os.Process)
            err := proc.Release()
            var ret *Thingy

            if (err == nil) {
                ret = NewBool(0)
            } else {
                ret = NewString(fmt.Sprintf("%v", err), nil)
            }

            ne.dataStack = pushStack(ne.dataStack, ret)
            return ne
        }))

        e=add(e, "WAITPROC",  NewCode("WAITPROC", 1, func (ne *Engine,c *Thingy) *Engine {
            var el *Thingy
            el, ne.dataStack = popStack(ne.dataStack)
            proc := el._structVal.(*os.Process)
            procState, err := proc.Wait()
            var ret *Thingy

            if (err == nil) {
                ret = NewString(fmt.Sprintf("%v\nPid: %v\nSystemTime: %v\nUserTime: %v\nSuccess: %v", procState.String(), procState.Pid(), procState.SystemTime(), procState.UserTime(), procState.Success()), nil)
            } else {
                ret = NewString(fmt.Sprintf("%v\nPid: %v\nSystemTime: %v\nUserTime: %v\nSuccess: %vError: %v", procState.String(), procState.Pid(), procState.SystemTime(), procState.UserTime(), procState.Success(), err), nil)
            }

            ne.dataStack = pushStack(ne.dataStack, ret)
            return ne
        }))

        e=add(e, "BYTE2STR",  NewCode("BYTE2STR", 0, func (ne *Engine,c *Thingy) *Engine {
            var el *Thingy
            el, ne.dataStack = popStack(ne.dataStack)
            var b []byte = el._bytesVal
            var str string  = string(b[:len(b)])
            ne.dataStack = pushStack(ne.dataStack, NewString(str, ne.environment))
            return ne
        }))



		e=add(e, "CLEARSTACK",  NewCode("CLEARSTACK", 9999, func (ne *Engine,c *Thingy) *Engine {
            ne.dataStack 	=	stack{}				//The argument stack
			ne.dyn			=	stack{}               //The current dynamic environment
			ne.codeStack	=	stack{}				//The future of the program
			ne.lexStack		=	stack{}

		    return ne
        }))
		e=add(e, "SIN",  NewCode("SIN", 0, func (ne *Engine,c *Thingy) *Engine {
            var arg *Thingy
            arg , ne.dataStack = popStack(ne.dataStack)
			var in,_ = strconv.ParseFloat( arg.getSource() , 32 )
			var ret = math.Sin(in)
            ne.dataStack = pushStack(ne.dataStack, NewString(fmt.Sprintf("%v", ret),ne.environment))
		    return ne
        }))
      e=add(e, "MAKEJIT",  NewCode("MAKEJIT", -1, func (ne *Engine,c *Thingy) *Engine {
          //var el, el_arr *Thingy

          s := tcc.New()

          if err := s.Compile(`
            float jit_func(float a, float b) {
                return a*b;
            }`); err != nil {
            log.Fatal(err)
          }
          running := NewWrapper(s)

          var ret *Thingy
          ret = running

          ne.dataStack = pushStack(ne.dataStack, ret)
          return ne
        }))

      e=add(e, "RUNJIT",  NewCode("RUNJIT", 0, func (ne *Engine,c *Thingy) *Engine {
          var jitwrap *Thingy

          jitwrap , ne.dataStack = popStack(ne.dataStack)
          jit := jitwrap._structVal.(*tcc.State)

          a:= C.float(2.0)
          b:= C.float(3.0)

          p, err := jit.Symbol("jit_func")
          if err != nil {
            log.Fatal(err)
          }

          n := float64(C.call(C.jitfunc(unsafe.Pointer(p)), a, b))


          ne.dataStack=pushStack(ne.dataStack,NewString(fmt.Sprintf("%v", n), e.environment))
          return ne
        }))



	//fmt.Println("Done")
	return e
}
