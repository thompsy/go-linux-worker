package backend

import (
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
		log.Fatal(err)
	}

	// TODO Create this dir using bin data for each command
	err = syscall.Chroot("/tmp/alpine/")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir("/")
	if err != nil {
		log.Fatal(err)
	}

	err = syscall.Mount("proc", "proc", "proc", 0, "")
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	err = syscall.Unmount("proc", 0)
	if err != nil {
		log.Fatal(err)
	}

}
