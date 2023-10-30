package server

import (
	"fmt"
	"github.com/tidwall/redcon"
	"hades/settings"
	"log"
)

type Handler struct {
	server *redcon.Server
}

func MakeHandler() *Handler {
	addr := fmt.Sprintf("%s:%d", settings.Conf.Bind, settings.Conf.Port)
	h:=&Handler{}
	return &Handler{
		server: redcon.NewServer(addr,execClientCommand, h.accept, h.Close)
	}
}

func (h *Handler) Handle() {
	log.Println("hades server running, ready to accept connections.")
	if err := h.server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (h *Handler) Close(conn redcon.Conn, err error) error {
	return h.server.Close()
}

func (h *Handler) accept(conn redcon.Conn) bool {
	log.Printf("client %s connected", conn.RemoteAddr())
	return true
}

func execClientCommand(conn redcon.Conn, cmd redcon.Command) {
	command := string(cmd.Args[0])
	switch command {
	case "quit":
		_ = conn.Close()
	case "ping":
		conn.WriteString("PONG")
	default:
		conn.WriteError("Err unsupported command: '" + command + "'")
	}
}