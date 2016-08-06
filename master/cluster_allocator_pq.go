package master

type pqSlice struct {
	Slice          []interface{}
	LessComparator func(i, j interface{}) bool
}

func (s pqSlice) Len() int {
	return len(s.Slice)
}

func (s pqSlice) Less(left, right int) bool {
	return s.LessComparator(s.Slice[left], s.Slice[right])
}

func (s *pqSlice) Swap(i, j int) {
	s.Slice[i], s.Slice[j] = s.Slice[j], s.Slice[i]
}

func (s *pqSlice) Push(i interface{}) {
	s.Slice = append(s.Slice, i)
}

func (s *pqSlice) Pop() interface{} {
	ret := s.Slice[len(s.Slice)-1]
	s.Slice = s.Slice[0 : len(s.Slice)-1]
	return ret
}
