#!/usr/bin/env bash

set -euo pipefail

usage() {
	cat <<'EOF'
Usage: scripts/benchmark-dogfood.sh [options]

Measure whole-repository gohawk runtime, CPU time and memory.

go vet caches each package's analysis by the vet tool's build ID, so a
repeated run with the same binary replays results instead of analyzing.
The warm-up run fills the compile cache; every measured run then uses a
freshly linked gohawk binary, so it reuses compiled packages but analyzes
every package again.

Options:
  --manifest PATH       Repository manifest (default: benchmarks/repositories.tsv)
  --only NAME           Measure only the named manifest entry (repeatable)
  --runs COUNT          Measured runs per repository (default: 3)
  --output DIRECTORY    New result directory (default: benchmark-results/<UTC time>)
  --gohawk-arg=ARG      Pass an analyzer-selection argument to gohawk (repeatable)
  --profile             After the measured runs, make one profiled run per
                        repository and write per-unit timings, an analyzer
                        summary and merged CPU profiles under profiles/
  --keep-workdir        Keep temporary repository checkouts
  -h, --help            Show this help
EOF
}

fail() {
	printf 'benchmark-dogfood: %s\n' "$*" >&2
	exit 1
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

selected_repository() {
	local candidate=$1
	local selected

	if ((${#only_repositories[@]} == 0)); then
		return 0
	fi
	for selected in "${only_repositories[@]}"; do
		if [[ $candidate == "$selected" ]]; then
			return 0
		fi
	done
	return 1
}

run_gohawk() {
	local checkout=$1
	local package_pattern=$2
	local log_file=$3

	set +e
	(
		cd "$checkout"
		"$gohawk_binary" "${gohawk_arguments[@]}" "$package_pattern"
	) >"$log_file" 2>&1
	command_status=$?
	set -e
}

repository_root=$(git rev-parse --show-toplevel 2>/dev/null) || fail "run from a gohawk checkout"
manifest_path="$repository_root/benchmarks/repositories.tsv"
runs=3
output_directory="$repository_root/benchmark-results/$(date -u +%Y%m%dT%H%M%SZ)"
keep_workdir=false
profile=false
declare -a only_repositories=()
declare -a gohawk_arguments=()

while (($# > 0)); do
	case $1 in
	--manifest)
		(($# >= 2)) || fail "--manifest requires a path"
		manifest_path=$2
		shift 2
		;;
	--only)
		(($# >= 2)) || fail "--only requires a repository name"
		only_repositories+=("$2")
		shift 2
		;;
	--runs)
		(($# >= 2)) || fail "--runs requires a count"
		runs=$2
		shift 2
		;;
	--output)
		(($# >= 2)) || fail "--output requires a directory"
		output_directory=$2
		shift 2
		;;
	--gohawk-arg=*)
		gohawk_arguments+=("${1#*=}")
		shift
		;;
	--keep-workdir)
		keep_workdir=true
		shift
		;;
	--profile)
		profile=true
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		fail "unknown option: $1"
		;;
	esac
done

[[ $runs =~ ^[1-9][0-9]*$ ]] || fail "--runs must be a positive integer"
[[ -f $manifest_path ]] || fail "manifest not found: $manifest_path"
[[ ! -e $output_directory ]] || fail "output path already exists: $output_directory"

require_command git
require_command go
require_command awk
require_command wc

mkdir -p "$output_directory/logs"
work_directory=$(mktemp -d "${TMPDIR:-/tmp}/gohawk-benchmark.XXXXXX")

cleanup() {
	if [[ $keep_workdir == true ]]; then
		printf 'Temporary repositories retained at %s\n' "$work_directory"
		return
	fi
	case $work_directory in
	"${TMPDIR:-/tmp}"/gohawk-benchmark.*)
		rm -rf -- "$work_directory"
		;;
	*)
		printf 'Refusing to remove unexpected work directory: %s\n' "$work_directory" >&2
		;;
	esac
}
trap cleanup EXIT

gohawk_binary="$work_directory/gohawk"
measure_binary="$work_directory/measure"
summary_binary="$work_directory/profilesummary"
sh "$repository_root/scripts/hyrule.sh" generate
go build -trimpath -o "$gohawk_binary" "$repository_root"
go build -trimpath -o "$measure_binary" "$repository_root/tools/measure"
go build -trimpath -o "$summary_binary" "$repository_root/tools/profilesummary"

# fresh_gohawk links a gohawk binary with a build ID go vet has not seen, so
# none of its cached analysis results apply. The -X value names no variable;
# it only changes the link inputs and therefore the binary. The value includes
# the unique work directory: a label reused by an earlier benchmark of the
# same source would rebuild an identical binary and hit that run's cache.
fresh_gohawk() {
	local label=$1
	local binary="$work_directory/gohawk-$label"
	go build -trimpath -ldflags="-X=main.benchmarkRun=${work_directory##*/}-$label" -o "$binary" "$repository_root"
	printf '%s\n' "$binary"
}

# merge_profiles combines every unit's CPU profile into one. It merges in
# batches so a run with thousands of units stays within argument limits.
merge_profiles() {
	local directory=$1
	local merged=$2
	local batch_directory="$directory.batches"
	local -a profiles=()
	local -a batches=()
	local index

	mapfile -d '' -t profiles < <(find "$directory" -name '*.pprof' -size +0 -print0 | sort -z)
	((${#profiles[@]} > 0)) || fail "no CPU profiles were written under $directory"
	mkdir -p "$batch_directory"
	for ((index = 0; index < ${#profiles[@]}; index += 500)); do
		batches+=("$batch_directory/batch-$index.pb.gz")
		go tool pprof -proto "${profiles[@]:index:500}" >"${batches[-1]}" 2>/dev/null
	done
	go tool pprof -proto "${batches[@]}" >"$merged" 2>/dev/null
	rm -rf -- "$batch_directory"
}

# A transient systemd scope gives each measured run its own cgroup, so the
# helper can report memory for the whole process tree rather than only the
# largest process. Without one, those columns are unknown.
declare -a measure_scope=()
declare -a measure_flags=()
if command -v systemd-run >/dev/null 2>&1 && systemd-run --user --scope --quiet --collect true >/dev/null 2>&1; then
	measure_scope=(systemd-run --user --scope --quiet --collect)
	measure_flags=(-cgroup)
else
	printf 'Warning: no systemd user scope; process-tree memory will be unknown.\n' >&2
fi

gohawk_revision=$(git -C "$repository_root" rev-parse HEAD)
if [[ -n $(git -C "$repository_root" status --porcelain) ]]; then
	gohawk_tree=dirty
	printf 'Warning: the gohawk worktree is dirty; results will not be fully reproducible.\n' >&2
else
	gohawk_tree=clean
fi

go_version=$(go version)
host_description=$(uname -a)
processor_count=$(getconf _NPROCESSORS_ONLN 2>/dev/null || printf 'unknown')
memory_description=$(awk '/^MemTotal:/ { print $2 " " $3 }' /proc/meminfo 2>/dev/null || true)
if [[ -z $memory_description ]]; then
	memory_description=unknown
fi

{
	printf '# Dogfooding benchmark\n\n'
	printf -- '- Generated: `%s`\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	printf -- '- gohawk revision: `%s` (%s worktree)\n' "$gohawk_revision" "$gohawk_tree"
	printf -- '- Go: `%s`\n' "$go_version"
	printf -- '- Host: `%s`\n' "$host_description"
	printf -- '- Logical processors: `%s`\n' "$processor_count"
	printf -- '- Physical memory: `%s`\n' "$memory_description"
	printf -- '- Measured runs per repository: `%s`\n' "$runs"
	if ((${#gohawk_arguments[@]} == 0)); then
		printf -- '- gohawk profile: default\n'
	else
		printf -- '- Additional gohawk arguments:'
		printf ' `%s`' "${gohawk_arguments[@]}"
		printf '\n'
	fi
	printf '\nClone time, dependency downloads, binary construction, package counting, and one warm-up run are excluded. '
	printf 'The warm-up fills the compile cache; each measured run links a fresh gohawk binary so go vet cannot replay cached analysis. '
	printf 'CPU time covers every process the run waited for. Tree peak is the cgroup memory.peak, which includes page cache; '
	printf 'tree anon is sampled anonymous memory; largest process is the biggest single-process peak RSS.\n\n'
	printf '| Repository | Revision | Packages | Run | Wall time (s) | CPU time (s) | Parallelism | Tree peak (MiB) | Tree anon (MiB) | Largest process (MiB) | Exit | Output |\n'
	printf '|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|\n'
} >"$output_directory/summary.md"

printf 'repository,revision,packages,run,wall_seconds,cpu_seconds,tree_peak_kib,tree_anon_peak_kib,peak_rss_kib,exit_code,log\n' \
	>"$output_directory/results.csv"

mebibytes() {
	if [[ $1 == unknown ]]; then
		printf 'unknown'
		return
	fi
	awk -v kib="$1" 'BEGIN { printf "%.1f", kib / 1024 }'
}

measured_repositories=0
while IFS=$'\t' read -r repository_name repository_url requested_revision package_pattern extra; do
	[[ -n $repository_name ]] || continue
	[[ $repository_name == \#* ]] && continue
	[[ -z ${extra:-} ]] || fail "manifest row for $repository_name has more than four fields"
	[[ -n ${repository_url:-} && -n ${requested_revision:-} && -n ${package_pattern:-} ]] || \
		fail "manifest row for $repository_name must have four tab-separated fields"
	[[ $repository_name =~ ^[a-zA-Z0-9._-]+$ ]] || \
		fail "manifest repository name contains unsafe characters: $repository_name"
	selected_repository "$repository_name" || continue

	checkout="$work_directory/repositories/$repository_name"
	mkdir -p "$checkout"
	git -C "$checkout" init --quiet
	if [[ $repository_url == . ]]; then
		repository_url=$repository_root
		requested_revision=$(git -C "$repository_root" rev-parse "${requested_revision}^{commit}")
	fi
	git -C "$checkout" remote add origin "$repository_url"
	git -C "$checkout" fetch --quiet --depth=1 origin "$requested_revision"
	git -C "$checkout" checkout --quiet --detach FETCH_HEAD
	resolved_revision=$(git -C "$checkout" rev-parse HEAD)

	package_log="$output_directory/logs/$repository_name-packages.log"
	if ! package_count=$(
		cd "$checkout"
		go list "$package_pattern" 2>"$package_log" | wc -l
	); then
		fail "go list failed for $repository_name; see $package_log"
	fi
	package_count=${package_count//[[:space:]]/}

	warmup_log="$output_directory/logs/$repository_name-warmup.log"
	run_gohawk "$checkout" "$package_pattern" "$warmup_log"
	if [[ $command_status -ne 0 && $command_status -ne 3 ]]; then
		fail "warm-up failed for $repository_name with exit $command_status; see $warmup_log"
	fi

	for ((run = 1; run <= runs; run++)); do
		run_log="$output_directory/logs/$repository_name-run-$run.log"
		timing_file="$work_directory/$repository_name-run-$run.time"
		run_binary=$(fresh_gohawk "$repository_name-$run")
		set +e
		(
			cd "$checkout"
			"${measure_scope[@]}" "$measure_binary" "${measure_flags[@]}" -output "$timing_file" -- \
				"$run_binary" "${gohawk_arguments[@]}" "$package_pattern"
		) >"$run_log" 2>&1
		command_status=$?
		set -e
		if [[ $command_status -ne 0 && $command_status -ne 3 ]]; then
			fail "measurement failed for $repository_name with exit $command_status; see $run_log"
		fi
		IFS=$'\t' read -r wall_seconds peak_rss_kib reported_status cpu_seconds tree_peak_kib tree_anon_kib <"$timing_file"
		[[ $reported_status == "$command_status" ]] || \
			fail "time reported exit $reported_status for $repository_name, command returned $command_status"
		parallelism=$(awk -v cpu="$cpu_seconds" -v wall="$wall_seconds" 'BEGIN { printf "%.1f", cpu / wall }')
		log_name="logs/$repository_name-run-$run.log"
		printf '%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n' \
			"$repository_name" "$resolved_revision" "$package_count" "$run" \
			"$wall_seconds" "$cpu_seconds" "$tree_peak_kib" "$tree_anon_kib" "$peak_rss_kib" "$command_status" "$log_name" \
			>>"$output_directory/results.csv"
		printf '| %s | `%s` | %s | %s | %s | %s | %s | %s | %s | %s | %s | [%s](%s) |\n' \
			"$repository_name" "${resolved_revision:0:12}" "$package_count" "$run" \
			"$wall_seconds" "$cpu_seconds" "$parallelism" "$(mebibytes "$tree_peak_kib")" "$(mebibytes "$tree_anon_kib")" \
			"$(mebibytes "$peak_rss_kib")" "$command_status" "log" "$log_name" \
			>>"$output_directory/summary.md"
	done

	if [[ $profile == true ]]; then
		profile_directory="$output_directory/profiles/$repository_name"
		profile_log="$output_directory/logs/$repository_name-profile.log"
		profile_binary=$(fresh_gohawk "$repository_name-profile")
		set +e
		(
			cd "$checkout"
			GOHAWK_PROFILE_DIR=$profile_directory "$profile_binary" "${gohawk_arguments[@]}" "$package_pattern"
		) >"$profile_log" 2>&1
		command_status=$?
		set -e
		if [[ $command_status -ne 0 && $command_status -ne 3 ]]; then
			fail "profiled run failed for $repository_name with exit $command_status; see $profile_log"
		fi
		"$summary_binary" "$profile_directory" >"$output_directory/profiles/$repository_name.md"
		merged_profile="$output_directory/profiles/$repository_name-cpu.pb.gz"
		merge_profiles "$profile_directory" "$merged_profile"
		# The analyzer label splits merged samples by analyzer; unlabelled
		# samples are setup: parsing, type checking and fact decoding.
		go tool pprof -top -nodecount=60 "$merged_profile" >"$output_directory/profiles/$repository_name-cpu-top.txt" 2>&1
		go tool pprof -tags "$merged_profile" >"$output_directory/profiles/$repository_name-cpu-analyzers.txt" 2>&1
	fi

	measured_repositories=$((measured_repositories + 1))
done <"$manifest_path"

if ((measured_repositories == 0)); then
	fail "no manifest entries matched the requested repositories"
fi

printf 'Benchmark results written to %s\n' "$output_directory"
