package cchan_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jae2274/goutils/cchan"
	"github.com/stretchr/testify/require"
)

func TestTooMuchError(t *testing.T) {

	// 가장 최근 10개의 에러가 2초 이내에 발생하면 종료
	errCount := uint(10)
	duration := time.Second * 1

	t.Run("가장 최근 10개의 에러가 1초 이내에 발생하면 종료", func(t *testing.T) {
		errorChan := make(chan error, 100)

		ctx, tooMuchErrFunc := context.WithCancel(context.Background())
		go cchan.TooMuchError(errCount, duration, errorChan, tooMuchErrFunc)

		for i := 0; i < 9; i++ {
			errorChan <- errors.New("error")
		}

		time.Sleep(duration - (time.Millisecond * time.Duration(200))) // 0.8초 대기
		errorChan <- errors.New("last error")                          // 10번째 에러

		assertDone(t, ctx)
	})

	t.Run("가장 최근 10개의 에러가 5초 이내에 발생하지 않으면 종료되지 않음", func(t *testing.T) {
		errorChan := make(chan error, 100)

		ctx, tooMuchErrFunc := context.WithCancel(context.Background())
		go cchan.TooMuchError(errCount, duration, errorChan, tooMuchErrFunc)

		for i := 0; i < 9; i++ {
			errorChan <- errors.New("error")
		}

		time.Sleep(duration + (time.Millisecond * time.Duration(100))) // 1.1초 대기
		errorChan <- errors.New("last error")                          // 10번째 에러

		assertNotDone(t, ctx)
	})

	t.Run("에러 발생 빈도 변화 테스트", func(t *testing.T) { //가장 최근의 10개의 에러가 1초 이내에 발생해야 트리거되므로, 마지막 에러는 제외한 9개를 기준으로 계산. 그 주기는 1/9 = 0.111...초
		errorChan := make(chan error, 100)

		ctx, tooMuchErrFunc := context.WithCancel(context.Background())
		go cchan.TooMuchError(errCount, duration, errorChan, tooMuchErrFunc)

		for i := 0; i < 10; i++ {
			errorChan <- errors.New("error")
			time.Sleep(time.Millisecond * time.Duration(130)) // 0.13초 대기, 아슬아슬하게 quitChan이 트리거되지 않는 주기
		}
		assertNotDone(t, ctx)

		for i := 0; i < 10; i++ {

			errorChan <- errors.New("error")
			time.Sleep(time.Millisecond * time.Duration(100)) // 0.1초 대기, 아슬아슬하게 quitChan이 트리거되는 주기
		}
		assertDone(t, ctx)
	})
}

func TestTimeout(t *testing.T) {
	initDuration := time.Second * 1
	duration := time.Millisecond * time.Duration(500)

	t.Run("initDuration 동안 processedChan에 데이터가 전달되지 않으면 종료", func(t *testing.T) {
		processedChan := make(chan ProcessedSignal, 100)

		ctx, timeoutFunc := context.WithCancel(context.Background())
		go cchan.Timeout(initDuration, duration, processedChan, timeoutFunc)

		time.Sleep(initDuration + (time.Millisecond * time.Duration(200))) // 1.2초 대기
		processedChan <- ProcessedSignal{}

		assertDone(t, ctx)
	})

	t.Run("initDuration 동안 processedChan에 데이터가 전달되면 종료되지 않음", func(t *testing.T) {
		processedChan := make(chan ProcessedSignal, 100)

		ctx, timeoutFunc := context.WithCancel(context.Background())
		go cchan.Timeout(initDuration, duration, processedChan, timeoutFunc)

		time.Sleep(initDuration - (time.Millisecond * time.Duration(200))) // 0.8초 대기
		processedChan <- ProcessedSignal{}

		assertNotDone(t, ctx)
	})

	t.Run("첫 번째 데이터 전달 이후 duration 동안 processedChan에 데이터가 전달되지 않으면 종료", func(t *testing.T) {
		processedChan := make(chan ProcessedSignal, 100)

		ctx, timeoutFunc := context.WithCancel(context.Background())
		go cchan.Timeout(initDuration, duration, processedChan, timeoutFunc)

		processedChan <- ProcessedSignal{}
		time.Sleep(duration + (time.Millisecond * time.Duration(100))) // 0.6초 대기
		processedChan <- ProcessedSignal{}

		assertDone(t, ctx)
	})

	t.Run("첫 번째 데이터 전달 이후 duration 동안 processedChan에 데이터가 전달되면 종료되지 않음", func(t *testing.T) {
		processedChan := make(chan ProcessedSignal, 100)

		ctx, timeoutFunc := context.WithCancel(context.Background())
		go cchan.Timeout(initDuration, duration, processedChan, timeoutFunc)

		processedChan <- ProcessedSignal{}
		time.Sleep(duration - (time.Millisecond * time.Duration(100))) // 0.4초 대기
		processedChan <- ProcessedSignal{}

		assertNotDone(t, ctx)
	})

	t.Run("데이터 전달 빈도 변화 테스트", func(t *testing.T) {
		processedChan := make(chan ProcessedSignal, 100)

		ctx, timeoutFunc := context.WithCancel(context.Background())
		go cchan.Timeout(initDuration, duration, processedChan, timeoutFunc)

		for i := 0; i < 3; i++ {
			processedChan <- ProcessedSignal{}
			time.Sleep(duration - (time.Millisecond * time.Duration(200))) //0.3초 주기로 데이터 전달
		}
		assertNotDone(t, ctx)

		for i := 0; i < 3; i++ {
			processedChan <- ProcessedSignal{}
			time.Sleep(duration - (time.Millisecond * time.Duration(100))) //0.4초 주기로 데이터 전달
		}
		assertNotDone(t, ctx)

		time.Sleep(duration + (time.Millisecond * time.Duration(100))) // 0.6초 대기
		processedChan <- ProcessedSignal{}
		assertDone(t, ctx)
	})
}

func assertDone(t *testing.T, ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-moment():
		require.Fail(t, "Context is not done")
	}
}

func assertNotDone(t *testing.T, ctx context.Context) {
	select {
	case <-ctx.Done():
		require.Fail(t, "Context is done")
	case <-moment():
		return
	}
}

// func initProcessedChan() chan ProcessedSignal {
// 	processedChan := make(chan ProcessedSignal, 100)

// 	return processedChan
// }
