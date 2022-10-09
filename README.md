# Instructions for installation

Tested environment requirement
- Raspberry PI 4 B Rev 1.4
- Debian GNU/Linux 11
- go version go1.18.4 linux/arm64
- $Home/Desktop directory should exist (Mount Point)

Navigate to the filesystem application for the infinite directory and start the application.

```
$ cd cmd/infiniteDirectory
$ go run main.go
```

## Author of code in files

- cmd/infiniteDirectory/main.go (author)
- fs/dirstream_linux.go (modifying each method, the original structure was maintained)
