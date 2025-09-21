package app

type Server struct {
}

func NewAppServer() *Server {
	return &Server{}
}

func (s *Server) Serve() error {
	return nil
}
