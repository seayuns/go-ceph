package rados

// #include <stdint.h>
import "C"

import (
	"unsafe"

	"github.com/ceph/go-ceph/internal/cutil"
)

type writeStep struct {
	withoutUpdate

	// inputs:
	b  []byte
	pg *cutil.PtrGuard

	// arguments:
	cBuffer   *C.char
	cDataLen  C.size_t
	cWriteLen C.size_t
	cOffset   C.uint64_t
}

func newWriteStep(b []byte, writeLen, offset uint64) *writeStep {
	bufPtr := unsafe.Pointer(&b[0])
	return &writeStep{
		b:         b,
		pg:        cutil.NewPtrGuard(bufPtr),
		cBuffer:   (*C.char)(bufPtr),
		cDataLen:  C.size_t(len(b)),
		cWriteLen: C.size_t(writeLen),
		cOffset:   C.uint64_t(offset),
	}
}

func (v *writeStep) free() {
	v.pg.Release()
}
