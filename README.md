# Golang 实现带过期时间的单机锁


单机锁要实现的目标：

1. 加锁：会记录一个锁的拥有者 `owner`
2. 解锁：只有锁的拥有者才能解锁
3. 如果有设定锁的超时时间，到时间自动解锁（避免忘记解锁）

代码很简单，直接贴源码。相信聪明的你一看就懂

```go
package expiredlock

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

/*
	实现一个带自动过期时间的单机锁

	只有锁的拥有者才能解锁
*/

type ExpiredLocker struct {
	mutex sync.Mutex // 单机锁

	processMutex sync.Mutex         // 保护下面的字段
	owner        string             // 锁的拥有者（用进程id+协程id作为拥有者身份标记）
	cancel       context.CancelFunc //取消函数（让协程停止）
}

func (l *ExpiredLocker) Lock(expireTime time.Duration) {
	// 加锁
	l.mutex.Lock()

	l.processMutex.Lock()
	defer l.processMutex.Unlock()
	// 加锁成功的唯一标识
	l.owner = getPidGid()

	if expireTime <= 0 { // 如果没有过期时间，说明不需要创建带过期时间的锁
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), expireTime)
	l.cancel = cancel

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded { // 过期取消
			l.unlocker(l.owner)
		}
	}()
}

func (l *ExpiredLocker) UnLock() {
	l.unlocker(getPidGid()) // 用户主动调用解锁（监视过期的协程,如果有启动，需主动取消掉）
}

func (l *ExpiredLocker) unlocker(owner string) {

	l.processMutex.Lock()
	defer l.processMutex.Unlock()
	// 必须是锁的拥有者
	if l.owner != "" && l.owner == owner {
		l.owner = ""

		if l.cancel != nil { // 说明是带过期时间的锁
			l.cancel() // 目的让协程也停止
			l.cancel = nil
		}
		l.mutex.Unlock() // 解锁
	}
}

// 进程id + 协程id 作为唯一标识
func getPidGid() string {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)

	return fmt.Sprintf("%d_%d", os.Getpid(), n)
}


```