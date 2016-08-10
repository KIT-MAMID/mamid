package main

import (
	"github.com/KIT-MAMID/mamid/msp"
	. "github.com/KIT-MAMID/mamid/slave"
	"flag"
	"golang.org/x/sys/unix"
	"fmt"
)

func main() {
	var dataDir string
	flag.StringVar(&dataDir, "data", "", "Persistent data and slave configuration directory")

	flag.Parse()

	if dataDir == "" {
		println("No root data directory passed; specify with -data=/path/to/root/dir")
		return
	}

	if err := unix.Access(dataDir, unix.W_OK); err != nil {
		println(fmt.Sprintf("Root data directory %s does not exist or is not writable", dataDir));
		return
	}
	dbDir := fmt.Sprintf("%s/%s", dataDir, DataDBDir)
	if err := unix.Access(dbDir, unix.R_OK | unix.W_OK | unix.X_OK); err != nil {
		if err := unix.Mkdir(dbDir, 0700); err != nil {
			println("Could not create a readable and writable directory at %s")
			return
		}
	}

	controller := NewController(dataDir)
	server := msp.NewServer(controller)
	server.Run()
}
