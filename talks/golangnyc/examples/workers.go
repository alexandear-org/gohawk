package examples

import "sync"

// #region unjoined
func refresh(update func()) {
	var group sync.WaitGroup
	group.Add(1)
	go func() {
		defer group.Done()
		update()
	}()
}

// #endregion unjoined

// #region joined
func refreshSafely(update func()) {
	var group sync.WaitGroup
	group.Add(1)
	go func() {
		defer group.Done()
		update()
	}()
	group.Wait()
}

// #endregion joined
