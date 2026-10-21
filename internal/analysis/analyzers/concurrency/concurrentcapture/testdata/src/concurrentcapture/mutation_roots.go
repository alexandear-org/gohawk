package concurrentcapture

// Root recognition follows parentheses and map indexes only. Neighboring
// array and slice writes retain the existing conservative gap; selectors do
// not make the containing object a reportable direct local mutation.
func parenthesizedMapRoot(items []int) {
	seen := map[int]bool{}
	for _, item := range items {
		go func() {
			seen[item] = true // want "captured local seen is mutated by goroutines launched repeatedly"
		}()
	}
}

func nestedMapRoot(items []int) {
	seen := map[int]map[int]bool{0: {}}
	for _, item := range items {
		go func() {
			seen[0][item] = true // want "captured local seen is mutated by goroutines launched repeatedly"
		}()
	}
}

func arrayAndSliceRoots(items []int) {
	var array [1]int
	slice := make([]int, 1)
	for range items {
		go func() {
			array[0] = 1
			slice[0] = 1
		}()
	}
}

func selectedMapRoot(items []int) {
	owner := struct{ seen map[int]bool }{seen: map[int]bool{}}
	for _, item := range items {
		go func() { owner.seen[item] = true }()
	}
}
