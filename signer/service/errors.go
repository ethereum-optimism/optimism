package service

type TransactionUnmarshalError struct{}

func (e *TransactionUnmarshalError) Error() string  { return "tx param failed to umarshal" }
func (e *TransactionUnmarshalError) ErrorCode() int { return -32010 }

type InvalidDigestError struct{}

func (e *InvalidDigestError) Error() string  { return "digest param does not match transaction hash" }
func (e *InvalidDigestError) ErrorCode() int { return -32011 }
