package adaptive

import (
	"errors"
	"fmt"
	"time"
)

// Ported from the Terraform provider's retry helper. Used to poll asynchronous
// Adaptive resources (sessions, etc.) until they reach a terminal state.

var (
	ErrTimeout           = errors.New("timeout occured")
	ErrMaxRetriesReached = errors.New("too many errors")
)

type RetryOption func(*retryOptions)

func Timeout(d time.Duration) RetryOption    { return func(o *retryOptions) { o.Timeout = d } }
func RetryLimit(n int) RetryOption           { return func(o *retryOptions) { o.RetryLimit = n } }
func Sleep(d time.Duration) RetryOption      { return func(o *retryOptions) { o.Sleep = d } }
func RetryChecker(f func(any, error) bool) RetryOption {
	return func(o *retryOptions) { o.Checker = f }
}
func RetryResultChecker(f func(any) bool) RetryOption {
	return func(o *retryOptions) { o.ResultChecker = f }
}

type retryOptions struct {
	Timeout       time.Duration
	RetryLimit    int
	Checker       func(any, error) bool
	ResultChecker func(any) bool
	Sleep         time.Duration
}

func newRetryOptions(opts ...RetryOption) retryOptions {
	s := retryOptions{
		Timeout:       5 * time.Minute,
		RetryLimit:    10,
		Checker:       func(any, error) bool { return true },
		ResultChecker: func(r any) bool { return r != nil },
	}
	for _, o := range opts {
		o(&s)
	}
	return s
}

func zeroVal[T any]() T { return *new(T) }

// Do runs op, retrying while the result/error checkers indicate a non-terminal
// state, up to the configured retry limit or timeout.
func Do[T any](op func() (T, error), opts ...RetryOption) (T, error) {
	o := newRetryOptions(opts...)

	var timeout <-chan time.Time
	if o.Timeout > 0 {
		timeout = time.After(o.Timeout)
	}

	tries := 0
	for {
		select {
		case <-timeout:
			return zeroVal[T](), ErrTimeout
		default:
		}

		tries++
		result, lastErr := op()

		isBadResult := o.ResultChecker(result)
		if isBadResult || lastErr != nil {
			if isBadResult || (o.Checker != nil && o.Checker(result, lastErr)) {
				if tries >= o.RetryLimit {
					return zeroVal[T](), fmt.Errorf("%w, (%d/%d). last error: %v", ErrMaxRetriesReached, tries, o.RetryLimit, lastErr)
				}
				if o.Sleep > 0 {
					time.Sleep(o.Sleep)
				}
				continue
			}
			return zeroVal[T](), lastErr
		}
		return result, nil
	}
}
