package ui

import (
	"fmt"
	"os"
	"time"
)

// requestRepairStore offers to move a damaged connection file aside and start
// with an empty store, which is the only safe way to unblock writes.
func (s *screen) requestRepairStore() {
	aware, ok := s.ws.store.(pathStore)
	if !ok {
		s.setStatus("This store does not expose a file to repair.")
		return
	}
	path := aware.Path()
	s.confirm("Repair connection file",
		fmt.Sprintf("Move %s aside as a backup and start with an empty store? Existing connections stay in the backup file.", path),
		func() { s.repairStore(path) }, nil)
}

// repairStore renames the damaged file and writes an empty document.
func (s *screen) repairStore(path string) {
	backup := path + ".bak-" + time.Now().UTC().Format("20060102T150405Z")
	if err := os.Rename(path, backup); err != nil && !os.IsNotExist(err) {
		s.setProblem("Could not move the damaged file aside: " + err.Error())
		return
	}
	if err := s.ws.reset(); err != nil {
		s.setProblem("Could not write a new connection file: " + err.Error())
		return
	}
	s.setWritable(true)
	s.reload()
	s.setStatus("Connection file repaired. The previous content is in " + backup + ".")
}
