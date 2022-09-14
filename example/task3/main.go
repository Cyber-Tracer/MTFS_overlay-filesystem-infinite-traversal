package main

import (
	"log"
	"os"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

func main() {
	home := os.Getenv("HOME")

	// Make $HOME available on a mount dir under /tmp/ . Caution:
	// write operations are also mirrored.
	root, err := fs.NewLoopbackRoot(home)
	if err != nil {
		log.Fatal(err)
	}

	millSec := time.Millisecond
	mountOpts := &fs.Options{
		EntryTimeout: &millSec,
		MountOptions: fuse.MountOptions{
			Debug: true,
		},
	}

	// Mount the file system
	server, err := fs.Mount(".", root, mountOpts)
	if err != nil {
		log.Fatal(err)
	}

	// Serve the file system, until unmounted by calling fusermount -u
	server.Wait()
}
