//go:build unix

package resourcelifetime

// The Unix implementation supplies a non-retaining summary for the process
// attributes, so inheriting descriptors does not settle the parent's cleanup.
// Windows leaves that handoff opaque; the portable fixture records that gap.

import "os"

func standardStreamsHandedToChild(name string) error {
	files := [3]*os.File{}
	var err error
	files[0], err = os.Open(os.DevNull) // want "owned resource from os.Open is not released on every return path"
	if err != nil {
		return err
	}
	files[1] = os.Stdout
	files[2] = os.Stderr
	attr := &os.ProcAttr{Files: []*os.File{files[0], files[1], files[2]}}
	process, err := os.StartProcess(name, []string{name}, attr)
	if err != nil {
		return err
	}
	return process.Release()
}
