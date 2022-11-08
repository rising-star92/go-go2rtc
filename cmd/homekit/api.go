package homekit

import (
	"encoding/json"
	"fmt"
	"github.com/AlexxIT/go2rtc/cmd/app/store"
	"github.com/AlexxIT/go2rtc/cmd/streams"
	"github.com/AlexxIT/go2rtc/pkg/homekit"
	"github.com/AlexxIT/go2rtc/pkg/homekit/mdns"
	"net/http"
	"net/url"
	"strings"
)

func apiHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		items := make([]interface{}, 0)

		for name, src := range store.GetDict("streams") {
			if src := src.(string); strings.HasPrefix(src, "homekit") {
				u, err := url.Parse(src)
				if err != nil {
					continue
				}
				device := Device{
					Name:   name,
					Addr:   u.Host,
					Paired: true,
				}
				items = append(items, device)
			}
		}

		for info := range mdns.GetAll() {
			if !strings.HasSuffix(info.Name, mdns.Suffix) {
				continue
			}
			name := info.Name[:len(info.Name)-len(mdns.Suffix)]
			device := Device{
				Name: strings.ReplaceAll(name, "\\", ""),
				Addr: fmt.Sprintf("%s:%d", info.AddrV4, info.Port),
			}
			for _, field := range info.InfoFields {
				switch field[:2] {
				case "id":
					device.ID = field[3:]
				case "md":
					device.Model = field[3:]
				case "sf":
					device.Paired = field[3] == '0'
				}
			}
			items = append(items, device)
		}

		_ = json.NewEncoder(w).Encode(items)

	case "POST":
		// TODO: post params...

		id := r.URL.Query().Get("id")
		pin := r.URL.Query().Get("pin")

		conn, err := homekit.Pair(id, pin)
		if err != nil {
			// log error
			log.Error().Err(err).Caller().Send()
			// response error
			_, err = w.Write([]byte(err.Error()))
			return
		}

		name := r.URL.Query().Get("name")
		dict := store.GetDict("streams")
		dict[name] = conn.URL()
		if err = store.Set("streams", dict); err != nil {
			// log error
			log.Error().Err(err).Caller().Send()
			// response error
			_, err = w.Write([]byte(err.Error()))
		}

		streams.New(name, conn.URL())

	case "DELETE":
		src := r.URL.Query().Get("src")
		dict := store.GetDict("streams")
		for name, rawURL := range dict {
			if name != src {
				continue
			}

			conn, err := homekit.Dial(rawURL.(string))
			if err != nil {
				// log error
				log.Error().Err(err).Caller().Send()
				// response error
				_, err = w.Write([]byte(err.Error()))
				return
			}

			go func() {
				if err = conn.Handle(); err != nil {
					log.Warn().Err(err).Caller().Send()
				}
			}()

			if err = conn.ListPairings(); err != nil {
				// log error
				log.Error().Err(err).Caller().Send()
				// response error
				_, err = w.Write([]byte(err.Error()))
				return
			}

			if err = conn.DeletePairing(conn.ClientID); err != nil {
				// log error
				log.Error().Err(err).Caller().Send()
				// response error
				_, err = w.Write([]byte(err.Error()))
			}

			delete(dict, name)

			if err = store.Set("streams", dict); err != nil {
				// log error
				log.Error().Err(err).Msg("[api.homekit] store set")
				// response error
				_, err = w.Write([]byte(err.Error()))
			}

			return
		}
	}
}

type Device struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Addr   string `json:"addr"`
	Model  string `json:"model"`
	Paired bool   `json:"paired"`
	//Type    string `json:"type"`
}
