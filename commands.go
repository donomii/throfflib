// commands.go
//All the basic commands supported by throff.  They are all
//cross platform, and usually are in caps
//
//They all need more error checking
package throfflib

//import "unsafe"
//import "github.com/thinxer/go-tcc"
import "github.com/codeskyblue/go-sh"
import "math/big"
import "fmt"
import "time"
import "strings"
import "bufio"
import "os"
import "io/ioutil"
import "strconv"
import "math"
import "runtime"
import "github.com/edsrzf/mmap-go"
import "net/http"
import "net"
import "html"
import "log"
import "io"
import "database/sql"
import "os/exec"
import (
	_ "github.com/mattn/go-sqlite3"
)

func String2Big(in string, precision uint) *big.Float {
	v1, _, _ := big.ParseFloat(in, 10, precision, big.ToZero)
	if v1 == nil {
		v1 = big.NewFloat(0)
	}
	v1 = v1.SetPrec(precision)
	return v1

}

//import "golang.org/x/net/websocket"

//Creates a new engine and populates it with the core functions
func MakeEngine() *Engine {

	e := NewEngine()

	e = add(e, "IDEBUGOFF", NewCode("IDEBUGOFF", 0, func(e *Engine, c *Thingy) *Engine {
		interpreter_debug = false
		return e
	}))

	e = add(e, "CLEAROUTPUT", NewCode("CLEAROUTPUT", 0, func(e *Engine, c *Thingy) *Engine {
		e.dataStack = pushStack(e.dataStack, NewString(clearOutput(), e.environment))
		return e
	}))

	e = add(e, "IDEBUGON", NewCode("IDEBUGON", 0, func(e *Engine, c *Thingy) *Engine {
		interpreter_debug = true
		return e
	}))

	e = add(e, "FORCEGC", NewCode("FORCEGC", 0, func(e *Engine, c *Thingy) *Engine {
		runtime.GC()
		return e
	}))

	e = add(e, "DEBUGOFF", NewCode("DEBUGOFF", 0, func(e *Engine, c *Thingy) *Engine {
		debug = false
		return e
	}))

	e = add(e, "DEBUGON", NewCode("DEBUGOFF", 0, func(e *Engine, c *Thingy) *Engine {
		debug = true
		return e
	}))

	e = add(e, "TROFF", NewCode("TROFF", 0, func(e *Engine, c *Thingy) *Engine {
		traceProg = false
		return e
	}))

	e = add(e, "TRON", NewCode("TRONS", 0, func(e *Engine, c *Thingy) *Engine {
		traceProg = true
		return e
	}))

	e = add(e, "ITROFF", NewCode("ITROFF", 0, func(e *Engine, c *Thingy) *Engine {
		interpreter_trace = false
		return e
	}))

	e = add(e, "ITRON", NewCode("ITROFF", 0, func(e *Engine, c *Thingy) *Engine {
		interpreter_trace = true
		return e
	}))

	e = add(e, "NULLSTEP", NewCode("NullStep", 0, func(e *Engine, c *Thingy) *Engine {
		emit(fmt.Sprintf("NullStep\n"))
		return e
	}))

	e = add(e, "DROP", NewCode("DROP", 1, func(ne *Engine, c *Thingy) *Engine {
		_, ne.dataStack = popStack(ne.dataStack)
		return ne
	}))

	e = add(e, "ZERO", NewCode("ZERO", -1, func(ne *Engine, c *Thingy) *Engine {
		ne.dataStack = pushStack(ne.dataStack, NewString("0", e.environment))
		return ne
	}))

	e = add(e, "ROLL", NewCode("ROLL", 1, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		var n, _ = strconv.ParseInt(el1.getSource(), 10, 32)
		n = int64(len(ne.dataStack)) - n - 1
		v := ne.dataStack[n]
		ne.dataStack = append(ne.dataStack[:n], ne.dataStack[n+1:]...)
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne
	}))

	e = add(e, "PICK", NewCode("PICK", 0, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		var n, _ = strconv.ParseInt(el1.getSource(), 10, 32)
		n = int64(len(ne.dataStack)) - n - 1
		v := ne.dataStack[n]
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne
	}))

	e = add(e, "NUM2CHAR", NewCode("NUM2CHAR", 0, func(ne *Engine, c *Thingy) *Engine {
		var v, el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		var n, _ = strconv.ParseInt(el1.getSource(), 10, 32)
		v = NewString(fmt.Sprintf("%c", n), e.environment)
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne
	}))

	e = add(e, "GETLINE", NewCode("GETLINE", 0, func(ne *Engine, c *Thingy) *Engine {
		var v *Thingy
		bio := bufio.NewReader(os.Stdin)
		line, _, _ := bio.ReadLine()
		v = NewString(string(line), e.environment)
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne
	}))

	e = add(e, "OPENFILE", NewCode("OPENFILE", -1, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy

		el1, ne.dataStack = popStack(ne.dataStack)
		f, err := os.Open(el1.GetString())
		if !(err == nil) {
			return ne.RunString(fmt.Sprintf("THROW [ Could not open file %v: %v ] ", el1.GetString(), err), "Internal Error")
		}

		reader := bufio.NewReaderSize(f, 999999)
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(f))
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(reader))
		return ne
	}))

	e = add(e, "OPENSQLITE", NewCode("OPENSQLITE", 0, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy

		el1, ne.dataStack = popStack(ne.dataStack)
		db, err := sql.Open("sqlite3", el1.GetString())
		if err != nil {
			log.Fatal(err)
		}

		ne.dataStack = pushStack(ne.dataStack, NewWrapper(db))
		return ne
	}))

	e = add(e, "QUERY", NewCode("QUERY", 1, func(ne *Engine, c *Thingy) *Engine {
		var el1, querystring *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		querystring, ne.dataStack = popStack(ne.dataStack)
		db := el1._structVal.(*sql.DB)
		str := querystring.GetString()

		rows, err := db.Query(str)

		if err != nil {
			emit(fmt.Sprintf("Error: Reading from table %v", err))
		}
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(rows))
		return ne
	}))

	e = add(e, "EXEC", NewCode("EXEC", 3, func(ne *Engine, c *Thingy) *Engine {
		var el1, querystring, wrappedArgs *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		querystring, ne.dataStack = popStack(ne.dataStack)
		wrappedArgs, ne.dataStack = popStack(ne.dataStack)
		db := el1._structVal.(*sql.DB)
		stringArgs := []interface{}{}
		for _, v := range wrappedArgs._arrayVal {
			stringArgs = append(stringArgs, v.GetString())
		}

		_, err := db.Exec(querystring.GetString(), stringArgs...)

		if err != nil {
			emit(fmt.Sprintf("Error: exec failed: %v", err))
		}
		return ne
	}))

	e = add(e, "NEXTROW", NewCode("NEXTROW", 0, func(ne *Engine, c *Thingy) *Engine {
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
		h := NewHash()
		for i, v := range result {
			h._hashVal[cols[i]] = NewString(v, e.environment)

		}

		ne.dataStack = pushStack(ne.dataStack, h)
		return ne
	}))

	e = add(e, "CLOSEFILE", NewCode("CLOSEFILE", 1, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy

		el1, ne.dataStack = popStack(ne.dataStack)
		f := el1._structVal.(*os.File)
		f.Close()
		return ne
	}))

	e = add(e, "MMAPFILE", NewCode("MMAPFILE", -1, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy

		el1, ne.dataStack = popStack(ne.dataStack)
		f, _ := os.OpenFile(el1.GetString(), os.O_RDWR, 0644)
		//info, _ :=os.Lstat(el1.getString())
		b, err := mmap.Map(f, mmap.RDWR, 0)
		if err != nil {
			emit(fmt.Sprintf("mmap failed: %v\n", err))

		}
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(f))
		bt := NewBytes(b, el1.environment)
		bt._structVal = f
		ne.dataStack = pushStack(ne.dataStack, bt)
		return ne
	}))

	e = add(e, "RUNSTRING", NewCode("RUNSTRING", 9001, func(ne *Engine, c *Thingy) *Engine {
		var el1, env *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		env, ne.dataStack = popStack(ne.dataStack)
		ne.environment = env
		ne = ne.RunString(el1.GetString(), "runstring")
		return ne
	}))

	e = add(e, "READFILELINE", NewCode("READFILELINE", 0, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		reader := el1._structVal.(*bufio.Reader)
		buff, _, ok := reader.ReadLine()
		var v *Thingy
		if ok == nil {
			v = NewString(string(buff), ne.environment)
		} else {
			v = NewBool(0)
		}
		ne.dataStack = pushStack(ne.dataStack, v)
		return ne
	}))

	e = add(e, "THIN", NewCode("THIN", 0, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		el2 := clone(el1)
		el2.share_parent_environment = true
		ne.dataStack = pushStack(ne.dataStack, el2)
		return ne
	}))

	e = add(e, "MACRO", NewCode("MACRO", 0, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		el2 := clone(el1)
		el2.no_environment = true
		el2.share_parent_environment = true
		el2.environment = nil
		ne.dataStack = pushStack(ne.dataStack, el2)
		return ne
	}))

	e = add(e, "CALL", NewCode("CALL", 1, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		el2 := clone(el1)
		if el2.tiipe == "LAMBDA" {
			el2.tiipe = "CODE"
		}
		ne.codeStack = pushStack(ne.codeStack, el2)
		ne.lexStack = pushStack(ne.lexStack, ne.environment)
		//engineDump(ne)
		return ne
	}))

	e = add(e, "EMIT", NewCode("EMIT", 1, func(ne *Engine, c *Thingy) *Engine {
		var v *Thingy
		v, ne.dataStack = popStack(ne.dataStack)
		emit(fmt.Sprintf("%v", v.GetString()))
		return ne
	}))

	e = add(e, "PRINTLN", NewCode("PRINTLN", 1, func(ne *Engine, c *Thingy) *Engine {
		var v *Thingy
		v, ne.dataStack = popStack(ne.dataStack)
		//fmt.Printf("printing type: %v\n", v.tiipe)
		emit(fmt.Sprintf("%v\n", v.GetString()))
		return ne
	}))

	e = add(e, "]", NewCode("StartFunctionDef", 0, func(ne *Engine, c *Thingy) *Engine {
		ne._buildingFunc = true
		ne.dataStack = pushStack(ne.dataStack, c)
		return ne
	}))

	e = add(e, "[", NewCode("BuildFuncFromStack", 9001, func(ne *Engine, c *Thingy) *Engine {
		ne._funcLevel += 1 //To match the ] we will find on the stack
		var f stack
		ne = buildFunc(ne, f)
		newFunc, _ := popStack(ne.dataStack)
		newFunc.environment = ne.environment
		return ne
	}))

	e = add(e, "DIRECTORY-LIST", NewCode("DIRECTORY-LIST", 0, func(ne *Engine, c *Thingy) *Engine {
		var dir []os.FileInfo
		var aDir *Thingy
		aDir, ne.dataStack = popStack(ne.dataStack)
		dir, _ = ioutil.ReadDir(aDir.GetString())
		var f stack
		for _, el := range dir {
			f = pushStack(f, NewString(el.Name(), e.environment))
		}
		c = NewArray(f)
		ne.dataStack = pushStack(ne.dataStack, c)
		return ne
	}))

	e = add(e, "SPLIT", NewCode("SPLIT", 2, func(ne *Engine, c *Thingy) *Engine {
		var aString, aSeparator, aCount *Thingy
		aString, ne.dataStack = popStack(ne.dataStack)
		aSeparator, ne.dataStack = popStack(ne.dataStack)
		aCount, ne.dataStack = popStack(ne.dataStack)
		n, _ := strconv.ParseInt(aCount.GetString(), 10, 32)
		bits := strings.SplitN(aString.GetString(), aSeparator.GetString(), int(n))
		var f stack
		for _, el := range bits {
			f = pushStack(f, NewString(el, e.environment))
		}
		c = NewArray(f)
		ne.dataStack = pushStack(ne.dataStack, c)
		return ne
	}))

	e = add(e, "SAFETYON", NewCode("SAFETYON", 0, func(ne *Engine, c *Thingy) *Engine {
		ne._safeMode = true
		return ne
	}))

	e = add(e, ":", NewCode(":", 2, func(ne *Engine, c *Thingy) *Engine {
		var aName, aVal *Thingy
		defer func() {
			if r := recover(); r != nil {
				emit(fmt.Sprintln("Unable to set variable ", aName.getSource(), " because ", r))
				engineDump(ne)
				os.Exit(1)
			}
		}()
		aName, ne.dataStack = popStack(ne.dataStack)
		aVal, ne.dataStack = popStack(ne.dataStack)
		env := ne.environment
		if interpreter_debug {
			emit(fmt.Sprintf("Environment: %p - Storing %v in %v\n", env, aVal.GetString(), aName.GetString()))
		}

		prev, ok := env._hashVal[aName.GetString()]
		if ok {
			if e._safeMode {
				emit(fmt.Sprintf("Warning:  mutating binding %v in %v at line %v(previous value %v)\n", aName.GetString(), aName._filename, aName._line, prev.GetString()))
				os.Exit(1)
			}
		}
		env._hashVal[aName.GetString()] = aVal
		checkVal, checkOk := env._hashVal[aName.GetString()]
		if interpreter_debug {
			emit(fmt.Sprintf("Checked var %v, value is %v, in environment %p - %v\n", aName.GetString(), checkVal, env, env))
		}
		if !checkOk {
			panic("bind name failed!")
		}
		if !(checkVal == aVal) {
			panic("bind name failed!")
		}
		if checkVal == nil {
			panic("bind name failed!")
		}
		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}

		return ne
	}))

	e = add(e, "REBIND", NewCode(":", 2, func(ne *Engine, c *Thingy) *Engine {
		var aName, aVal *Thingy
		defer func() {
			if r := recover(); r != nil {
				emit(fmt.Sprintln("Unable to set variable ", aName.getSource(), " because ", r))
				engineDump(ne)
				os.Exit(1)
			}
		}()

		aName, ne.dataStack = popStack(ne.dataStack)
		aVal, ne.dataStack = popStack(ne.dataStack)
		env := aName.environment
		if interpreter_debug {
			emit(fmt.Sprintf("Environment: %p - Storing %v in %v\n", env, aVal.GetString(), aName.GetString()))
		}

		_, ok := env._hashVal[aName.GetString()]
		if !ok {
			if e._safeMode {
				emit(fmt.Sprintf("Warning:  Could not mutate: binding %v not found at line %v\n", aName.GetString(), aName._line))
				os.Exit(1)
			}
		}
		env._hashVal[aName.GetString()] = aVal
		_, ok = env._hashVal[aName.GetString()]
		if !ok {
			panic("key not found")
		}
		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne
	}))

	e = add(e, "ENVIRONMENTOF", NewCode("ENVIRONMENTOF", 0, func(ne *Engine, c *Thingy) *Engine {
		var aName, aVal *Thingy
		aName, ne.dataStack = popStack(ne.dataStack)
		if interpreter_debug {
			emit(fmt.Sprintf("Environment: %p - Storing %v in %v\n", aName.environment, aVal.GetString(), aName.GetString()))
		}
		ne.dataStack = pushStack(ne.dataStack, ne.environment)

		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne
	}))

	e = add(e, "SETENV", NewCode("SETENV", 1, func(ne *Engine, c *Thingy) *Engine {
		var aFunc, anEnv, newFunc *Thingy
		aFunc, ne.dataStack = popStack(ne.dataStack)
		anEnv, ne.dataStack = popStack(ne.dataStack)
		newFunc = clone(aFunc)
		newFunc.environment = anEnv
		ne.dataStack = pushStack(ne.dataStack, newFunc)

		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne
	}))

	e = add(e, "LOCATIONOF", NewCode("LOCATIONOF", 0, func(ne *Engine, c *Thingy) *Engine {
		var aVal *Thingy
		aVal, ne.dataStack = popStack(ne.dataStack)
		if interpreter_debug {
			emit(fmt.Sprintf("Location: %v\n", aVal._line))
		}
		H := NewString(fmt.Sprintf("%v", aVal._line), c.environment)
		ne.dataStack = pushStack(ne.dataStack, H)

		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne
	}))

	e = add(e, "FILEOF", NewCode("FILEOF", 0, func(ne *Engine, c *Thingy) *Engine {
		var aVal *Thingy
		aVal, ne.dataStack = popStack(ne.dataStack)
		if interpreter_debug {
			emit(fmt.Sprintf("File: %v\n", aVal._filename))
		}
		H := NewString(fmt.Sprintf("%v", aVal._filename), c.environment)
		ne.dataStack = pushStack(ne.dataStack, H)

		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne
	}))

	e = add(e, "SETLEX", NewCode("SETLEX", 2, func(ne *Engine, c *Thingy) *Engine {
		var aName, aVal *Thingy
		aName, ne.dataStack = popStack(ne.dataStack)
		aVal, ne.dataStack = popStack(ne.dataStack)
		env := ne.environment
		//fmt.Printf("Storing %v in %v\n", aVal._source, aName._source)
		env._hashVal[aName.GetString()] = aVal
		_, ok := env._hashVal[aName.GetString()]
		if !ok {
			panic("key not found in environment after set")
		}
		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne
	}))

	e = add(e, "SCRUBLEX", NewCode("SCRUBLEX", 1, func(ne *Engine, c *Thingy) *Engine {
		var aName *Thingy
		aName, ne.dataStack = popStack(ne.dataStack)
		env := ne.environment
		if interpreter_debug {
			emit(fmt.Sprintf("Scrubbing %v from %v\n", aName.GetString(), env))
		}
		delete(env._hashVal, aName.GetString())
		_, ok := env._hashVal[aName.GetString()]
		if ok {
			panic("key found in environment after set")
		}
		//for k,v := range ne.environment {fmt.Printf("%v: %v\n", k,v)}
		return ne
	}))

	e = add(e, "GETLEX", NewCode("GETLEX", 0, func(ne *Engine, c *Thingy) *Engine {
		var aName *Thingy
		aName, ne.dataStack = popStack(ne.dataStack)
		//fmt.Printf("Fetching %v\n", aName.getString())

		aVal, ok := ne.environment._hashVal[aName.GetString()]
		if !ok {
			for k, v := range ne.environment._hashVal {
				emit(fmt.Sprintf("%v: %v\n", k, v))
			}
			emit(fmt.Sprintln("key not found ", aName.GetString()))
			panic("Key not found error")
		}

		ne.dataStack = pushStack(ne.dataStack, aVal)
		return ne
	}))

	e = add(e, "EQUAL", NewCode("EQUAL", 1, func(ne *Engine, c *Thingy) *Engine {
		var aVal, bVal *Thingy
		aVal, ne.dataStack = popStack(ne.dataStack)
		bVal, ne.dataStack = popStack(ne.dataStack)

		if aVal.GetString() == bVal.GetString() {
			ne.dataStack = pushStack(ne.dataStack, NewBool(1))
		} else {
			ne.dataStack = pushStack(ne.dataStack, NewBool(0))
		}
		return ne
	}))

	e = add(e, "IF", NewCode("IF", 3, func(ne *Engine, c *Thingy) *Engine {
		var testVal, trueBranch, falseBranch *Thingy
		testVal, ne.dataStack = popStack(ne.dataStack)
		trueBranch, ne.dataStack = popStack(ne.dataStack)
		falseBranch, ne.dataStack = popStack(ne.dataStack)

		ne.codeStack = pushStack(ne.codeStack, NewToken("CALL", nil))
		ne.lexStack = pushStack(ne.lexStack, ne.environment)

		if testVal._intVal == 1 {
			ne.codeStack = pushStack(ne.codeStack, trueBranch)
			ne.lexStack = pushStack(ne.lexStack, ne.environment)
		} else {
			ne.codeStack = pushStack(ne.codeStack, falseBranch)
			ne.lexStack = pushStack(ne.lexStack, ne.environment)
		}
		//engineDump(ne)
		return ne
	}))

	e = add(e, "NOT", NewCode("NOT", 0, func(ne *Engine, c *Thingy) *Engine {
		var aVal *Thingy
		aVal, ne.dataStack = popStack(ne.dataStack)
		aVal = clone(aVal)

		if aVal._intVal == 0 {
			aVal._intVal = 1
		} else {
			aVal._intVal = 0
		}
		ne.dataStack = pushStack(ne.dataStack, aVal)
		return ne
	}))

	e = add(e, "LESSTHAN", NewCode("LESSTHAN", 1, func(ne *Engine, c *Thingy) *Engine {
		var aVal, bVal *Thingy
		aVal, ne.dataStack = popStack(ne.dataStack)
		bVal, ne.dataStack = popStack(ne.dataStack)

		var a = String2Big(aVal.getSource(), precision)
		var b = String2Big(bVal.getSource(), precision)
		cmp := a.Cmp(b)
		if cmp == -1 { // -1 means a < b
			ne.dataStack = pushStack(ne.dataStack, NewBool(1))
		} else {
			ne.dataStack = pushStack(ne.dataStack, NewBool(0))
		}
		return ne
	}))

	e = add(e, "THREAD", NewCode("THREAD", 1, func(ne *Engine, c *Thingy) *Engine {

		var threadBranch *Thingy
		threadBranch, ne.dataStack = popStack(ne.dataStack)

		ne2 := cloneEngine(ne, true)
		ne2.codeStack = stack{}
		ne2.lexStack = stack{}
		ne2.dataStack = stack{}

		ne2.codeStack = pushStack(ne2.codeStack, NewToken("CALL", ne.environment))
		ne2.lexStack = pushStack(ne2.lexStack, ne.environment)

		ne2.codeStack = pushStack(ne2.codeStack, threadBranch)
		ne2.lexStack = pushStack(ne2.lexStack, ne.environment)
		go func() { run(ne2) }()

		return ne
	}))

	e = add(e, "SLEEP", NewCode("SLEEP", 1, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		n, _ := strconv.ParseInt(el1.getSource(), 10, 64)
		time.Sleep(time.Duration(n) * time.Millisecond)
		return ne
	}))

	e = add(e, "GETTYPE", NewCode("GETTYPE", 0, func(ne *Engine, c *Thingy) *Engine {
		var v *Thingy
		v, ne.dataStack = popStack(ne.dataStack)
		ne.dataStack = pushStack(ne.dataStack, NewString(v.tiipe, e.environment))
		return ne
	}))

	e = add(e, "SETTYPE", NewCode("SETTYPE", 1, func(ne *Engine, c *Thingy) *Engine {
		var t, el *Thingy
		t, ne.dataStack = popStack(ne.dataStack)
		el, ne.dataStack = popStack(ne.dataStack)

		targetType := t.GetString()
		el = clone(el)
		if targetType == "STRING" && (el.tiipe == "CODE" || el.tiipe == "LAMBDA") {
			el._stringVal = el.GetString() //Calculate the string representation of the array before we change the type
			el._source = el.getSource()    //Calculate the string representation of the array before we change the type
		}
		if targetType == "STRING" && (el.tiipe == "BOOLEAN") {
			el._stringVal = el.GetString() //Calculate the string representation of the array before we change the type
			el._source = el.getSource()    //Calculate the string representation of the array before we change the type
		}
		if targetType == "STRING" && (el.tiipe == "ARRAY") {
			el._stringVal = el.GetString() //Calculate the string representation of the array before we change the type
			el._source = el.getSource()    //Calculate the string representation of the array before we change the type
		}
		if targetType == "CODE" && (el.tiipe == "CODE" || el.tiipe == "LAMBDA") {
			el.arity = 0
		}
		el.tiipe = targetType
		ne.dataStack = pushStack(ne.dataStack, el)
		return ne
	}))

	e = add(e, "->BYTES", NewCode("->BTES", 0, func(ne *Engine, c *Thingy) *Engine {
		var t, el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)

		t = NewBytes([]byte(el.GetString()), el.environment)

		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "SPACE", NewCode("SPACE", -1, func(ne *Engine, c *Thingy) *Engine {
		ne.dataStack = pushStack(ne.dataStack, NewString(" ", e.environment))
		return ne
	}))

	e = add(e, ".S", NewCode(".S", 0, func(e *Engine, c *Thingy) *Engine {
		stackDump(e.dataStack)
		return e
	}))

	e = add(e, ".C", NewCode(".C", 0, func(e *Engine, c *Thingy) *Engine {
		stackDump(e.codeStack)
		return e
	}))

	e = add(e, ".L", NewCode(".S", 0, func(e *Engine, c *Thingy) *Engine {
		emit(fmt.Sprintln())
		//stackDump(e.codeStack)
		emit(fmt.Sprintln("lexstack"))
		stackDump(e.lexStack)
		emit(fmt.Sprintln())
		return e
	}))

	e = add(e, ".E", NewCode(".E", 0, func(e *Engine, c *Thingy) *Engine {
		fmt.Println("Engine environment")
		dumpEnv(e.environment)
		fmt.Println("Thingy environment")
		dumpEnv(c.environment)
		return e
	}))

	e = add(e, "ARRAYPUSH", NewCode("ARRAYPUSH", 1, func(ne *Engine, c *Thingy) *Engine {
		var arr, el *Thingy
		arr, ne.dataStack = popStack(ne.dataStack)
		el, ne.dataStack = popStack(ne.dataStack)
		newarr := clone(arr)
		newarr._arrayVal = append(arr._arrayVal, el)

		ne.dataStack = pushStack(ne.dataStack, newarr)
		return ne
	}))

	e = add(e, "NEWARRAY", NewCode("NEWARRAY", -1, func(ne *Engine, c *Thingy) *Engine {
		var arr *Thingy
		arr = NewArray(stack{})
		ne.dataStack = pushStack(ne.dataStack, arr)
		return ne
	}))

	e = add(e, "POPARRAY", NewCode("POPARRAY", -1, func(ne *Engine, c *Thingy) *Engine {
		var arr, el *Thingy
		arr, ne.dataStack = popStack(ne.dataStack)
		var newarr *Thingy = clone(arr)
		newarr._arrayVal = nil
		el, newarr._arrayVal = popStack(arr._arrayVal)
		ne.dataStack = pushStack(ne.dataStack, newarr)
		ne.dataStack = pushStack(ne.dataStack, el)
		return ne
	}))

	e = add(e, "SHIFTARRAY", NewCode("SHIFTARRAY", -1, func(ne *Engine, c *Thingy) *Engine {
		var arr *Thingy
		arr, ne.dataStack = popStack(ne.dataStack)
		el := arr._arrayVal[0]
		newarr := clone(arr)
		newarr._arrayVal = nil
		newarr._arrayVal = append(stack{}, arr._arrayVal[1:]...)
		ne.dataStack = pushStack(ne.dataStack, newarr)
		ne.dataStack = pushStack(ne.dataStack, el)
		return ne
	}))

	e = add(e, "UNSHIFTARRAY", NewCode("UNSHIFTARRAY", 1, func(ne *Engine, c *Thingy) *Engine {
		var arr, el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		arr._arrayVal = append(stack{el}, arr._arrayVal...)

		ne.dataStack = pushStack(ne.dataStack, arr)
		return ne
	}))

	e = add(e, "GETARRAY", NewCode("GETARRAY", 1, func(ne *Engine, c *Thingy) *Engine {
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
		var n, _ = strconv.ParseInt(el.getSource(), 10, 64)
		ret := arr._arrayVal[n]

		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))

	e = add(e, "GETBYTE", NewCode("GETBYTE", 1, func(ne *Engine, c *Thingy) *Engine {
		var arr, el *Thingy

		el, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		var n, _ = strconv.ParseInt(el.getSource(), 10, 32)
		ret := arr._bytesVal[n]

		ne.dataStack = pushStack(ne.dataStack, NewString(fmt.Sprintf("%c", ret), el.environment))
		return ne
	}))

	e = add(e, "GETSTRING", NewCode("GETSTRING", 1, func(ne *Engine, c *Thingy) *Engine {
		var arr, el *Thingy

		el, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		var n, _ = strconv.ParseInt(el.getSource(), 10, 32)
		s := arr.GetString()
		var s1 string
		for index, r := range s {
			if index == int(n) {
				s1 = fmt.Sprintf("%c", r)
			}
		}
		ret := NewString(s1, arr.environment)
		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))
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
	e = add(e, "SETARRAY", NewCode("SETARRAY", 2, func(ne *Engine, c *Thingy) *Engine {
		var arr, index, value *Thingy
		index, ne.dataStack = popStack(ne.dataStack)
		value, ne.dataStack = popStack(ne.dataStack)
		arr, ne.dataStack = popStack(ne.dataStack)
		var n, _ = strconv.ParseInt(index.getSource(), 10, 32)

		newarr := clone(arr)
		newarr._arrayVal = make(stack, len(arr._arrayVal), len(arr._arrayVal))
		copy(newarr._arrayVal, arr._arrayVal)

		newarr._arrayVal[n] = value

		ne.dataStack = pushStack(ne.dataStack, newarr)
		return ne
	}))

	e = add(e, "KEYVALS", NewCode("KEYVALS", 0, func(ne *Engine, c *Thingy) *Engine {
		var arr, hash *Thingy
		hash, ne.dataStack = popStack(ne.dataStack)
		arr = NewArray(stack{})
		for k, v := range hash._hashVal {
			arr._arrayVal = append(arr._arrayVal, NewString(k, ne.environment), v)
		}

		ne.dataStack = pushStack(ne.dataStack, arr)
		return ne
	}))

	e = add(e, "STRING-CONCATENATE", NewCode("STRING-CONCATENATE", 1, func(ne *Engine, c *Thingy) *Engine {
		var s1, s2 *Thingy
		s1, ne.dataStack = popStack(ne.dataStack)
		s2, ne.dataStack = popStack(ne.dataStack)
		s3 := NewString(fmt.Sprintf("%s%s", s1.GetString(), s2.GetString()), ne.environment)

		ne.dataStack = pushStack(ne.dataStack, s3)
		return ne
	}))

	/*e=add(e, "SWAP",  NewCode("SWAP", 1, func (ne *Engine,c *Thingy) *Engine {
	var el, el1 *Thingy
	el, ne.dataStack = popStack(ne.dataStack)
	el1, ne.dataStack = popStack(ne.dataStack)
	ne.dataStack = pushStack(ne.dataStack, el  )
	ne.dataStack = pushStack(ne.dataStack, el1  )
	return ne}))
	*/

	e = add(e, "ADD", NewCode("ADD", 1, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		var v1, v2 *big.Float
		var v3 big.Float
		//var err error
		v1 = String2Big(el.getSource(), precision)
		v2 = String2Big(el1.getSource(), precision)

		v3.SetPrec(0)
		v3 = *v3.Add(v1, v2)

		var t *Thingy = NewString(fmt.Sprintf("%v", v3.Text('g', int(precision))), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "FLOOR", NewCode("FLOOR", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)

		var v1 = String2Big(el.getSource(), precision)

		v2, _ := v1.Int(nil)
		var t *Thingy = NewString(fmt.Sprintf("%v", v2), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "SETHASH", NewCode("SETHASH", 2, func(ne *Engine, c *Thingy) *Engine {
		var key, val, hash, newhash *Thingy
		key, ne.dataStack = popStack(ne.dataStack)
		val, ne.dataStack = popStack(ne.dataStack)
		hash, ne.dataStack = popStack(ne.dataStack)
		newhash = clone(hash)
		newhash._hashVal = cloneMap(hash._hashVal)
		newhash._hashVal[key.GetString()] = val
		ne.dataStack = pushStack(ne.dataStack, newhash)
		return ne
	}))

	e = add(e, "GETHASH", NewCode("GETHASH", 1, func(ne *Engine, c *Thingy) *Engine {
		var key, val, hash *Thingy
		key, ne.dataStack = popStack(ne.dataStack)
		hash, ne.dataStack = popStack(ne.dataStack)
		val = hash._hashVal[key.GetString()]
		if val == nil {
			emit(fmt.Sprintf("Warning: %v not found in hash%v\n\nCreating empty value\n", key.GetString(), hash._hashVal))
			val = NewString(fmt.Sprintf("UNDEFINED:%v", key.GetString()), ne.environment)
		}
		ne.dataStack = pushStack(ne.dataStack, val)
		return ne
	}))

	e = add(e, "NEWHASH", NewCode("NEWHASH", -1, func(ne *Engine, c *Thingy) *Engine {
		var hash *Thingy = NewHash()
		ne.dataStack = pushStack(ne.dataStack, hash)
		return ne
	}))

	e = add(e, "SUB", NewCode("SUB", 1, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		var v1 = String2Big(el.getSource(), precision)
		var v2 = String2Big(el1.getSource(), precision)
		var t *Thingy = NewString(fmt.Sprintf("%v", v1.Sub(v1, v2)), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "MULT", NewCode("MULT", 1, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		v1 := String2Big(el.getSource(), precision)
		v2 := String2Big(el1.getSource(), precision)
		var t *Thingy = NewString(fmt.Sprintf("%v", v1.Mul(v1, v2)), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "MODULUS", NewCode("MODULUS", 1, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		v1B := String2Big(el.getSource(), precision)
		v2B := String2Big(el1.getSource(), precision)

		v1, _ := v1B.Float64()
		v2, _ := v2B.Float64()
		var t *Thingy = NewString(fmt.Sprintf("%v", math.Mod(v1, v2)), el.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "MODULUSI", NewCode("MODULUS", 1, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy
		var ShutTheFuckUpAndDoTheCalculation *big.Int
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		v1 := String2Big(el.getSource(), precision)
		v2 := String2Big(el1.getSource(), precision)
		v1I, _ := v1.Int(ShutTheFuckUpAndDoTheCalculation)
		v2I, _ := v2.Int(ShutTheFuckUpAndDoTheCalculation)
		m := v1I.Mod(v1I, v2I)
		var t *Thingy = NewString(fmt.Sprintf("%v", m), el.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "LN", NewCode("LN", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		v1B := String2Big(el.getSource(), 32)
		v1, _ := v1B.Float64()
		var t *Thingy = NewString(fmt.Sprintf("%v", math.Log2(v1)), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "DIVIDE", NewCode("DIVIDE", 1, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		var v1, v2 *big.Float
		var v3 big.Float
		v1 = String2Big(el.getSource(), precision)
		v2 = String2Big(el1.getSource(), precision)
		v3.SetPrec(0)
		v3 = *v3.Quo(v1, v2)
		var t *Thingy = NewString(fmt.Sprintf("%v", v3.Text('g', int(precision))), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "TIMESEC", NewCode("TIMESEC", -1, func(ne *Engine, c *Thingy) *Engine {
		var t *Thingy = NewString(fmt.Sprintf("%v", int32(time.Now().Unix())), e.environment)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "TOK", NewCode("TOK", -1, func(ne *Engine, c *Thingy) *Engine {
		var el, lex *Thingy
		el, ne.codeStack = popStack(ne.codeStack)
		lex, ne.lexStack = popStack(ne.lexStack)
		el1 := clone(el)
		el1.environment = lex
		ne.dataStack = pushStack(ne.dataStack, el1)
		return ne
	}))

	e = add(e, "GETFUNCTION", NewCode("GETFUNCTION", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		val, ok := nameSpaceLookup(ne, el)
		if ok {
			ne.dataStack = pushStack(ne.dataStack, val)
		} else {
			ne.dataStack = pushStack(ne.dataStack, NewToken("FALSE", ne.environment))
		}
		//stackDump(ne.dataStack)
		return ne
	}))

	e = add(e, "RPCSERVER", NewCode("RPCSERVER", 0, func(ne *Engine, c *Thingy) *Engine {
		rpc_server("127.0.0.1:80")
		return ne
	}))

	e = add(e, "TCPSERVER", NewCode("TCPSERVER", 1, func(ne *Engine, c *Thingy) *Engine {
		var server, port *Thingy
		server, ne.dataStack = popStack(ne.dataStack)
		port, ne.dataStack = popStack(ne.dataStack)
		// Listen on TCP port 2000 on all interfaces.
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.GetString(), port.GetString()))
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
		return e
	}))
	e = add(e, "OPENSOCKET", NewCode("OPENSOCKET", 1, func(ne *Engine, c *Thingy) *Engine {
		var server, port *Thingy
		server, ne.dataStack = popStack(ne.dataStack)
		port, ne.dataStack = popStack(ne.dataStack)
		conn, err := net.Dial("tcp", fmt.Sprintf("%v:%v", server.GetString(), port.GetString()))
		emit(fmt.Sprintf("%v", err))
		t := NewWrapper(conn)
		ne.dataStack = pushStack(ne.dataStack, t)
		return ne
	}))

	e = add(e, "PRINTSOCKET", NewCode("PRINTSOCKET", 2, func(ne *Engine, c *Thingy) *Engine {
		var message, conn *Thingy
		message, ne.dataStack = popStack(ne.dataStack)
		conn, ne.dataStack = popStack(ne.dataStack)
		fmt.Fprintf(conn._structVal.(io.Writer), message.GetString())
		return ne
	}))

	e = add(e, "READSOCKETLINE", NewCode("READSOCKETLINE", 0, func(ne *Engine, c *Thingy) *Engine {
		var server *Thingy
		server, ne.dataStack = popStack(ne.dataStack)
		var conn net.Conn
		conn = server._structVal.(net.Conn)
		message, _ := bufio.NewReader(conn).ReadString('\n')
		ret := NewString(message, ne.environment)
		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))

	e = add(e, "HTTPSERVER", NewCode("HTTPSERVER", 0, func(ne *Engine, c *Thingy) *Engine {
		var path, callback *Thingy
		path, ne.dataStack = popStack(ne.dataStack)
		callback, ne.dataStack = popStack(ne.dataStack)

		http.HandleFunc(path.GetString(), func(w http.ResponseWriter, r *http.Request) {
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
			fmt.Fprintf(w, "Hello, %q, %q, %v", callback.GetString(), html.EscapeString(r.URL.Path), ret.GetString())
		})
		cwd, _ := os.Getwd()
		emit(fmt.Sprintf("Serving /resources/ from:%s\n", cwd))
		http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir(cwd))))
		http.ListenAndServe(":80", nil)
		return ne
	}))
	/*
		e = add(e, "WEBSOCKETCLIENT", NewCode("WEBSOCKETCLIENT", 0, func(ne *Engine, c *Thingy) *Engine {
			var url, protocol, origin *Thingy
			var q1, q2 *Thingy
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
			go func() {
				for {
					var msg = make([]byte, 512)
					var n int
					if n, err = ws.Read(msg); err != nil {
						log.Fatal(err)
					}
					//fmt.Printf("Received: %s.\n", msg[:n])
					rqueue <- NewBytes(msg[:n], ne.environment)
				}
			}()

			go func() {
				for {
					msg := <-wqueue
					emit(fmt.Sprintf("Sending %v\n", msg.getString()))
					if _, err := ws.Write([]byte(msg.getString())); err != nil {
						log.Fatal(err)
					}
				}
			}()

			ne.dataStack = pushStack(ne.dataStack, NewString("", e.environment))
			return ne
		}))
	*/
	e = add(e, "RPCSERVER", NewCode("RPCSERVER", 0, func(ne *Engine, c *Thingy) *Engine {
		rpc_server("127.0.0.1:80")
		return ne
	}))

	e = add(e, "GETWWW", NewCode("GETWWW", 0, func(ne *Engine, c *Thingy) (re *Engine) {
		var path *Thingy
		path, ne.dataStack = popStack(ne.dataStack)
		defer func() {
			if r := recover(); r != nil {
				emit(fmt.Sprintln("Failed to retrieve ", path.getSource(), " because ", r))
				ne.dataStack = pushStack(ne.dataStack, NewString("", e.environment))
				re = ne
			}
		}()
		res, err := http.Get(path.GetString())
		if err != nil {
			log.Println(err)
		}
		robots, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Println(err)
		}

		ne.dataStack = pushStack(ne.dataStack, NewString(string(robots), ne.environment))

		return ne
	}))

	e = add(e, "EXIT", NewCode("EXIT", 0, func(ne *Engine, c *Thingy) *Engine {
		/*for f, m := range ne._heatMap {
			emit(fmt.Sprintln("Hotspots in file ", f))
			for i, v := range m {
				emit(fmt.Sprintf("%d: %d\n", i,v))
			}
		}*/
		emit(fmt.Sprintln("Goodbye"))
		os.Exit(0)
		return e
	}))

	e = add(e, "EXITw/CODE", NewCode("EXITw/CODE", 0, func(ne *Engine, c *Thingy) *Engine {
		var el1 *Thingy
		el1, ne.dataStack = popStack(ne.dataStack)
		var n, _ = strconv.ParseInt(el1.getSource(), 10, 32)
		os.Exit(int(n))
		return e
	}))

	e = add(e, "LENGTH", NewCode("LENGTH", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		var val int
		if el.tiipe == "ARRAY" || el.tiipe == "LAMBDA" || el.tiipe == "CODE" {
			val = len(el._arrayVal)
		}
		if el.tiipe == "STRING" {
			val = len(el.GetString())
		}
		if el.tiipe == "BYTES" {
			val = len(el._bytesVal)
		}
		ne.dataStack = pushStack(ne.dataStack, NewString(fmt.Sprintf("%v", val), ne.environment))
		//stackDump(ne.dataStack)
		return ne
	}))

	e = add(e, "SHELL", NewCode("SHELL", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		var args []string
		for _, s := range el._arrayVal {
			args = append(args, s.GetString())
		}
		//This is lunacy
		command := args[0]
		var arglist []interface{}
		for _, i := range args[1:] {
			arglist = append(arglist, i)
		}
		//sh.Command(command, arglist...).Run()
		out, err := sh.Command(command, arglist...).Output()
		if err != nil {
			log.Fatal(err)
		}
		ne.dataStack = pushStack(ne.dataStack, NewString(string(out), ne.environment))
		return ne
	}))

	e = add(e, "NEWQUEUE", NewCode("NEWQUEUE", -1, func(ne *Engine, c *Thingy) *Engine {
		q := make(chan *Thingy, 1000)
		ne.dataStack = pushStack(ne.dataStack, NewWrapper(q))
		//stackDump(ne.dataStack)
		return ne
	}))

	e = add(e, "WRITEQ", NewCode("WRITEQ", 2, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)

		q := el._structVal.(chan *Thingy)

		q <- el1
		//ne.dataStack = pushStack(ne.dataStack, NewWrapper(q))
		//stackDump(ne.dataStack)
		return ne
	}))

	e = add(e, "READQ", NewCode("READQ", 0, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy
		el, ne.dataStack = popStack(ne.dataStack)

		q := el._structVal.(chan *Thingy)
		el1 = <-q

		ne.dataStack = pushStack(ne.dataStack, el1)
		//stackDump(ne.dataStack)
		return ne
	}))

	e = add(e, "DNS.CNAME", NewCode("DNS.CNAME", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		r, _ := net.LookupCNAME(el.GetString())
		ne.dataStack = pushStack(ne.dataStack, NewString(string(r), nil))
		return ne
	}))

	e = add(e, "DNS.HOST", NewCode("DNS.HOST", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		r, _ := net.LookupHost(el.GetString())
		a := fmt.Sprintf("->ARRAY [ %v  ]", strings.Join(r, " "))
		ne = ne.RunString(a, "DNS.HOST")
		//ne.dataStack = pushStack(ne.dataStack, NewString(string(a), nil))
		return ne
	}))

	e = add(e, "DNS.TXT", NewCode("DNS.TXT", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		r, _ := net.LookupTXT(el.GetString())
		a := fmt.Sprintf("->ARRAY [ %v  ]", strings.Join(r, " "))
		ne = ne.RunString(a, "DNS.TXT")
		//ne.dataStack = pushStack(ne.dataStack, NewString(string(a), nil))
		return ne
	}))

	e = add(e, "DNS.REVERSE", NewCode("DNS.REVERSE", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		r, _ := net.LookupAddr(el.GetString())
		a := fmt.Sprintf("->ARRAY [ %v  ]", strings.Join(r, " "))
		ne = ne.RunString(a, "DNS.REVERSE")
		//ne.dataStack = pushStack(ne.dataStack, NewString(string(a), nil))
		return ne
	}))

	e = add(e, "CALL/CC", NewCode("CALL/CC", -1, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		cc := NewWrapper(ne)
		cc._engineVal = ne

		ne = cloneEngine(ne, true)
		ne.codeStack = pushStack(ne.codeStack, NewToken("CALL", nil))
		ne.lexStack = pushStack(ne.lexStack, ne.environment)

		ne.dataStack = pushStack(ne.dataStack, cc)
		ne.dataStack = pushStack(ne.dataStack, el)

		return ne
	}))

	e = add(e, "ACTIVATE/CC", NewCode("ACTIVATE/CC", 9999, func(ne *Engine, c *Thingy) *Engine {
		var el, el1 *Thingy

		el, ne.dataStack = popStack(ne.dataStack)
		el1, ne.dataStack = popStack(ne.dataStack)
		ne = el._structVal.(*Engine)

		ne.dataStack = pushStack(ne.dataStack, el1)
		return ne
	}))

	e = add(e, "INSTALLDYNA", NewCode("INSTALLDYNA", 2, func(ne *Engine, c *Thingy) *Engine {
		var el, err *Thingy
		err, ne.dataStack = popStack(ne.dataStack)
		el, ne.dataStack = popStack(ne.dataStack)

		var new_env = ne.environment
		var errStack = append(ne.dyn, err)
		new_env.errorChain = errStack
		ne.lexStack = pushStack(ne.lexStack, new_env)
		ne.codeStack = pushStack(ne.codeStack, el)
		return ne
	}))
	e = add(e, "ERRORHANDLER", NewCode("ERRORHANDLER", -1, func(ne *Engine, c *Thingy) *Engine {
		var errHandler *Thingy
		var new_env = ne.environment
		errHandler, _ = popStack(new_env.errorChain)
		ne.dataStack = pushStack(ne.dataStack, errHandler)
		return ne
	}))
	e = add(e, "DUMP", NewCode("DUMP", 1, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)

		ret := NewString(fmt.Sprintf("%+v", el), nil)
		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))

	e = add(e, "OS", NewCode("OS", -1, func(ne *Engine, c *Thingy) *Engine {
		ret := NewString(runtime.GOOS, nil)

		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))

	e = add(e, "CMDSTDOUTSTDERR", NewCode("CMDSTDOUTSTDERR", 0, func(ne *Engine, c *Thingy) *Engine {
		var el_arr *Thingy
		el_arr, ne.dataStack = popStack(ne.dataStack)

		var argv = []string{}
		for _, v := range el_arr._arrayVal {
			argv = append(argv, v.GetString())
		}

		cmd := exec.Command(argv[0], argv[1:]...)
		stdoutStderr, _ := cmd.CombinedOutput()

		ret := NewString(string(stdoutStderr), nil)

		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))

	e = add(e, "STARTPROCESS", NewCode("STARTPROCESS", 1, func(ne *Engine, c *Thingy) *Engine {
		var el, el_arr *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		el_arr, ne.dataStack = popStack(ne.dataStack)

		var argv = []string{el.GetString()}
		//fmt.Printf("$V", el_arr._arrayVal)
		for _, v := range el_arr._arrayVal {
			argv = append(argv, v.GetString())
		}
		//fmt.Printf("$V", argv)
		attr := os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		}
		proc, err := os.StartProcess(el.GetString(), argv, &attr)
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
	e = add(e, "KILLPROC", NewCode("KILLPROC", 1, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		proc := el._structVal.(*os.Process)
		err := proc.Kill()
		var ret *Thingy

		if err == nil {
			ret = NewBool(0)
		} else {
			ret = NewString(fmt.Sprintf("%v", err), nil)
		}

		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))
	e = add(e, "RELEASEPROC", NewCode("RELEASEPROC", 1, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		proc := el._structVal.(*os.Process)
		err := proc.Release()
		var ret *Thingy

		if err == nil {
			ret = NewBool(0)
		} else {
			ret = NewString(fmt.Sprintf("%v", err), nil)
		}

		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))

	e = add(e, "WAITPROC", NewCode("WAITPROC", 1, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		proc := el._structVal.(*os.Process)
		procState, err := proc.Wait()
		var ret *Thingy

		if err == nil {
			ret = NewString(fmt.Sprintf("%v\nPid: %v\nSystemTime: %v\nUserTime: %v\nSuccess: %v", procState.String(), procState.Pid(), procState.SystemTime(), procState.UserTime(), procState.Success()), nil)
		} else {
			ret = NewString(fmt.Sprintf("%v\nPid: %v\nSystemTime: %v\nUserTime: %v\nSuccess: %vError: %v", procState.String(), procState.Pid(), procState.SystemTime(), procState.UserTime(), procState.Success(), err), nil)
		}

		ne.dataStack = pushStack(ne.dataStack, ret)
		return ne
	}))

	e = add(e, "BYTE2STR", NewCode("BYTE2STR", 0, func(ne *Engine, c *Thingy) *Engine {
		var el *Thingy
		el, ne.dataStack = popStack(ne.dataStack)
		var b []byte = el._bytesVal
		var str string = string(b[:len(b)])
		ne.dataStack = pushStack(ne.dataStack, NewString(str, ne.environment))
		return ne
	}))

	e = add(e, "CLEARSTACK", NewCode("CLEARSTACK", 9999, func(ne *Engine, c *Thingy) *Engine {
		ne.dataStack = stack{} //The argument stack
		ne.dyn = stack{}       //The current dynamic environment
		ne.codeStack = stack{} //The future of the program
		ne.lexStack = stack{}

		return ne
	}))
	e = add(e, "SIN", NewCode("SIN", 0, func(ne *Engine, c *Thingy) *Engine {
		var arg *Thingy
		arg, ne.dataStack = popStack(ne.dataStack)
		flin := String2Big(arg.getSource(), precision)
		in, _ := flin.Float64()
		var ret = math.Sin(in)
		ne.dataStack = pushStack(ne.dataStack, NewString(fmt.Sprintf("%v", ret), ne.environment))
		return ne
	}))

	/*
			e = add(e, "MAKEJIT", NewCode("MAKEJIT", -1, func(ne *Engine, c *Thingy) *Engine {
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

			e = add(e, "RUNJIT", NewCode("RUNJIT", 0, func(ne *Engine, c *Thingy) *Engine {
				var jitwrap *Thingy

				jitwrap, ne.dataStack = popStack(ne.dataStack)
				jit := jitwrap._structVal.(*tcc.State)

				a := C.float(2.0)
				b := C.float(3.0)

				p, err := jit.Symbol("jit_func")
				if err != nil {
					log.Fatal(err)
				}

				n := float64(C.call(C.jitfunc(unsafe.Pointer(p)), a, b))

				ne.dataStack = pushStack(ne.dataStack, NewString(fmt.Sprintf("%v", n), e.environment))
				return ne
			}))
	*/

	//fmt.Println("Done")
	return e
}
