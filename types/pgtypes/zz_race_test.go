package pgtypes

import (
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestArrayScanConcurrent(t *testing.T) {
	bin := encodeBinary(t, pgtype.Int4ArrayOID, []int32{1, 2, 3})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				var a Array[int32]
				if err := a.Scan(bin); err != nil {
					t.Error(err)
				}
				var s Array[string]
				if err := s.Scan("{a,b}"); err != nil {
					t.Error(err)
				}
				var f Array[float64]
				if err := f.Scan(`{1.5}`); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
}
