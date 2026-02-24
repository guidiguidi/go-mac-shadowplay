package buffer

import "testing"

func TestRingBufferBasic(t *testing.T) {
	rb := New(4)
	if !rb.IsEmpty() || rb.Len() != 0 {
		t.Fatal("init fail")
	}

	rb.Put(Frame{Data: []byte{1}})
	if rb.Len() != 1 || rb.IsEmpty() {
		t.Fatal("put fail")
	}

	f, ok := rb.Get()
	if !ok || len(f.Data) != 1 || f.Data[0] != 1 || !rb.IsEmpty() {
		t.Fatal("get fail")
	}
}

func TestRingBufferOverflow(t *testing.T) {
	rb := New(3)
	rb.Put(Frame{Data: []byte{1}})
	rb.Put(Frame{Data: []byte{2}})
	rb.Put(Frame{Data: []byte{3}})

	if !rb.IsFull() || rb.Len() != 3 {
		t.Fatal("full fail")
	}
	if rb.Put(Frame{Data: []byte{4}}) {
		t.Fatal("overflow put")
	}

	f, _ := rb.Get()
	if len(f.Data) != 1 || f.Data[0] != 1 {
		t.Fatal("order fail")
	}

	rb.Put(Frame{Data: []byte{4}})
	f2, _ := rb.Get()
	if len(f2.Data) != 1 || f2.Data[0] != 2 {
		t.Fatal("overwrite fail")
	}
}

func BenchmarkPutGet(b *testing.B) {
	rb := New(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Put(Frame{Data: []byte{byte(i)}})
		rb.Get()
	}
}

