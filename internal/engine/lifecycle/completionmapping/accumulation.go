package completionmapping

// Paths folds the paths of completing calls seen during one body's coverage.
// An exported completion claim needs to say where the target was settled:
// closing j.out does not establish that j itself was closed. Keep an exact
// path only while every completing call agrees. Summaries, invoked callbacks,
// unnamed receivers and conflicting paths leave the final path unknown.
// Path retains its payload even after evidence becomes inexact.
type Paths struct {
	Path  string
	Seen  bool
	Exact bool
}

// Record folds one completing call. A nil receiver means no body is collecting
// paths. An unnamed call still marks the body seen and invalidates exactness.
func (paths *Paths) Record(path string, known bool) {
	if paths != nil {
		*paths = FoldPath(*paths, path, known)
	}
}

// Merge folds another body's path evidence without inventing an observation
// when that body supplied none.
func (paths *Paths) Merge(other Paths) {
	if other.Seen {
		paths.Record(other.Path, other.Exact)
	}
}

// Known reports whether every completing call named the same static path.
func (paths *Paths) Known() bool { return paths.Seen && paths.Exact }

type foldPathSignature func(Paths, string, bool) Paths

var _ foldPathSignature = FoldPath

func differentPath(left, right string) bool { return left != right }
