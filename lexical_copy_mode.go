// lexical_copy_mode.go
//
// These functions provide lexical scoping.
//
/* The strategy is: every time we enter a scope (e.g. a function),
 we copy the current scope.  Yes, all of it.  Not the values, which are
 references, but we do copy all the bindings.

This is horribly inefficient, but does give us the lexical behaviour that
I want.  Variables inside this scope are "mutable", variables outside this
scope are "immutable".  You can shadow a variable from an outer scope, but
you can't actually alter it for any other part of the code.  This means that
the only way to send data back to the parent scope is by returning a value,
either by exiting the function, or sending it through a queue.

This minimises the ability of programmers to accidentally cause errors in
other routines, while still allowing reasonably natural code.

*/

package throfflib

import (
	"fmt"
	"strconv"
)

//Search for the value of t, in its assigned scope.
//Throff uses a flat namespace, much like its predecessor forth, except that a new copy is made each time we enter a function.
func nameSpaceLookup(e *Engine, t *Thingy) (*Thingy, bool) {
	key := t.GetString()
	val, ok := e.environment._hashVal[key]
	if interpreter_debug {
		emit(fmt.Sprintf("%p: Looking up: %v -> %v in %v\n", e.environment, key, val, e.environment))
	}
	if !ok {
		var _, ok = strconv.ParseFloat(t.getSource(), 32) //Numbers don't need to be defined in the namespace
		if ok != nil {
			//Maybe enable this once we reduce the amount of bad code a bit.  Or add a "nag" option
			//emit(fmt.Sprintf("Warning: %v not defined at line %v\n", key, t._line))
			if e._safeMode {
				panic(fmt.Sprintf("Error: %v not defined at line %v\n", key, t._line))
			}
		}
	}
	return val, ok
}

func cloneMap(m map[string]*Thingy) map[string]*Thingy {
	//fmt.Printf("Cloning map %v\n\n", m )
	var nm = make(map[string]*Thingy, 1000)
	for k, v := range m {
		nm[k] = v
	}
	return nm
}
