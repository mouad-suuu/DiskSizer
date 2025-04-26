package main

import (
	"DiskSizer/app"
	"os"
)

func main() {
    var startPath string
    if len(os.Args) > 1 {
        startPath = os.Args[1]
    }
    app.StartApp(startPath)
}
