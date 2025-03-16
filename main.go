package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// container name
const (
	cName    = "lilContainer"
	maxProcs = "100"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go run <command> [args..]")
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		log.Fatal("unknown command")
	}
}

func run() {
	log.Printf("running parent process ID %d\n", os.Getpid())

	// recursively run this process again with namespaces set via clone/unshare so changes don't affect host
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// establish namespaces - equivalent with Linux `unshare` and 'clone' commands
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS | syscall.CLONE_NEWCGROUP,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings:  []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
		GidMappings:  []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
		//Chroot: "/mnt",
	}

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func child() {
	log.Printf("running child process ID %d\n", os.Getpid())

	initCGroups()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	syscall.Sethostname([]byte(cName))

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

// initCGroups creates control group for the container, e.g. limits its resource usage
func initCGroups() {
	cGroups := "/sys/fs/cgroup"
	err := os.Mkdir(filepath.Join(cGroups, cName), 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	// add `pids` to the new control group's controllers
	file, err := os.OpenFile(filepath.Join(cGroups, cName, "/cgroup.subtree_control"), os.O_WRONLY, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	_, err = file.WriteString("+pids")
	if err != nil {
		log.Fatal(err)
	}

	// set maximum processes allowed in the control group
	err = os.WriteFile(filepath.Join(cGroups, cName, "/pids.max"), []byte(maxProcs), 0700)
	if err != nil {
		log.Fatal(err)
	}

	// notify when the last process in the cgroup exits; can be used for cleanup tasks, e.g., /sys/fs/cgroup/pids/release_agent
	err = os.WriteFile(filepath.Join(cGroups, cName, "/notify_on_release"), []byte("1"), 0700)
	if err != nil {
		log.Fatal(err)
	}
	// move the existing process ID into container's control group process list
	err = os.WriteFile(filepath.Join(cGroups, cName, "/cgroup.procs"), []byte(strconv.Itoa(os.Getppid())), 0700)
	if err != nil {
		log.Fatal(err)
	}
}
