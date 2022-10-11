package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

func main() {

	home := os.Getenv("HOME")
	mntDir, _ := ioutil.TempDir("", "")

	root, err := fs.NewLoopbackRoot(mntDir)
	if err != nil {
		log.Fatal(err)
	}

	mountOpts := &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug:      true,
			AllowOther: true,
		},
	}

	// Mount the file system
	server, err := fs.Mount(home+"/Desktop", root, mountOpts)
	if err != nil {
		log.Fatal(err)
	}

	// Serve the file system, until unmounted by calling fusermount -u
	server.Wait()
}
