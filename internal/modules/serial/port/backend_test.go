package port

import "testing"

func TestRealBackendImplementsBackend(t *testing.T) {
	var _ Backend = RealBackend{}
}
