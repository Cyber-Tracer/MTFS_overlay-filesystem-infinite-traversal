// Copyright 2019 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//TODO 1, attack at the beginning a fake directory through
//TODO 1.1, how to calculate reclen
// d_reclen is the size (in bytes) of the returned record)
// What if we created the variable and then transformed it into bytes, would this work?
// Lets try

// Another solution is to add a variable in loopbackDirStream, then check the variable it it's set
// This seems the easiest way!.

//TODO maybe we want to create new folders in the filesystem ... I dont like this but it might be best solution.

package fs

import (
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"github.com/hanwen/go-fuse/v2/fuse"
)

type loopbackDirStream struct {
	//array of uint8
	buf []byte
	//array of uint8
	todo []byte

	// Protects fd so we can guard against double close
	mu sync.Mutex
	//fd = file descriptor
	fd int
}

// NewLoopbackDirStream open a directory for reading as a DirStream
func NewLoopbackDirStream(name string) (DirStream, syscall.Errno) {
	fd, err := syscall.Open(name, syscall.O_DIRECTORY, 0755)
	if err != nil {
		return nil, ToErrno(err)
	}

	ds := &loopbackDirStream{
		// make creates a slice of size 4096, with value unit8, slice == array!
		// this is used to call getDetns, but how many entries can the buffer hold, 4096?? TODO
		// we can just assume MAX ---. ? What do you think?
		buf: make([]byte, 4096),
		fd:  fd,
	}

	//create a directory on each directory
	p := filepath.Join(name, "fake")
	os.Mkdir(p, 0755)

	if err := ds.load(); err != 0 {
		ds.Close()
		return nil, err
	}
	return ds, OK
}

func (ds *loopbackDirStream) Close() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.fd != -1 {
		syscall.Close(ds.fd)
		ds.fd = -1
	}
}

func (ds *loopbackDirStream) HasNext() bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return len(ds.todo) > 0
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

// it is trying to return a dirEntry, it kinda uses some unsafe thigns race detector
func (ds *loopbackDirStream) Next() (fuse.DirEntry, syscall.Errno) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// We can't use syscall.Dirent here, because it declares a
	// [256]byte name, which may run beyond the end of ds.todo.
	// when that happens in the race detector, it causes a panic
	// "converted pointer straddles multiple allocations"

	//ds.todo[0] returns the first entry
	//it does smth with safety wich I dont understand
	// It says that the thing I am returning is a dirent type ?
	// & is the address
	// * retrives the variable
	// Getting the first byte section.
	de := (*dirent)(unsafe.Pointer(&ds.todo[0]))

	// getting the byte where the name should be stored .
	nameBytes := ds.todo[unsafe.Offsetof(dirent{}.Name):de.Reclen]

	//sets where the new directory should occur, removes the directory, as it has already processed so the start now starts from de.Reclen till end!
	ds.todo = ds.todo[de.Reclen:]

	// After the loop, l contains the index of the first '\0'.
	// I think it loops until it gets to 0/ meaning at 0/ is the name.
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
	return result, ds.load()
}

// if it has more than one element it returns ok, else is gets the new elements
func (ds *loopbackDirStream) load() syscall.Errno {
	// if it has elements dont touch it, return OK
	// it checks how many elements does todo has, if it has more than 0 returns OK,
	if len(ds.todo) > 0 {
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

	// So here because we have a buffer of 4096, but not everytime all will be filled by getDents
	// so buf says the read bytes are from X to Y, so from 0 to n, where N is how many bites the getDent returned
	ds.todo = ds.buf[:n]
	return OK
}
