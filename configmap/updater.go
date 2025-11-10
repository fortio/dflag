// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

// Package kubernetes provides an a K8S ConfigMap watcher for the jobs systems.

package configmap

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"sync/atomic"

	"fortio.org/dflag"
	"fortio.org/dflag/dynloglevel"
	"fortio.org/log"
	"github.com/fsnotify/fsnotify"
)

const (
	k8sInternalsPrefix = ".."
	k8sDataSymlink     = "..data"
)

var (
	errFlagNotDynamic = errors.New("flag is not dynamic")
	errFlagNotFound   = errors.New("flag not found")
)

// Updater is the encapsulation of the directory watcher.
// TODO: hide details, just return opaque interface.
type Updater struct {
	started    bool
	dirPath    string
	parentPath string
	watcher    *fsnotify.Watcher
	flagSet    *flag.FlagSet
	done       chan bool
	warnings   atomic.Int32 // Count of unknown flags that have been logged (increases at each iteration).
	errors     atomic.Int32 // Count of validation errors that have been logged (increases at each iteration).
}

// Setup is a combination/shortcut for New+Initialize+Start.
// It also sets up the `loglevel` flag.
func Setup(flagSet *flag.FlagSet, dirPath string) (*Updater, error) {
	dynloglevel.LoggerFlagSetup()
	log.Infof("Configmap flag value watching on %v", dirPath)
	u, err := New(flagSet, dirPath)
	if err != nil {
		return nil, err
	}
	err = u.Initialize()
	if err != nil {
		return nil, err
	}
	if err := u.Start(); err != nil {
		return nil, err
	}
	return u, nil
}

// New creates an Updater for the directory.
func New(flagSet *flag.FlagSet, dirPath string) (*Updater, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.New("dflag: error initializing fsnotify watcher")
	}
	return &Updater{
		flagSet:    flagSet,
		dirPath:    path.Clean(dirPath),
		parentPath: path.Clean(path.Join(dirPath, "..")), // add parent in case the dirPath is a symlink itself
		watcher:    watcher,
		started:    false,
		done:       nil,
	}, nil
}

// Initialize reads the values from the directory for the first time.
func (u *Updater) Initialize() error {
	if u.started {
		return errors.New("dflag: already initialized updater")
	}
	return u.readAll( /* allowNonDynamic */ false)
}

// Start kicks off the go routine that watches the directory for updates of values.
func (u *Updater) Start() error {
	if u.started {
		return errors.New("dflag: updater already started")
	}
	if err := u.watcher.Add(u.parentPath); err != nil {
		return fmt.Errorf("unable to add parent dir %v to watch: %w", u.parentPath, err)
	}
	if err := u.watcher.Add(u.dirPath); err != nil { // add the dir itself.
		return fmt.Errorf("unable to add config dir %v to watch: %w", u.dirPath, err)
	}
	log.Infof("Now watching %v and %v", u.parentPath, u.dirPath)
	u.started = true
	u.done = make(chan bool)
	go u.watchForUpdates()
	return nil
}

// Stop stops the auto-updating go-routine.
func (u *Updater) Stop() error {
	if !u.started {
		return errors.New("dflag: not updating")
	}
	u.done <- true
	_ = u.watcher.Remove(u.dirPath)
	_ = u.watcher.Remove(u.parentPath)
	return nil
}

func (u *Updater) readAll(dynamicOnly bool) error {
	files, err := os.ReadDir(u.dirPath)
	if err != nil {
		return fmt.Errorf("dflag: updater initialization: %w", err)
	}
	errorStrings := []string{}
	for _, f := range files {
		if strings.HasPrefix(path.Base(f.Name()), ".") {
			// skip random ConfigMap internals and dot files
			continue
		}
		fullPath := path.Join(u.dirPath, f.Name())
		log.S(log.Debug, "checking flag", log.Str("flag", f.Name()), log.Str("path", fullPath))
		if err := u.readFlagFile(fullPath, dynamicOnly); err != nil {
			if errors.Is(err, errFlagNotFound) {
				log.S(log.Warning, "config map for unknown flag", log.Str("flag", f.Name()), log.Str("path", fullPath))
				u.warnings.Add(1)
			} else if !errors.Is(err, errFlagNotDynamic) || !dynamicOnly {
				errorStrings = append(errorStrings, fmt.Sprintf("flag %v: %v", f.Name(), err.Error()))
				u.errors.Add(1)
			}
		}
	}
	if len(errorStrings) > 0 {
		return fmt.Errorf("encountered %d errors while parsing flags from directory  \n  %v",
			len(errorStrings), strings.Join(errorStrings, "\n"))
	}
	return nil
}

// Warnings returns the warnings count.
func (u *Updater) Warnings() int {
	return int(u.warnings.Load())
}

// Errors returns the errors count.
func (u *Updater) Errors() int {
	return int(u.errors.Load())
}

func (u *Updater) readFlagFile(fullPath string, dynamicOnly bool) error {
	flagName := path.Base(fullPath)
	flag := u.flagSet.Lookup(flagName)
	if flag == nil {
		return errFlagNotFound
	}
	if dynamicOnly && !dflag.IsFlagDynamic(flag) {
		return errFlagNotDynamic
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	if v := dflag.IsBinary(flag); v != nil {
		log.Infof("Updating binary %q to new blob (len %d)", flagName, len(content))
		err = v.SetV(content)
		if err != nil {
			return err
		}
		return nil
	}
	str := string(content)
	log.Infof("Updating %q to %q", flagName, str)
	// do not call flag.Value.Set, instead go through flagSet.Set to change "changed" state.
	return u.flagSet.Set(flagName, str)
}

func (u *Updater) watchForUpdates() {
	log.Infof("Background thread watching %s now running", u.dirPath)
	for {
		select {
		case event := <-u.watcher.Events:
			log.LogVf("ConfigMap got fsnotify %v ", event)
			if event.Name == u.dirPath || event.Name == path.Join(u.dirPath, k8sDataSymlink) { //nolint:nestif // to fix maybe later
				// case of the whole directory being re-symlinked
				switch event.Op {
				case fsnotify.Create:
					if err := u.watcher.Add(u.dirPath); err != nil { // add the dir itself.
						log.Errf("unable to add config dir %v to watch: %v", u.dirPath, err)
					}
					log.Infof("dflag: Re-reading flags after ConfigMap update.")
					if err := u.readAll( /* dynamicOnly */ true); err != nil {
						log.Errf("dflag: directory reload yielded errors: %v", err.Error())
					}
				case fsnotify.Remove, fsnotify.Chmod, fsnotify.Rename, fsnotify.Write:
				}
			} else if strings.HasPrefix(event.Name, u.dirPath) && !isK8sInternalDirectory(event.Name) {
				log.LogVf("ConfigMap got prefix %v", event)
				switch event.Op {
				case fsnotify.Create, fsnotify.Write, fsnotify.Rename, fsnotify.Remove:
					flagName := path.Base(event.Name)
					if err := u.readFlagFile(event.Name, true); err != nil {
						log.Errf("dflag: failed setting flag %s: %v", flagName, err.Error())
						u.errors.Add(1)
					}
				case fsnotify.Chmod:
				}
			}
		case <-u.done:
			return
		}
	}
}

func isK8sInternalDirectory(filePath string) bool {
	basePath := path.Base(filePath)
	return strings.HasPrefix(basePath, k8sInternalsPrefix)
}
