package backend

import (
	"path/filepath"
	"os"
	"os/exec"
	"strings"
	"syscall" //TODO replace syscall usage with newer x/sys/unix versions

	log "github.com/sirupsen/logrus"
)

// Exec runs the given command in an isolated environment.
// TODO: limit the amount of logging here to prevent leaking implementation
// details to clients.
func Exec(command string) {
	parts := strings.Split(command, " ")
	//TODO validate that there is actually a command
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := syscall.Sethostname([]byte("container"))
	if err != nil {
		//TODO on error all of these calls should exit the process and output the same generic error message
		log.Fatal(err)
	}

	// Create a temp directory to mount the container filesystem on
	tmpDir, err := os.MkdirTemp(os.TempDir(), "worker-api-*")
	if err != nil {
		log.Fatal(err)
	}
	//TODO how do we clean this up after we've chrooted?
	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Create a new in-memory filesystem mounted at the temp directory
	err = syscall.Mount("tmpfs", tmpDir, "tmpfs", 0, "")
	if err != nil {
		log.Fatal(err)
	}

	// Create a directory to mount /proc on
	err = os.Mkdir(filepath.Join(tmpDir, "proc"), 0755)
	if err != nil {
		log.Fatal(err)
	}

	//TODO use bindata for this?
	// Extract the Apline filesystem into the container filesystem
	copy := exec.Command("tar", "-xf", "/tmp/alpine.tar.gz", "-C", tmpDir)
	err = copy.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Chroot into the newly created filesystem
	err = syscall.Chroot(tmpDir)
	if err != nil {
		log.Fatal(err)
	}

	// Change directory to /
	err = os.Chdir("/")
	if err != nil {
		log.Fatal(err)
	}

	// Mount the proc filesystem
	err = syscall.Mount("proc", "proc", "proc", 0, "")
	if err != nil {
		log.Fatal(err)
	}

	// Now that we've setup our container we can run the actual client submitted command
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Remove the proc mount once we're finished
	err = syscall.Unmount("proc", 0)
	if err != nil {
		log.Fatal(err)
	}

}
