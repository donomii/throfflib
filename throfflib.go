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

//import "unsafe"
//import "github.com/thinxer/go-tcc"
import "fmt"
import "strings"
import "io/ioutil"
import "strconv"
import "log"
import "io"
import (
	_ "github.com/mattn/go-sqlite3"
)
import "github.com/chzyer/readline"

var precision uint = 256
var interpreter_debug = true
var interpreter_trace = false
var traceProg = false
var debug = false
var USE_FUNCTION_CACHE = false
var seqID = 0 //Every instruction in the program gets a unique number, used by the optimiser and similar tasks

type StepFunc func(*Engine, *Thingy) *Engine

// The fundamental unit of data.  In this engine, arrays, hashes, strings and numbers are all "Thingy"s
type Thingy struct {
	annotations map[string]Thingy
	_stub       StepFunc //The native code function that will perform the next command in the program
	tiipe       string   //The tiipe is the user-visible type.  Tiipes are very flexible especially string<->token and code<->array
	subType     string   //The interpreter visible type.  Almost always "NATIVE" or "INTERPRETED"
	userType    *Thingy  //A purely user defined type.  Can be anything
	_source     string   //A string that, in theory, could be eval-ed to recreate the data structure.  Also used for if statement comparisons
	environment *Thingy  //Functions carry a reference to their parent environment, which allows us to create a new environment for the function to run in each time it is called
	errorChain  stack    //Error handlers are functions that are kept on a stack
	//_parentEnvironment *Thingy
	lock                     *Thingy
	_structVal               interface{}
	_stringVal               string
	_arrayVal                stack
	_engineVal               *Engine
	_hashVal                 map[string]*Thingy
	_llVal                   *ll_t
	_bytesVal                []byte
	_note                    string
	arity                    int    //The number of arguments this function pops off the stack, minus the number of return results`
	_intVal                  int    //Currently used for booleans, will also be used to cache string->int conversions
	_id                      int    //Every token gets a unique id number, this allows us to do JIT on the code, among other tricks
	_line                    int    //Line number of instruction
	_filename                string //The source code file this instruction was written in
	codeStackConsume         int    //When the functions are cached, we need to know how long they are, and skip the build phase by popping this many Thingys off the code stack
	immutable                bool
	share_parent_environment bool //Instead of creating a new lexical pad (environment to store variables), use the surrounding lexical environment.  This allows e.g. if statements to affect variables in the surrounding function
	no_environment           bool //AKA macro.  Instead of using its own lexical environment, this function will use the lexical environment that it is INVOKED in.  Required for doing complicated bind operations, probably a bad idea for anything else

}

type stack []*Thingy
type Engine struct {
	//previousEngine	*Engine
	environment    *Thingy //The current lexical environment
	dataStack      stack   //The argument stack
	dyn            stack   //The current dynamic environment
	codeStack      stack   //The future of the program
	lexStack       stack
	_buildingFunc  bool //do we run functions or just shift them to the data stack?
	_funcLevel     int
	_prevLevel     int
	_functionCache map[int]*Thingy //Cache built functions here to avoid expensive rebuilds
	_line          int
	_heatMap       map[string][]int
	_safeMode      bool
	LastFunction   *Thingy
}

func (t *Thingy) setString(val string) {
	t._stringVal = val
	t._source = val
}
func (t *Thingy) setStub(val StepFunc) {
	t._stub = val
	t._source = "A function added by setStub"
	t._note = "A function added by setStub"
}

func cloneMap(m map[string]*Thingy) map[string]*Thingy {
	//fmt.Printf("Cloning map %v\n\n", m )
	var nm = make(map[string]*Thingy, 1000)
	for k, v := range m {
		nm[k] = v
	}
	return nm
}

//Builds a string that, when EVALed, will recreate the data structure
//The string might not be an exact representation of the data, but should recreate it perfectly
func (t *Thingy) getSource() string {
	if t.subType == "NATIVE" {
		return t._source
	}

	if t.tiipe == "ARRAY" {
		var accum string = "A[ "
		for _, el := range t._arrayVal {
			accum = fmt.Sprintf("%v STRING [ %v ]", accum, el._source)
		}
		return fmt.Sprintf("%v ]A", accum)
	}
	if t.tiipe == "CODE" || t.tiipe == "LAMBDA" {
		var accum string = "[ "
		for _, el := range t._arrayVal {
			accum = fmt.Sprintf("%v %v", accum, el._source)
		}
		return fmt.Sprintf("%v ]", accum)
	}

	if t.tiipe == "HASH" {
		var accum string = "H[ "
		for k, v := range t._hashVal {
			accum = fmt.Sprintf("%s %s %s", accum, k, v.getSource())
		}
		return fmt.Sprintf("%v ]H", accum)
	}

	// 		if t.tiipe == "STRING" {
	// 		return fmt.Sprintf("STRING [ %v ]", t._stringVal)
	// 	}

	return t._source
}

//Builds a string for display to the user.  Probably can't be used to re-create the original data structure
func (t *Thingy) GetString() string {
	if t.tiipe == "ARRAY" {
		var accum string = ""
		for i, el := range t._arrayVal {
			if i == 0 {
				accum = fmt.Sprintf("%v", el.GetString())
			} else {
				accum = fmt.Sprintf("%v %v", accum, el.GetString())
			}
		}
		return accum
	}

	if t.tiipe == "CODE" || t.tiipe == "LAMBDA" {
		var accum string = ""
		for i, el := range t._arrayVal {
			if i == 0 {
				accum = fmt.Sprintf("%v", el.GetString())
			} else {
				accum = fmt.Sprintf("%v %v", accum, el.GetString())
			}
		}
		return accum
	}

	if t.tiipe == "HASH" {
		var accum string = "{ "
		for k, v := range t._hashVal {
			accum = fmt.Sprintf("%s, %s -> %s", accum, k, v.GetString())
		}
		return fmt.Sprintf("%v }", accum)
	}

	if t.tiipe == "BOOLEAN" {
		if t._intVal == 0 {
			return "FALSE"
		} else {
			return "TRUE"
		}
	}

	return t._stringVal
}

func actualClone(t Thingy) *Thingy {
	return &t
}

func clone(t *Thingy) *Thingy {
	//fmt.Printf("Cloning thingy %v\n\n", t )
	return actualClone(*t)
}

//The engine is cloned at each step
func actualCloneEngine(t Engine) *Engine { return &t }
func cloneEngine(t *Engine, immutable bool) *Engine {
	//fmt.Printf("Cloning engine %v\n\n", t )
	//engineDump(t)
	newt := actualCloneEngine(*t)
	if immutable {
		newt.environment = cloneEnv(t.environment)
	} else {
		newt.environment = t.environment
	}
	//newt.previousEngine=t  //This is a memory leak

	return newt
}

func cloneEnv(env *Thingy) *Thingy {
	newEnv := clone(env)
	return newEnv
}

func add(e *Engine, s string, t *Thingy) *Engine {
	ne := cloneEngine(e, false)
	t._note = s
	t._source = s
	ne.environment._llVal = ll_add(ne.environment._llVal, s, t)
	//t.environment = ne.environment
	t.share_parent_environment = true
	t.no_environment = true
	return ne
}

func newThingy() *Thingy {
	t := new(Thingy)
	//The default action is to push the thing on to the data stack
	t._stub = func(e *Engine, c *Thingy) *Engine {
		ne := cloneEngine(e, false)
		ne.dataStack = pushStack(ne.dataStack, c)
		return ne
	}
	return t
}

//This is the native code called when the engine encounters a token.  It looks up the token and replaces it with the correct value.
//It also builds functions when it spots the function brackets [ ]
//This function got out of hand, and should be broken up
func tokenStepper(e *Engine, c *Thingy) *Engine {
	ne := cloneEngine(e, false) //FIXME e should already be a clone?
	//Are we in function-building mode?
	ne._prevLevel = ne._funcLevel
	if c.getSource() == "[" {
		ne._funcLevel -= 1
	} //Finish a (possibly nested) function
	if c.getSource() == "]" {
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
			for i = 0; i <= f.codeStackConsume; i++ {
				_, ne.codeStack = popStack(ne.codeStack)
				_, ne.lexStack = popStack(ne.lexStack)
			}
			//fmt.Printf("Skip complete\n")
			//engineDump(ne)
			return ne
		} else {
			ne._funcLevel += 1
		}
	} //Start a (possibly nested) function
	//fmt.Printf("TOKEN: in function level: %v\n", ne._funcLevel)
	if ne._funcLevel < 0 {
		emit(fmt.Sprintf("Unmatched [ at line %v\n", c._line))
		engineDump(ne)
		panic(fmt.Sprintf("Unmatched [ at line %v\n", c._line))

	} //Too many close functions, not enough opens
	if ne._funcLevel == 0 { //Either we are not building a function, or we just finished
		if c.getSource() == "BuildFuncFromStack" { //We move to phase 2, assembling the function from pieces on the data stack
			//fmt.Printf("debug: %v\n", c._source)
			ne._buildingFunc = false //Switch out of phase 1 build mode
			ne._funcLevel += 1       //Start counting function brackets
			ne.dataStack = pushStack(ne.dataStack, NewString("[", e.environment))
			return c._stub(ne, c)
		} else {
			val, ok := nameSpaceLookup(ne, c)
			if ok {
				if val.tiipe == "CODE" {
					ne.codeStack = pushStack(ne.codeStack, val)
					ne.lexStack = pushStack(ne.lexStack, e.environment)
				} else {
					ne.dataStack = pushStack(ne.dataStack, val)
				}
			} else {
				var _, ok = strconv.ParseFloat(c.getSource(), 32) //Numbers don't need to be defined in the namespace
				if ok != nil {
					fmt.Printf("Warning:  %v not defined at %v:%v\n", c.GetString(), c._filename, c._line)

				}
				ne.dataStack = pushStack(ne.dataStack, c)
			}
		}
	} else {
		ne.dataStack = pushStack(ne.dataStack, c)
	}
	//fmt.Printf("TokenStep: %v\n", c._source)
	return ne
}

//Tokens cause a namespace lookup on their string value
//Whatever gets returned is pushed onto the code stack
//Then on the next step, that Thing gets activated, usually moving itself to the data stack, or running some code
//This can cause infinite loops if the token resolves to itself
//So we should probably detect that and move it to the data stack, since that is probably what the user wanted?
func NewToken(aString string, env *Thingy) *Thingy {
	t := newThingy()
	t.tiipe = "TOKEN"
	t.subType = "NATIVE"
	t.setString(aString)
	t.environment = env
	t._stub = tokenStepper
	t.arity = 0
	return t
}

//Raw byte representated.
func NewBytes(bytes []byte, env *Thingy) *Thingy {
	t := newThingy()
	t.tiipe = "BYTES"
	t.subType = "NATIVE"
	t._bytesVal = bytes
	t.environment = nil
	t._stub = func(e *Engine, c *Thingy) *Engine {
		ne := cloneEngine(e, false)
		ne.dataStack = pushStack(ne.dataStack, c)
		//fmt.Printf("StringStep: %v\n", c._source)
		return ne
	}
	t.arity = -1
	return t
}

//Unicode strings.  Length etc might not be the same as its byte representation
func NewString(aString string, env *Thingy) *Thingy {
	t := newThingy()
	t.tiipe = "STRING"
	t.subType = "NATIVE"
	t.setString(aString)
	t.environment = nil
	t._stub = func(e *Engine, c *Thingy) *Engine {
		ne := cloneEngine(e, false)
		ne.dataStack = pushStack(ne.dataStack, c)
		//fmt.Printf("StringStep: %v\n", c._source)
		return ne
	}
	t.arity = -1
	return t
}

//Stores a reference to the engine at the point where it was called.  When activated, execution continues at that point
func NewContinuation(e *Engine) *Thingy {
	t := newThingy()
	t.tiipe = "CONTINUATION"
	t.subType = "NATIVE"
	t.setString("Continuation")
	t._engineVal = e
	return t
}

//Holds any go structure, like a filehandle, network socket or database handle
func NewWrapper(s interface{}) *Thingy {
	t := newThingy()
	t._structVal = s
	t.tiipe = "WRAPPER"
	t.subType = "NATIVE"
	t.setString("Native structure wrapper")
	t.arity = -1
	return t
}

//Wraps a native go function
func NewCode(aName string, arity int, aFunc StepFunc) *Thingy {
	t := newThingy()
	t.tiipe = "CODE"
	t.subType = "NATIVE"
	t.setStub(aFunc)
	t.setString(aName)
	t.arity = arity
	return t
}

func NewArray(a stack) *Thingy {
	t := newThingy()
	t.tiipe = "ARRAY"
	t.subType = "INTERPRETED"
	t.setString("Array - add code to fill this in properly")
	t.arity = -1
	t._arrayVal = a
	return t
}

func NewHash() *Thingy {
	t := newThingy()
	t.tiipe = "HASH"
	t.subType = "NATIVE"
	t.setString("hash - add code to fill this in properly")
	t.arity = -1
	t._hashVal = make(map[string]*Thingy, 1000)
	return t
}

func NewBool(a int) *Thingy {
	t := newThingy()
	t.tiipe = "BOOLEAN"
	t.subType = "NATIVE"
	t.setString("BOOLEAN")
	t._intVal = a
	t.arity = -1
	return t
}
func NewEngine() *Engine {
	e := new(Engine)
	e.environment = NewHash()
	e._functionCache = map[int]*Thingy{}
	e._heatMap = map[string][]int{}
	e._funcLevel = 0
	e._safeMode = false
	return e
}

//Sometimes, we just don't want to do anything
func NullStep(e *Engine) *Engine {
	return e
}

//Note, it works on immutable stacks, giving us the ability to save old engine states
func pushStack(s stack, v *Thingy) stack {
	return append(s, v)
}

//Note, it works on immutable stacks, giving us the ability to save old engine states
func popStack(s stack) (*Thingy, stack) {

	if len(s) > 0 {
		v, sret := s[len(s)-1], s[:len(s)-1]
		return v, sret
	} else {

		panic("Attempted read past end of stack!")
	}
}

//neatly print out all the variables in scope
func dumpEnv(ll *ll_t) {
	if ll != nil {
		emit(fmt.Sprintf("===Env=== %p\n", ll))
		dumpEnvRec(ll.cdr)
		emit(fmt.Sprintf("===End Env=== %p\n", ll))

	}
}

func dumpEnvRec(ll *ll_t) {
	if ll != nil {
		emit(ll.key + ": " + ll.val.getSource() + "\n")
		dumpEnvRec(ll.cdr)
	}
}

//The core of the interpreter.  Each step advances the program by one command
func doStep(e *Engine) (*Engine, bool) {
	if len(e.codeStack) > 0 { //If there are any instructions left
		var v, lex *Thingy
		var dyn stack
		ne := cloneEngine(e, false) //Clone the current engine state.  The false means "do not clone the lexical environment" i.e. it
		//will be common to this step and the previous step.  Otherwise we would be running in fully
		//immutable mode (to come)
		v, ne.codeStack = popStack(ne.codeStack) //Pop an instruction off the instruction stack.  Usually a token, but could be data or native code
		lex, ne.lexStack = popStack(ne.lexStack)
		dyn = lex.errorChain

		//Move to own routine?
		if v._line != 0 && len(v._filename) > 1 {
			m := ne._heatMap[v._filename]
			if m == nil {
				m = make([]int, 1000000, 1000000)
			}
			m[v._line] = m[v._line] + 1
			ne._heatMap[v._filename] = m

		}

		if interpreter_debug && v.environment != nil {
			dumpEnv(v.environment._llVal)
		}

		if v.tiipe == "CODE" && v.no_environment == true { //Macros do not carry their own environment, they use the environment from the previous instruction
			if interpreter_debug {
				emit(fmt.Sprintf("Macro using invoked environment"))
			}
			lex = ne.environment //Set the macro environment to the current environment i.e. of the previous instruction, which will usually be the token's environment
		} else {
			ne._line = v._line
			if lex != nil {
				ne.environment = lex //Set the current environment to that carried by the current instruction
				ne.dyn = dyn
			}
		}

		//ne.environment = lex
		if traceProg {
			emit(fmt.Sprintf("%v:Step: %v(%v) - (%p) \n", v._line, v.GetString(), v.tiipe, lex))
		}
		//fmt.Printf("Calling: '%v'\n", v.getString())
		if interpreter_debug {
			emit(fmt.Sprintf("Choosing environment %p for command %v(%v)\n", ne.environment, v.GetString(), ne.environment))
		}

		oldlen := len(ne.dataStack) //Note the size of the data stack
		if interpreter_debug {
			emit(fmt.Sprintf("Using environment: %p for command : %v\n", ne.environment, v.GetString()))
		}
		ne = v._stub(ne, v)                           //Call the handler function for this instruction
		newlen := len(ne.dataStack)                   //Note the new data stack size
		if (v.arity > 9000) || (v.tiipe == "TOKEN") { //Tokens and some other instructions can have variable arity
			return ne, true
		} else {
			if v.arity == (oldlen - newlen) { //Check the number of arguments to an instruction against the change in the stack size
				return ne, true
			} else {
				if v.tiipe == "CODE" {
					emit(fmt.Sprintln(fmt.Sprintf("Arity mismatch in native function! %v claimed %v, but actually took %v\n", v.getSource(), v.arity, (oldlen - newlen)))) //FIXME name not printing

				}
				return ne, true
			}
		}
	} else {
		//fmt.Printf("No code left to run!\n")
		return e, false
	}
}

func SlurpFile(fname string) string {
	content, _ := ioutil.ReadFile(fname)
	return string(content)
}

func tokenise(s string, filename string) stack {
	var line int = 0
	s = strings.Replace(s, "\n", " LINEBREAKHERE ", -1)
	s = strings.Replace(s, "\r", " ", -1)
	s = strings.Replace(s, "\t", " ", -1)
	stringBits := strings.Split(s, " ")
	var tokens stack
	for _, v := range stringBits {
		seqID = seqID + 1
		if len(v) > 0 {
			if v == "LINEBREAKHERE" {
				line = line + 1
			} else {
				t := NewToken(v, NewHash())
				t._id = seqID
				t._line = line
				t._filename = filename
				//fmt.Printf("Token id: %v\n", i)
				tokens = pushStack(tokens, t)
			}
		}
	}
	return tokens
}

func StringsToTokens(stringBits []string) stack {
	var tokens stack
	for i, v := range stringBits {
		if len(v) > 0 {
			t := NewToken(v, NewHash())
			t._id = i
			tokens = pushStack(tokens, t)
		}
	}
	return tokens
}

func engineDump(e *Engine) {
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
func run(e *Engine) (*Engine, bool) {
	ok := true
	for ok {
		e, ok = doStep(e)
	}
	return e, ok
}

func (e *Engine) Run() *Engine {
	e, _ = run(e)
	return e
}

func (e *Engine) LoadTokens(s stack) {
	for _, elem := range s {
		elem.environment = e.environment
		e.lexStack = pushStack(e.lexStack, e.environment)
	} //All tokens start off sharing the root environment
	e.codeStack = append(e.codeStack, s...)
}

func (e *Engine) DataStackTop() *Thingy {
	return e.dataStack[len(e.dataStack)-1]
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

func Repl(e *Engine) *Engine {
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

func realRepl(e *Engine, rl *readline.Instance) *Engine {
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
	if len(text) > 0 {
		e.LoadTokens(tokenise(text, "repl"))
		//stackDump(e.codeStack)
		e, _ = run(e)
		emit(fmt.Sprintln(e.dataStack[len(e.dataStack)-1].GetString()))
		realRepl(e, rl)
		return e
	} else {
		return e
	}
}

var outputBuff string = ""

//All output to STDOUT should go through this function, so we can resend it to network ports, logfiles etc
func emit(s string) {
	outputBuff = fmt.Sprintf("%s%s", outputBuff, s)
	fmt.Printf("%s", s)
}

func clearOutput() string {
	var temp = outputBuff
	outputBuff = ""
	return temp
}

func (e *Engine) RunString(s string, source string) *Engine {
	e.LoadTokens(tokenise(s, source))
	e, _ = run(e)
	return e
}

func (e *Engine) CallArray(s string, args []string) ([]string, *Engine) {
	e.LoadTokens(tokenise(s, "Call from go"))
	tarray := []*Thingy{}
	for _, v := range args {
		tarray = append(tarray, tokenise(v, "Call args loader")...)
	}
	t := NewArray(tarray)
	e.dataStack = append(e.dataStack, t)
	e, _ = run(e)
	var out []string
	throffArr := e.DataStackTop()
	for _, v := range throffArr._arrayVal {
		out = append(out, v.GetString())
	}
	return out, e
}

func (e *Engine) CallArgs(s string, args ...string) (string, *Engine) {
	e.LoadTokens(tokenise(s, "CallArgs from go"))
	for _, v := range args {
		e.LoadTokens(tokenise(v, "CallArgs args loader"))
	}
	e, _ = run(e)
	return e.DataStackTop().GetString(), e
}

func (e *Engine) CallArgs1(s string, args ...interface{}) (string, *Engine) {
	e.LoadTokens(tokenise(s, "CallArgs from go"))
	for _, v := range args {
		str := fmt.Sprintf("%v", v)
		e.LoadTokens(tokenise(str, "CallArgs args loader"))
	}
	e, _ = run(e)
	return e.DataStackTop().GetString(), e
}

func (e *Engine) RunFile(s string) *Engine {
	codeString := SlurpFile(s)
	//println(codeString)
	return e.RunString(codeString, s)
}

func stackDump(s stack) {
	emit(fmt.Sprintf("\nStack: "))
	for i, _ := range s {
		if i < 20 {
			emit(fmt.Sprintf(":%v(%v):", s[len(s)-1-i].getSource(), s[len(s)-1-i].tiipe))
		}
	}
}

//This is the code that is called when a function is activated
func buildFuncStepper(e *Engine, c *Thingy) *Engine {
	ne := cloneEngine(e, false)
	//fmt.Printf("Loading code\n")
	var lexical_env *Thingy
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
			lexical_env = cloneEnv(c.environment) //Copy the lexical environment to make a new environment for this function (e.g. think of recursion)
			lexical_env.errorChain = e.dyn
		}
		//Take all the words (the functions) from the current function definition and push them onto the code stack
		for _, ee := range c._arrayVal {
			ne.codeStack = pushStack(ne.codeStack, ee)
			ne.lexStack = pushStack(ne.lexStack, lexical_env)
		}
	} else {
		//If it's not code, it's data, so it goes on the data stack
		ne.dataStack = pushStack(ne.dataStack, c)
	}
	return ne
}

//Recurse down the data stack, picking up words and collecting them in the stack f. f will become our function.
func buildFunc(e *Engine, f stack) *Engine {
	var v *Thingy
	ne := cloneEngine(e, false)
	v, ne.dataStack = popStack(ne.dataStack)
	if v.getSource() == "[" {
		ne._funcLevel += 1
	}
	if v.getSource() == "]" {
		ne._funcLevel -= 1
	}

	//fmt.Printf("BUILDFUNC: in function level: %v\n", ne._funcLevel)
	//fmt.Printf("Considering %v\n", v.getSource())
	if v.getSource() == "]" && ne._funcLevel == 0 {
		//fmt.Printf("fINISHING FUNCTION\n")
		//This code is called when the newly-built function is activated
		newFunc := NewCode("InterpretedCode", 0, buildFuncStepper)
		newFunc.tiipe = "LAMBDA"
		newFunc.subType = "INTERPRETED"
		newFunc._arrayVal = f
		newFunc._line = e._line
		ne.dataStack = pushStack(ne.dataStack, newFunc)
		if USE_FUNCTION_CACHE && v._id > 0 {
			newFunc.codeStackConsume = len(f)
			ne._functionCache[v._id] = newFunc

			//fmt.Printf("Caching new function of length %v, id %v\ncache %v", newFunc.codeStackConsume, v._id, ne._functionCache)
		}
		return ne
	} else {
		//fmt.Printf("Building FUNCTION\n")
		v.environment = e.environment
		f = pushStack(f, v)
		ne = buildFunc(ne, f)
	}
	//fmt.Printf("Returning\n")
	return ne
}
