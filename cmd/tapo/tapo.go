package tapo

import (
	"github.com/AlexxIT/go2rtc/cmd/streams"
	"github.com/AlexxIT/go2rtc/pkg/streamer"
	"github.com/AlexxIT/go2rtc/pkg/tapo"
)

func Init() {
	streams.HandleFunc("tapo", handle)
}

func handle(url string) (streamer.Producer, error) {
	conn := tapo.NewClient(url)
	if err := conn.Dial(); err != nil {
		return nil, err
	}
	if err := conn.Play(); err != nil {
		return nil, err
	}
	if err := conn.Handle(); err != nil {
		return nil, err
	}
	return conn, nil
}
