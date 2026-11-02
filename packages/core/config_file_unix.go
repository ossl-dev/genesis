//go:build !windows

package core

import (
	"os"
	"syscall"
)

// Nonblocking open prevents a replaced FIFO from hanging between stat and open.
const configOpenFlags = os.O_RDONLY | syscall.O_NONBLOCK
