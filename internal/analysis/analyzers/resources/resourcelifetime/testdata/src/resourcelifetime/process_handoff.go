package resourcelifetime

// Dropping a file after building a local stream slice remains diagnostic on
// every platform. The process handoff case lives in process_handoff_unix.go:
// Windows's imported os.StartProcess evidence is opaque, so the analyzer cannot
// establish the parent's missing cleanup there and deliberately stays silent.

import "os"

func streamsOnlyInspected(name string) error {
	files := [3]*os.File{}
	var err error
	files[0], err = os.Open(os.DevNull) // want "owned resource from os.Open is not released on every return path"
	if err != nil {
		return err
	}
	_ = []*os.File{files[0]}
	return nil
}
