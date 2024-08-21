package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"winse.com/study/grpc/server/user"
)

func main() {
	conn, err := grpc.Dial(":50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	c := user.NewUserCliClient(conn)

	r, err := c.GetUser(context.TODO(), &user.UserParam{Id: 1001})
	if err != nil {
		panic(err)
	}

	fmt.Println(r.GetId(), r.GetName())
}
