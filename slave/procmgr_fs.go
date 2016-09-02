package slave

import (
	"bufio"
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const skeletonDirPermissions = 0700
const KeyfilePermissions = 0600

// Create directories managed by ProcessManager
// returns nil if already exists and permissions are suitable
func (p *ProcessManager) CreateManagedDirs() (err error) {

	if err = unix.Access(p.dataDir, unix.W_OK); err != nil {
		return fmt.Errorf("`%s` does not exist or is not writable", p.dataDir)
	}

	if err = unix.Access(p.dataDir, unix.R_OK|unix.W_OK|unix.X_OK); err != nil {
		if err := os.Mkdir(p.managedDirMongods(), os.ModeDir|0700); err != nil {
			return err
		}
	}

	return nil

}

// directory for mongod process root dirs
func (p *ProcessManager) managedDirMongods() string {
	return filepath.Join(p.dataDir, "mongods")
}

// Create root directory for a process in a subdirectory of
// a directory created by CreateManagedDirs()
// returns nil if already exists and permissions are suitable
func (p *ProcessManager) createDirSkeleton(m msp.Mongod) (err error) {

	if err = os.MkdirAll(p.processRootDir(m), os.ModeDir|skeletonDirPermissions); err != nil {
		goto wrapError
	}

	if err = os.MkdirAll(p.processDBPathDir(m), os.ModeDir|skeletonDirPermissions); err != nil {
		goto wrapError
	}

	if err = os.MkdirAll(p.processConfDir(m), os.ModeDir|skeletonDirPermissions); err != nil {
		goto wrapError
	}

	return nil

wrapError:
	return fmt.Errorf("could not create directory skeleton for Mongod: %s", err)

}

func (p *ProcessManager) destroyDataDirectory(m msp.Mongod) error {
	return os.RemoveAll(p.processRootDir(m))
}

// Generate list of process root directories in process root directory tree
func (p *ProcessManager) parseProcessDirTree() (replSetNameByPortNumber map[msp.PortNumber]string, err error) {

	entries, err := ioutil.ReadDir(p.managedDirMongods())
	if err != nil {
		return
	}

	replSetNameByPortNumber = make(map[msp.PortNumber]string)
	for _, entry := range entries {
		if entry.IsDir() {
			port, replSet, err := p.parseProcessRootDirentry(entry)
			if err != nil {
				log.Errorf("error parsing process root directory entry: %s", err)
			} else {
				replSetNameByPortNumber[port] = replSet
			}
		} else {
			log.Errorf("unexpected directory entry in `%s` (not a directory)", entry.Name())
			continue
		}
	}

	return

}

// Root directory of a process
// process data should not be directly stored there
func (p *ProcessManager) processRootDir(m msp.Mongod) string {
	return filepath.Join(p.managedDirMongods(), fmt.Sprintf("%d:%s", m.Port, m.ReplicaSetConfig.ReplicaSetName))
}

// Given a direntry (no prefix!), extract the data encoded in the direntry
func (p *ProcessManager) parseProcessRootDirentry(entry os.FileInfo) (port msp.PortNumber, replSet string, err error) {
	dirname := entry.Name()
	colonLoc := strings.Index(entry.Name(), ":")
	if colonLoc < 0 {
		err = fmt.Errorf("directory with unparsable name `%s` in db dir", entry.Name())
		return
	}
	portStr := dirname[0:colonLoc]
	port64, parseErr := strconv.ParseUint(portStr, 10, 16)
	if parseErr != nil {
		err = fmt.Errorf("could not parse port number `%s`", portStr)
		return
	}
	port = msp.PortNumber(port64)
	replSet = dirname[colonLoc+1:]
	return
}

func (p *ProcessManager) processDBPathDir(m msp.Mongod) string {
	return filepath.Join(p.processRootDir(m), "dbpath")
}

func (p *ProcessManager) processConfDir(m msp.Mongod) string {
	return filepath.Join(p.processRootDir(m), "conf")
}

func (p *ProcessManager) processKeyfilePath(m msp.Mongod) string {
	return filepath.Join(p.processConfDir(m), "keyfile")
}

// Update or create the keyfile of a Mongod
func (p *ProcessManager) UpdateKeyfile(m msp.Mongod) (err error) {

	var keyFilePath = p.processKeyfilePath(m)

	equal, err := fileContentEqualToBytes(keyFilePath, m.KeyfileContent)
	switch {
	case equal:
		return nil
	case err != nil:
		return err
	}

	keyFile, err := os.OpenFile(keyFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, KeyfilePermissions)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	w := bufio.NewWriter(keyFile)
	_, err = w.WriteString(m.KeyfileContent)
	if err != nil {
		return err
	}
	if err = w.Flush(); err != nil {
		return err
	}

	return nil

}

func fileContentEqualToBytes(path string, content string) (equal bool, err error) {

	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) { // nonexistent equivalent to unequal
			return false, nil
		} else {
			return false, err
		}
	}

	fileContentStr := string(fileContent)

	return content == fileContentStr, nil

}
