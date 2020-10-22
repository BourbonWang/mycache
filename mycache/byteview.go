package mycache

type ByteView struct {
	b []byte
}

func (this ByteView) Len() int {
	return len(this.b)
}

func (this ByteView) ByteSlice() []byte {
	return cloneBytes(this.b)
}

func cloneBytes(b []byte) []byte {
	slice := make([]byte,len(b))
	copy(slice,b)
	return slice
}

func (this ByteView) String() string {
	return string(this.b)
}