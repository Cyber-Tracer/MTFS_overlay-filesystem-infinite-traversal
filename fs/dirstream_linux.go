// Copyright 2019 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"unsafe"

	"github.com/hanwen/go-fuse/v2/fuse"
)

// Modified by @RinorSefa
type loopbackDirStream struct {
	buf  []byte
	todo []byte

	dirList []fuse.DirEntry

	// Protects fd so we can guard against double close
	mu sync.Mutex
	fd int
}

//// Modified by @RinorSefa
// NewLoopbackDirStream open a directory for reading as a DirStream
func NewLoopbackDirStream(name string) (DirStream, syscall.Errno) {
	fd, err := syscall.Open(name, syscall.O_DIRECTORY, 0755)
	if err != nil {
		return nil, ToErrno(err)
	}

	ds := &loopbackDirStream{
		buf:     make([]byte, 4096),
		fd:      fd,
		dirList: make([]fuse.DirEntry, 0),
	}

	//create a directory on each directory
	//" " is first char sorted ASCII Sort Order
	// "!" is the second ch sorted ASCII Sort Order
	p := filepath.Join(name, "!")
	os.Mkdir(p, 0755)

	if err := ds.load(); err != 0 {
		ds.Close()
		return nil, err
	}
	return ds, OK
}

//// Modified by @RinorSefa
func (ds *loopbackDirStream) Close() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.fd != -1 {
		syscall.Close(ds.fd)
		ds.fd = -1
	}
}

//// Modified by @RinorSefa
func (ds *loopbackDirStream) HasNext() bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return len(ds.dirList) > 0
}

// Like syscall.Dirent, but without the [256]byte name.
// check https://man7.org/linux/man-pages/man2/getdents.2.html
type dirent struct {
	Ino    uint64   // Inode number
	Off    int64    // Offset to next dirent ( distance from the start of the directory to the start of the next)
	Reclen uint16   // length of this dirent
	Type   uint8    // file type (offset is (d_reclen - 1)
	Name   [1]uint8 // filename, (reclen - 2 - offsetof(direct,name)
}

// Modified by @RinorSefa
func (ds *loopbackDirStream) Next() (fuse.DirEntry, syscall.Errno) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	result := ds.dirList[0]
	ds.dirList = ds.dirList[1:]
	return result, ds.load()
}

// Author @RinorSefa
func (ds *loopbackDirStream) createDirList() {
	for len(ds.todo) > 0 {
		de := (*dirent)(unsafe.Pointer(&ds.todo[0]))
		nameBytes := ds.todo[unsafe.Offsetof(dirent{}.Name):de.Reclen]
		ds.todo = ds.todo[de.Reclen:]
		l := 0
		for l = range nameBytes {
			if nameBytes[l] == 0 {
				break
			}
		}
		nameBytes = nameBytes[:l]
		result := fuse.DirEntry{
			Ino:  de.Ino,
			Mode: (uint32(de.Type) << 12),
			Name: string(nameBytes),
		}
		ds.dirList = append(ds.dirList, result)
	}
	sort.Slice(ds.dirList, func(i, j int) bool {
		return ds.dirList[i].Name < ds.dirList[j].Name
	})
}

// Modified by @RinorSefa
// if it has more than one element it returns ok, else is gets the new elements
func (ds *loopbackDirStream) load() syscall.Errno {
	if len(ds.dirList) > 0 {
		return OK
	}

	//getdents reads several dirent structures from the directory into
	// the buffer pointed, buffer contains the structure of dirent
	// how many bytes it read is returned
	//getdents is passed the file descriptor and a buffer
	// the bytes read is returned!
	// on error -1 is returned and then the Errno is set to indicate the error
	//The data in the buffer is a series	of dirent structures each containing
	//     the following entries:
	// https://www.freebsd.org/cgi/man.cgi?query=getdents&sektion=2
	// get dents is like a univerzal, it can work for any filesystem.
	// How do we specify how many dents to read. You do they by specifying the buffer bytesize!
	// I think it gets in One go these things. If it returns again, again it will call the same thing?
	// ---------------
	// Here if we add a fake directory, then we need to make sure, then whenever that fakedirectory
	// is going to be opened, we dont need to call the underlying file system but
	// handle it inside the code. Can we do that is the question?

	//Option 1. Create a new Directory each time? by calling mkdir and insert in the begging or end..
	//Option 2. Create a temporary Directory? But how to do this. Maybe create a set of Ino, and whenever
	// something is called with those return fake directories etc etc.
	// amybe we can use fs/inode.go
	// create a new set of fake functions smth similar to LoopbackNode struct, and implement the interfaces
	// yes I like this idea very very much.
	// once in this directories to have a fake filesytem. I dont like this idea..
	// Another idea is with softLinks, to create a softLink to an already infinity directory?
	n, err := syscall.Getdents(ds.fd, ds.buf)
	if err != nil {
		return ToErrno(err)
	}

	ds.todo = ds.buf[:n]
	ds.createDirList()

	return OK
}
