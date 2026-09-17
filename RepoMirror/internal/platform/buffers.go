package platform

import (
	"sync"
	"unsafe"
)

const fileCompareBufferSize = 32 * 1024
const maxRetainedPathCap = 4 * 1024

var fileBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, fileCompareBufferSize)
	},
}

var pathBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 256)
	},
}

func borrowFileBuffer() []byte {
	return fileBufferPool.Get().([]byte)
}

func releaseFileBuffer(buffer []byte) {
	fileBufferPool.Put(buffer[:fileCompareBufferSize])
}

func borrowPathBuffer(targetCap int) []byte {
	buffer := pathBufferPool.Get().([]byte)
	if cap(buffer) < targetCap {
		return make([]byte, 0, targetCap)
	}
	return buffer[:0]
}

func releasePathBuffer(buffer []byte) {
	if cap(buffer) > maxRetainedPathCap {
		return
	}
	pathBufferPool.Put(buffer[:0])
}

func bytesToStringView(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(raw), len(raw))
}
