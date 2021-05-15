package backend

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	cgroupsRoot = "/sys/fs/cgroup/"
	cgroupName  = "worker-api"
)

// CGroups is a map of subsystem -> filename -> limit used to create the cgroups.
type CGroups struct {
	Limits map[string]map[string]string
}

// SetupCGroup creates the given cgroups and associated limits.
func (c *CGroups) SetupCGroup() error {

	// check if the cgroups directory already exists and is populated
	files, err := ioutil.ReadDir(cgroupsRoot)
	if err != nil || len(files) < 0 {
		// if not mount the relevant subsystem
		// TODO test this
		err = unix.Mount("worker-api", cgroupsRoot, "cgroup", 0, "")
		if err != nil {
			log.Fatalf("error mounting %s: %s\n", cgroupsRoot, err)
			return err
		}
	}

	for system, limits := range c.Limits {
		systemPath := path.Join(cgroupsRoot, system, cgroupName)
		err = os.MkdirAll(systemPath, 0555)
		if err != nil {
			log.WithError(err).Fatalf("unable to create cgroup directory for %s", system)
			return err
		}

		for filename, limit := range limits {
			limitFilename := strings.Join([]string{system, filename}, ".")
			err = ioutil.WriteFile(path.Join(systemPath, limitFilename), []byte(limit), 0600)
			if err != nil {
				log.WithError(err).Fatalf("unable to write to %s", filename)
				return err
			}
		}
	}
	return nil
}

// Cleanup removes the worker-api specific cgroups from the system.
func (c *CGroups) Cleanup() {
	for system := range c.Limits {
		err := os.RemoveAll(path.Join(cgroupsRoot, system, cgroupName))
		if err != nil {
			log.WithError(err).Fatalf("unable to remove %s cgroups", system)
		}
	}
}

// AddPid adds the given pid to the cgroup.
func (c *CGroups) AddPid(pid int) error {
	for system := range c.Limits {
		f, err := os.OpenFile(path.Join(cgroupsRoot, system, cgroupName, "cgroup.procs"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.WithError(err).Fatalf("%d: unable to open %s/cgroup.procs for writing", pid, system)
			return err
		}
		defer f.Close()

		if _, err = f.Write([]byte(strconv.Itoa(pid))); err != nil {
			log.WithError(err).Fatalf("%d: unable to write to %s/cgroup.procs", pid, system)
			return err
		}
	}
	return nil
}
