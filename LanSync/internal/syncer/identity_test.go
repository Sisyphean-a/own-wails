package syncer

import (
	"sync"
	"testing"
)

func TestConcurrentFirstLaunchDoesNotOverwriteIdentity(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	success := make(chan string, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cert, err := loadIdentity(dir)
			if err == nil {
				success <- fingerprint(cert.Certificate[0])
			}
		}()
	}
	wg.Wait()
	close(success)
	cert, err := loadIdentity(dir)
	must(t, err)
	count := 0
	for id := range success {
		count++
		if id != fingerprint(cert.Certificate[0]) {
			t.Fatal("并发创建覆盖设备身份")
		}
	}
	if count == 0 {
		t.Fatal("并发启动未能创建有效身份")
	}
}
