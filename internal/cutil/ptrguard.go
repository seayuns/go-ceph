package cutil

import (
	"sync"
	"unsafe"
)

// PtrGuard respresents a pinned Go pointer (pointing to memory allocated by Go
// runtime) that might get stored in C memory (allocated by C)
type PtrGuard struct {
	// These mutexes will be used as binary semaphores for signalling events
	// from one thread to another, which - in contrast to other languages like
	// C++ - is possible in Go, that is a Mutex can be locked in one thread and
	// unlocked in another.
	pinned, release sync.Mutex
	ptr             uintptr
	store           *uintptr
}

// WARNING: using binary semaphores (mutexes) for signalling like this is quite
// a delicate task in order to avoid deadlocks or panics. Whenever changing the
// code logic, please review at least three times that there is no unexpected
// state possible. Usually the natural choice would be to use channels instead,
// but these can not easily passed to C code because of the pointer-to-pointer
// cgo rule, and would require the use of a Go object registry.

// NewPtrGuard pins the goPtr (pointing to Go memory) and returns a PtrGuard
// object
func NewPtrGuard(goPtr unsafe.Pointer) *PtrGuard {
	var v PtrGuard
	v.ptr = uintptr(goPtr)
	// Since the mutexes are used for signalling, they have to be initialized to
	// locked state, so that following lock attempts will block.
	v.release.Lock()
	v.pinned.Lock()
	// Start a background go routine that lives until Release is called. This
	// calls a special function that makes sure the garbage collector doesn't
	// touch goPtr, and then waits until it reveices the "release" signal, after
	// which it exits.
	go func() {
		pinUntilRelease(&v, uintptr(goPtr))
		v.pinned.Unlock() // send "released" signal to main thread -->(3)
	}()
	// Wait for the "pinned" signal from the go routine. <--(1)
	v.pinned.Lock()
	return &v
}

// Store the pinned Go pointer in C memory at cPtr
func (v *PtrGuard) Store(cPtr CPtr) *PtrGuard {
	if v.ptr == 0 {
		return v
	}
	if v.store != nil {
		panic("double call of Poke()")
	}
	v.store = uintptrPtr(cPtr)
	*v.store = v.ptr // store Go pointer in C memory at cPtr
	return v
}

// Release the pinned Go pointer and set C memory to NULL if it has been stored
func (v *PtrGuard) Release() {
	if v.ptr == 0 {
		return
	}
	v.ptr = 0
	if v.store != nil {
		*v.store = 0
		v.store = nil
	}
	v.release.Unlock() // Send the "release" signal to the go routine. -->(2)
	v.pinned.Lock()    // Wait for the "released" signal <--(3)
}

// The uintptrPtr() helper function below assumes that uintptr has the same size
// as a pointer, although in theory it could be larger.  Therefore we use this
// constant expression to assert size equality as a safeguard at compile time.
// How it works: if sizes are different, either the inner or outer expression is
// negative, which always fails with "constant ... overflows uintptr", because
// unsafe.Sizeof() is a uintptr typed constant.
const _ = -(unsafe.Sizeof(uintptr(0)) - PtrSize) // size assert
func uintptrPtr(p CPtr) *uintptr {
	return (*uintptr)(unsafe.Pointer(p))
}

//go:uintptrescapes

// From https://golang.org/src/cmd/compile/internal/gc/lex.go:
// For the next function declared in the file any uintptr arguments may be
// pointer values converted to uintptr. This directive ensures that the
// referenced allocated object, if any, is retained and not moved until the call
// completes, even though from the types alone it would appear that the object
// is no longer needed during the call. The conversion to uintptr must appear in
// the argument list.
// Also see https://golang.org/cmd/compile/#hdr-Compiler_Directives

func pinUntilRelease(v *PtrGuard, _ uintptr) {
	v.pinned.Unlock() // send "pinned" signal to main thread -->(1)
	v.release.Lock()  // wait for "release" signal from main thread when
	//                   Release() has been called. <--(2)
}
