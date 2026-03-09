package audit

import "testing"

func TestLocalAppender_Init(t *testing.T) {
	appender := NewLocalAppender("test.log")
	if appender == nil {
		t.Fatalf("expected non-nil appender")
	}
}
