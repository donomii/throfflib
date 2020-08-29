
// These functions provide lexical scoping, using a linked list.
//
/* The strategy is: We use a linked list to hold all the variables, so we don't have to copy anything.  New variables
	are added to the front of the list, and are automatically freed when the scope disappears
	and releases the pointer to the front of the list.
 */

package throfflib

import (
	"fmt"
	"strconv"
)

type ll_t struct {
	key string
	val *Thingy
	cdr *ll_t
}

func ll_add(ll *ll_t, key string, val *Thingy) *ll_t {
	return &ll_t{key: key, val: val, cdr: ll}
}

func ll_find(ll *ll_t, search string) *Thingy {
	if ll == nil {
		return nil
	}
	if ll.key == search {
		return ll.val
	}
	return ll_find(ll.cdr, search)
}

//Search for the value of t, in its assigned scope.
//Throff uses relatively normal lexical scoping, except that the outer scopes are immutable.
func nameSpaceLookup(e *Engine, t *Thingy) (*Thingy, bool) {
	key := t.GetString()
	val := ll_find(e.environment._llVal, key)
	//emit(fmt.Sprintf("%p: Looking up: %v -> %v in %v\n", e.environment, key, val, e.environment))
	if interpreter_debug {
		emit(fmt.Sprintf("%p: Looking up: %v -> %v in %v\n", e.environment, key, val, e.environment._llVal))
	}
	if val == nil {
		var _, ok = strconv.ParseFloat(t.getSource(), 32) //Numbers don't need to be defined in the namespace
		if ok != nil {
			if e._safeMode {
				panic(fmt.Sprintf("Error: %v not defined at line %v\n", key, t._line))
			}

		}
		return nil, false
	}
	return val, true
}
