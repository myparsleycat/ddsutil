package ddsutil

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// minParallelWork is the number of work units (blocks or pixels) below which
// parallelFor runs on the calling goroutine.
const minParallelWork = 256

// parallelFor calls fn for every row in [0, rows) using up to GOMAXPROCS
// goroutines. Each row represents workPerRow units of work; small jobs run
// on the calling goroutine. Rows are handed out dynamically because the cost
// of a row can vary.
func parallelFor(rows, workPerRow int, fn func(row int)) {
	workers := min(runtime.GOMAXPROCS(0), rows)
	if workers <= 1 || rows*workPerRow < minParallelWork {
		for row := 0; row < rows; row++ {
			fn(row)
		}
		return
	}

	var next atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for {
				row := int(next.Add(1) - 1)
				if row >= rows {
					return
				}
				fn(row)
			}
		}()
	}
	wg.Wait()
}
