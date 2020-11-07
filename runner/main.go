package main

import (
	"fmt"
	"context"
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

func ws(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	ctx := context.Background()
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	// Read messages from socket
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			return
		}

		if string(msg) == "alpine" {
			containerConfig := &container.Config{
				Image:       string(msg),
				Cmd:         []string{"sh && sleep 1"},
				AttachStdin: true,
			}

			_, err := dockerClient.ImagePull(ctx, string(msg), types.ImagePullOptions{})
			if err != nil {
				panic(err)
			}

			response, err := dockerClient.ContainerCreate(ctx, containerConfig, nil, nil, "")
			if err != nil {
				panic(err)
			}

			dockerClient.ContainerStart(ctx, response.ID, types.ContainerStartOptions{})

			containers, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{})
			if err != nil {
				panic(err)
			}

			id, err := dockerClient.ContainerExecCreate(ctx, response.ID, types.ExecConfig{Cmd: []string{"sh"}, AttachStdout: true})
			// attachResponse, _ := dockerClient.ContainerExecAttach(ctx, id.ID, types.ExecStartCheck{Detach: false})

			// attachResponse.Conn.Write([]byte("echo 'hello world'"))
			// attachResponse.Reader.WriteTo(os.Stdout)

			dockerClient.ContainerExecStart(ctx, id.ID, types.ExecStartCheck{Detach: false})

			out, err6 := dockerClient.ContainerLogs(ctx, response.ID, types.ContainerLogsOptions{ShowStdout: true})
			if err6 != nil {
				panic(err6)
			}

			io.Copy(os.Stdout, out)

			for _, container := range containers {
				conn.WriteMessage(msgType, []byte(container.ID))
			}
		} else {
			// Print the message to the console
			fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))
			if err = conn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
		log.Printf("msg: %s", string(msg))
	}
}

func main() {
	http.HandleFunc("/echo", ws)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
