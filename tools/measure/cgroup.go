package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// treeSampleInterval bounds how often anonymous memory is read. A shorter
// interval catches briefer peaks at the cost of more reads; per-package vet
// processes live for hundreds of milliseconds, so 50ms sees each of them.
const treeSampleInterval = 50 * time.Millisecond

// treeMemory reports the memory of every process in the helper's own cgroup.
// It is meaningful only when the cgroup is dedicated to this measurement,
// such as a transient systemd scope; the caller asserts that with -cgroup.
//
// memory.peak includes page cache charged during the run, such as build
// cache reads. Sampled anonymous memory is closer to what the processes
// need, but can miss a peak shorter than the sample interval.
type treeMemory struct {
	directory string
	stop      chan struct{}
	done      chan struct{}
	anonPeak  int64
	sampleErr error
}

func startTreeMemory() (*treeMemory, error) {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return nil, fmt.Errorf("read own cgroup: %w", err)
	}
	relative, found := strings.CutPrefix(strings.TrimSpace(string(data)), "0::")
	if !found || strings.Contains(relative, "\n") {
		return nil, errors.New("own cgroup is not a single cgroup v2 hierarchy")
	}
	tree := &treeMemory{
		directory: filepath.Join("/sys/fs/cgroup", relative),
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	if _, err := tree.anon(); err != nil {
		return nil, err
	}
	go tree.sample()
	return tree, nil
}

// sample runs until finish closes stop; finish waits on done.
func (t *treeMemory) sample() {
	defer close(t.done)
	ticker := time.NewTicker(treeSampleInterval)
	defer ticker.Stop()
	for {
		anon, err := t.anon()
		if err != nil {
			t.sampleErr = err
			return
		}
		t.anonPeak = max(t.anonPeak, anon)
		select {
		case <-t.stop:
			return
		case <-ticker.C:
		}
	}
}

// finish stops sampling and returns the cgroup peak and sampled anonymous
// peak, both in KiB.
func (t *treeMemory) finish() (peakKiB, anonPeakKiB int64, err error) {
	close(t.stop)
	<-t.done
	if t.sampleErr != nil {
		return 0, 0, t.sampleErr
	}
	data, err := os.ReadFile(filepath.Join(t.directory, "memory.peak"))
	if err != nil {
		return 0, 0, fmt.Errorf("read cgroup memory.peak: %w", err)
	}
	peak, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse cgroup memory.peak: %w", err)
	}
	return peak / 1024, t.anonPeak / 1024, nil
}

func (t *treeMemory) anon() (int64, error) {
	data, err := os.ReadFile(filepath.Join(t.directory, "memory.stat"))
	if err != nil {
		return 0, fmt.Errorf("read cgroup memory.stat: %w", err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		value, found := strings.CutPrefix(scanner.Text(), "anon ")
		if !found {
			continue
		}
		amount, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse cgroup anonymous memory: %w", err)
		}
		return amount, nil
	}
	return 0, errors.New("cgroup memory.stat has no anon entry")
}
