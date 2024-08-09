package expiredlock

import (
	"sync"
	"testing"
	"time"
)

func TestExpiredLock(t *testing.T) {

	lock := ExpiredLocker{}

	start := time.Now()
	lock.Lock(1 * time.Second) // 加锁

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {

		lock.UnLock() // 无法解锁(因为当前协程不是锁的拥有者)

		lock.Lock(0) // 需要等到 1s 超时，锁自动释放了，才能加锁成功
		t.Log(time.Since(start).Seconds())
		lock.UnLock()
		wg.Done()
	}()
	wg.Wait()
}
