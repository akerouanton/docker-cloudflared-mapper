package sliceutil

func Map[S ~[]In, In, Out any](s S, fn func(In) Out) []Out {
	res := make([]Out, len(s))
	for i, v := range s {
		res[i] = fn(v)
	}
	return res
}
