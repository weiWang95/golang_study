package main

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"winse.com/study/grpc/server/user"
)

type Server struct {
	user.UnimplementedUserCliServer
}

func (s *Server) GetUser(ctx context.Context, param *user.UserParam) (*user.UserResp, error) {
	return &user.UserResp{Id: 1001, Name: "test name"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer()
	user.RegisterUserCliServer(s, &Server{})
	if err := s.Serve(lis); err != nil {
		panic(err)
	}
}
