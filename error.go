package service

// sentinelError represents a constant error value that can be used for error comparisons.
// This type implements the error interface and allows for efficient error checking
// using errors.Is() without allocating new error values.
type sentinelError string

// Error implements the error interface for sentinelError.
// It returns the string representation of the sentinel error.
func (err sentinelError) Error() string { return string(err) }

// ErrUnexpectedReturn indicates that a runner stopped unexpectedly.
//
// This error is returned by Run() when a runner exits without being canceled
// by the context. Runners are expected to run indefinitely until the context
// is canceled, so any early return is considered unexpected.
//
// This can happen in several scenarios:
//   - A runner exits due to an internal error
//   - A runner completes its work and exits normally (which is still "unexpected"
//     in the context of long-running services)
//   - A runner encounters a condition that causes it to stop
//
// Note: This error is often wrapped with additional context, so always use
// errors.Is() rather than direct comparison when checking for this error.
const ErrUnexpectedReturn sentinelError = "unexpected return"

// ErrRunnerPanic indicates that a runner panicked during execution.
//
// This error is returned when a runner goroutine panics. The panic
// is recovered and converted to this error to prevent the entire
// program from crashing.
const ErrRunnerPanic sentinelError = "runner panicked"
