package websocket

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func Start(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Read messages from socket
	go run(conn)
}

func run(conn *websocket.Conn) {
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			return
		}

		switch string(msg) {
		case
			"alpine",
			"ubuntu":
			for _, container := range runContainer(string(msg)) {
				conn.WriteMessage(msgType, []byte(container.ID))
			}
		default:
			// Print the message to the console
			fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))
			if err = conn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}

		log.Printf("msg: %s", string(msg))
	}
}

func runContainer(msg string) []types.Container {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containerConfig := &container.Config{
		Image:       string(msg),
		Cmd:         []string{"sh && sleep 1"},
		AttachStdin: true,
	}

	_, err = cli.ImagePull(ctx, string(msg), types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	response, err := cli.ContainerCreate(ctx, containerConfig, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	cli.ContainerStart(ctx, response.ID, types.ContainerStartOptions{})

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	id, err := cli.ContainerExecCreate(ctx, response.ID, types.ExecConfig{Cmd: []string{"sh"}, AttachStdout: true})
	// attachResponse, _ := cli.ContainerExecAttach(ctx, id.ID, types.ExecStartCheck{Detach: false})

	// attachResponse.Conn.Write([]byte("echo 'hello world'"))
	// attachResponse.Reader.WriteTo(os.Stdout)

	cli.ContainerExecStart(ctx, id.ID, types.ExecStartCheck{Detach: false})

	out, err := cli.ContainerLogs(ctx, response.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, out)

	return containers
}
