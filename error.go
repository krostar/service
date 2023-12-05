package service

type sentinelError string

func (err sentinelError) Error() string { return string(err) }

// ErrUnexpectedReturn is triggered if a runner is not expected to return but returned.
const ErrUnexpectedReturn sentinelError = "unexpected return"
