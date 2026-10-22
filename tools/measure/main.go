// Command measure runs a command and records its wall time, CPU time, peak
// resident memory, and exit status for the dogfooding benchmark harness.
//
// The output is one tab-separated line: wall seconds, largest single-process
// peak RSS in KiB, exit status, CPU seconds, cgroup peak in KiB, and sampled
// anonymous peak in KiB. CPU time and the single-process peak cover the
// command and every descendant it waited for. The two cgroup columns are
// "unknown" unless -cgroup asserts that the helper runs in a dedicated cgroup.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(arguments []string) int {
	flags := flag.NewFlagSet("measure", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	output := flags.String("output", "", "write tab-separated measurements to this file")
	dedicatedCgroup := flags.Bool("cgroup", false, "the helper runs in a dedicated cgroup v2; report memory of the whole process tree")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	commandArguments := flags.Args()
	if *output == "" || len(commandArguments) == 0 {
		fmt.Fprintln(os.Stderr, "usage: measure -output FILE -- COMMAND [ARG ...]")
		return 2
	}

	var tree *treeMemory
	if *dedicatedCgroup {
		var err error
		if tree, err = startTreeMemory(); err != nil {
			fmt.Fprintf(os.Stderr, "measure: %v\n", err)
			return 1
		}
	}

	command := exec.CommandContext(context.Background(), commandArguments[0], commandArguments[1:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	started := time.Now()
	err := command.Run()
	elapsed := time.Since(started)
	treePeak, treeAnonPeak := "unknown", "unknown"
	if tree != nil {
		peak, anonPeak, treeErr := tree.finish()
		if treeErr != nil {
			fmt.Fprintf(os.Stderr, "measure: inspect process-tree memory: %v\n", treeErr)
			return 1
		}
		treePeak, treeAnonPeak = strconv.FormatInt(peak, 10), strconv.FormatInt(anonPeak, 10)
	}
	if command.ProcessState == nil {
		fmt.Fprintf(os.Stderr, "measure: start command: %v\n", err)
		return 1
	}

	peakRSS, rssErr := peakRSSKiB(command.ProcessState.SysUsage(), runtime.GOOS)
	if rssErr != nil {
		fmt.Fprintf(os.Stderr, "measure: inspect peak resident memory: %v\n", rssErr)
		return 1
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			fmt.Fprintf(os.Stderr, "measure: run command: %v\n", err)
			return 1
		}
		exitCode = exitError.ExitCode()
	}

	cpu := command.ProcessState.UserTime() + command.ProcessState.SystemTime()
	measurement := fmt.Sprintf("%.6f\t%d\t%d\t%.6f\t%s\t%s\n",
		elapsed.Seconds(), peakRSS, exitCode, cpu.Seconds(), treePeak, treeAnonPeak)
	if err := os.WriteFile(*output, []byte(measurement), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "measure: write result: %v\n", err)
		return 1
	}
	return exitCode
}

// peakRSSKiB uses the Maxrss field exposed by os.ProcessState without tying
// the helper to one syscall.Rusage definition. Unix kernels report this value
// in KiB except Darwin, which reports bytes.
func peakRSSKiB(usage any, goos string) (int64, error) {
	value := reflect.ValueOf(usage)
	if !value.IsValid() {
		return 0, errors.New("process resource usage is unavailable")
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, errors.New("process resource usage is unavailable")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0, fmt.Errorf("unexpected resource usage type %T", usage)
	}
	maxRSS := value.FieldByName("Maxrss")
	if !maxRSS.IsValid() || !maxRSS.CanInt() {
		return 0, fmt.Errorf("resource usage type %T has no integer Maxrss field", usage)
	}
	result := maxRSS.Int()
	if goos == "darwin" {
		result /= 1024
	}
	return result, nil
}
