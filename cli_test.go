package main

import (
	"context"
	"log"
	"testing"
	"tr/com/havelsan/hloader/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestLogOff(t *testing.T) {
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())
	cc, err := grpc.Dial("10.10.11.40:8080", opts)
	if err != nil {
		panic(err)
	}
	defer cc.Close()
	log.Println("Start")
	cli := api.NewLoaderClient(cc)

	req := &api.PowerCtlOrder{Order: api.PowerStatusCommand_Logoff}

	resp, err := cli.PowerCtl(context.Background(), req)
	if err != nil {
		panic(err)
	}
	t.Fatal("Resp:", resp.Message)

}
