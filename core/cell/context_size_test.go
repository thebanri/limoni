package cell

import "testing"
import "unsafe"

func TestContextStaysUnder128Bytes(t *testing.T) {
	if n := unsafe.Sizeof(Context{}); n > 128 {
		t.Errorf("Context is %d bytes; above 128 a closure captures it by reference, "+
			"which moves it to the heap and costs an allocation per widget per frame", n)
	}
}
