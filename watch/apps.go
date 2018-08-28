package template

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/kelseyhightower/confd/backends"
	"github.com/kelseyhightower/confd/log"
	util "github.com/CBike1103/protoagent/util"
)

type Config struct {
	ReloadCmd   string
	Prefix      string `toml:"prefix"`
	StoreClient backends.StoreClient
}

// Apps is apps to watch
type Apps struct {
	Prefix      string
	ReloadCmd   string
	StageFile   *os.File
	UID         int
	GID         int
	lastIndex   uint64
	storeClient backends.StoreClient
}

// NewApps creates a Apps.
func NewApps(path string,config Config) (*Apps, error) {
	if config.StoreClient == nil {
		return nil, errors.New("A valid StoreClient is required")
	}

	log.Debug("Creating Apps from " + path)

	app := &Apps{}
	app.storeClient = config.StoreClient

	if config.Prefix != "" {
		app.Prefix = config.Prefix
	}

	if !strings.HasPrefix(app.Prefix, "/") {
		app.Prefix = "/" + app.Prefix
	}

	app.UID = os.Geteuid()

	app.GID = os.Getegid()

	return app, nil
}

// sync compares the staged and dest config files
// and attempts to delete them if they differ.
func (a *Apps) sync() error {
	staged := t.StageFile.Name()
	if t.keepStageFile {
		log.Info("Keeping staged file: " + staged)
	} else {
		defer os.Remove(staged)
	}

	log.Debug("Comparing candidate config to " + a.Dest)
	ok, err := util.IsConfigChanged(staged, a.Dest)
	if err != nil {
		log.Error(err.Error())
	}
	if ok {
		log.Info("Target config " + t.Dest + " out of sync")
		if !t.syncOnly && t.CheckCmd != "" {
			if err := t.check(); err != nil {
				return errors.New("Config check failed: " + err.Error())
			}
		}
		log.Debug("Overwriting target config " + t.Dest)
		err := os.Rename(staged, t.Dest)
		if err != nil {
			if strings.Contains(err.Error(), "device or resource busy") {
				log.Debug("Rename failed - target is likely a mount. Trying to write instead")
				// try to open the file and write to it
				var contents []byte
				var rerr error
				contents, rerr = ioutil.ReadFile(staged)
				if rerr != nil {
					return rerr
				}
				err := ioutil.WriteFile(t.Dest, contents, t.FileMode)
				// make sure owner and group match the temp file, in case the file was created with WriteFile
				os.Chown(t.Dest, t.Uid, t.Gid)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
		if !t.syncOnly && t.ReloadCmd != "" {
			if err := t.reload(); err != nil {
				return err
			}
		}
		log.Info("Target config " + t.Dest + " has been updated")
	} else {
		log.Debug("Target config " + t.Dest + " in sync")
	}
	return nil
}

func isConfigChanged(dest string, version int64) (bool, error) {
	// do nothing if file not exist
	if !util.IsFileExist(dest) {
		return true, nil
	}

	// TODO

	return true, nil
}

// check executes the check command to validate the staged config file. The
// command is modified so that any references to src template are substituted
// with a string representing the full path of the staged file. This allows the
// check to be run on the staged file before overwriting the destination config
// file.
// It returns nil if the check command returns 0 and there are no other errors.
func (t *Apps) check() error {
	var cmdBuffer bytes.Buffer
	data := make(map[string]string)
	data["src"] = t.StageFile.Name()
	tmpl, err := template.New("checkcmd").Parse(t.CheckCmd)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(&cmdBuffer, data); err != nil {
		return err
	}
	return runCommand(cmdBuffer.String())
}

// reload executes the reload command.
// It returns nil if the reload command returns 0.
func (t *Apps) reload() error {
	return runCommand(t.ReloadCmd)
}

// runCommand is a shared function used by check and reload
// to run the given command and log its output.
// It returns nil if the given cmd returns 0.
// The command can be run on unix and windows.
func runCommand(cmd string) error {
	log.Debug("Running " + cmd)
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/C", cmd)
	} else {
		c = exec.Command("/bin/sh", "-c", cmd)
	}

	output, err := c.CombinedOutput()
	if err != nil {
		log.Error(fmt.Sprintf("%q", string(output)))
		return err
	}
	log.Debug(fmt.Sprintf("%q", string(output)))
	return nil
}

// process is a convenience function that wraps calls to the three main tasks
// required to keep local configuration files in sync. First we gather vars
// from the store, then we stage a candidate configuration file, and finally sync
// things up.
// It returns an error if any.
func (t *Apps) process() error {
	if err := t.setFileMode(); err != nil {
		return err
	}
	if err := t.setVars(); err != nil {
		return err
	}
	if err := t.createStageFile(); err != nil {
		return err
	}
	if err := t.sync(); err != nil {
		return err
	}
	return nil
}

// setFileMode sets the FileMode.
func (t *Apps) setFileMode() error {
	if t.Mode == "" {
		if !util.IsFileExist(t.Dest) {
			t.FileMode = 0644
		} else {
			fi, err := os.Stat(t.Dest)
			if err != nil {
				return err
			}
			t.FileMode = fi.Mode()
		}
	} else {
		mode, err := strconv.ParseUint(t.Mode, 0, 32)
		if err != nil {
			return err
		}
		t.FileMode = os.FileMode(mode)
	}
	return nil
}
