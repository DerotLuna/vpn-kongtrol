//go:build unix

package api

import "syscall"

// openNoFollow makes open fail on a symlink at the last path element.
const openNoFollow = syscall.O_NOFOLLOW
