package workspace

type Session struct {
	current string
}

func NewSession() *Session {
	return &Session{}
}

func (s *Session) Current() string {
	return s.current
}

func (s *Session) Use(name string) {
	s.current = name
}
