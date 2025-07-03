package docker

import (
	"syscall"
	"unsafe"
)

type TerminalSize struct {
	Height uint16
	Width  uint16
}

func GetTerminalSize() (*TerminalSize, error) {
	var size TerminalSize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&size)),
	)
	if errno != 0 {
		return nil, errno
	}
	return &size, nil
}

func SetRawTerminal(fd uintptr) (*syscall.Termios, error) {
	var oldState syscall.Termios
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGETA, uintptr(unsafe.Pointer(&oldState))); err != 0 {
		return nil, err
	}

	newState := oldState
	newState.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	newState.Iflag &^= syscall.ICRNL | syscall.INLCR | syscall.IGNCR
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0

	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSETA, uintptr(unsafe.Pointer(&newState))); err != 0 {
		return nil, err
	}

	return &oldState, nil
}

func RestoreTerminal(fd uintptr, state *syscall.Termios) error {
	if state == nil {
		return nil
	}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSETA, uintptr(unsafe.Pointer(state)))
	if err != 0 {
		return err
	}
	return nil
}

var BuildDockerImage = func() error {
	// This will be overridden by image.go
	return nil
}