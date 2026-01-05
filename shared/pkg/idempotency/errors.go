package idempotency

import "errors"

var (
	// ErrKeyRequired indicates that an idempotency key is required but was not provided
	ErrKeyRequired = errors.New("idempotency key is required for this operation")

	// ErrKeyInvalid indicates that the idempotency key format is invalid
	ErrKeyInvalid = errors.New("invalid idempotency key format")

	// ErrKeyTooLong indicates that the idempotency key exceeds the maximum length
	ErrKeyTooLong = errors.New("idempotency key exceeds maximum length of 255 characters")

	// ErrParameterMismatch indicates that request parameters differ from the original request
	ErrParameterMismatch = errors.New("request parameters differ from original request with this idempotency key")

	// ErrConcurrentRequest indicates that another request with the same key is currently being processed
	ErrConcurrentRequest = errors.New("a request with this idempotency key is currently being processed")

	// ErrStorageFailure indicates that the idempotency storage is unavailable
	ErrStorageFailure = errors.New("idempotency storage is temporarily unavailable")

	// ErrNotFound indicates that an idempotency key was not found
	ErrNotFound = errors.New("idempotency key not found")

	// ErrLockAcquisitionFailed indicates that acquiring a lock failed
	ErrLockAcquisitionFailed = errors.New("failed to acquire lock for idempotency key")

	// ErrMessageAlreadyProcessed indicates that a message has already been processed
	ErrMessageAlreadyProcessed = errors.New("message has already been processed")
)
