package collector

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/xperimental/steam-exporter/internal/config"
	"io"
	"net"
	"time"
)

var (
	errNoServers           = errors.New("no servers configured")
	errTimeout             = errors.New("server timed out")
	errResponseWrongHeader = errors.New("incorrect response header")

	varLabels    = []string{"address"}
	descServerUp = prometheus.NewDesc(
		"steam_server_up",
		"Set to 1 if the server is reachable.",
		varLabels, nil)
	descServerPing = prometheus.NewDesc(
		"steam_server_response_time_seconds",
		"Shows the response time of the server in seconds.",
		varLabels, nil)
	descPlayers = prometheus.NewDesc(
		"steam_server_players_total",
		"Shows current number of players on the server.",
		varLabels, nil)
	descMaxPlayers = prometheus.NewDesc(
		"steam_server_max_players_total",
		"Shows current number of players on the server.",
		varLabels, nil)
	descBots = prometheus.NewDesc(
		"steam_server_bots_total",
		"Shows current number of players on the server.",
		varLabels, nil)

	serverQuery = []byte{
		0xFF, 0xFF, 0xFF, 0xFF, 0x54, 0x53, 0x6F, 0x75,
		0x72, 0x63, 0x65, 0x20, 0x45, 0x6E, 0x67, 0x69,
		0x6E, 0x65, 0x20, 0x51, 0x75, 0x65, 0x72, 0x79,
		0x00,
	}
	responseHeader = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x49}
)

type Collector struct {
	log     logrus.FieldLogger
	servers []config.Server
	timeout time.Duration
}

func New(log logrus.FieldLogger, servers []config.Server, timeout time.Duration) (*Collector, error) {
	if len(servers) == 0 {
		return nil, errNoServers
	}

	return &Collector{
		log:     log,
		servers: servers,
		timeout: timeout,
	}, nil
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descServerUp
	ch <- descServerPing
	ch <- descPlayers
	ch <- descMaxPlayers
	ch <- descBots
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	for _, s := range c.servers {
		data, err := c.pingServer(s.Address)
		if err != nil {
			c.log.Errorf("Can not ping %q: %s", s.Address, err)
			ch <- prometheus.MustNewConstMetric(descServerUp, prometheus.GaugeValue, 0, s.Address)
			continue
		}
		c.log.Debugf("Data for %q: %#v", s.Address, data)

		for _, v := range []struct {
			desc  *prometheus.Desc
			value float64
		}{
			{
				desc:  descServerUp,
				value: 1,
			},
			{
				desc:  descServerPing,
				value: data.Ping.Seconds(),
			},
			{
				desc:  descPlayers,
				value: float64(data.Players),
			},
			{
				desc:  descMaxPlayers,
				value: float64(data.MaxPlayers),
			},
			{
				desc:  descBots,
				value: float64(data.Bots),
			},
		} {
			ch <- prometheus.MustNewConstMetric(v.desc, prometheus.GaugeValue, v.value, s.Address)
		}
	}
}

type serverData struct {
	Ping        time.Duration
	Protocol    uint8
	Name        string
	Map         string
	Folder      string
	Game        string
	ID          uint16
	Players     uint8
	MaxPlayers  uint8
	Bots        uint8
	ServerType  uint8
	Environment uint8
	Visibility  uint8
	VAC         uint8
}

type serverResponse struct {
	Data serverData
	Err  error
}

func (c *Collector) pingServer(address string) (serverData, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return serverData{}, fmt.Errorf("can not resolve %q: %w", address, err)
	}

	udp, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return serverData{}, fmt.Errorf("can not create UDP socket: %w", err)
	}
	defer udp.Close()

	start := time.Now()
	if _, err := udp.Write(serverQuery); err != nil {
		return serverData{}, fmt.Errorf("can not write datagram: %w", err)
	}

	select {
	case res := <-c.waitForResponse(start, udp):
		return res.Data, res.Err
	case <-time.After(c.timeout):
		return serverData{}, errTimeout
	}
}

func (c *Collector) waitForResponse(start time.Time, udp *net.UDPConn) <-chan serverResponse {
	ch := make(chan serverResponse, 1)
	go func() {
		defer close(ch)
		buf := make([]byte, 1400)
		count, addr, err := udp.ReadFromUDP(buf)
		if err != nil {
			ch <- serverResponse{
				Err: fmt.Errorf("error reading from socket: %w", err),
			}
			return
		}

		ping := time.Since(start)
		c.log.Debugf("Received %d bytes from %q", count, addr)
		buf = buf[:count]

		data, err := parseResponse(buf)
		if err != nil {
			ch <- serverResponse{
				Err: fmt.Errorf("can not parse response: %w", err),
			}
			return
		}
		data.Ping = ping

		ch <- serverResponse{
			Data: data,
		}
	}()
	return ch
}

func parseResponse(buf []byte) (serverData, error) {
	if len(buf) < 19 {
		return serverData{}, fmt.Errorf("response too short %d < 19 byte", len(buf))
	}

	if bytes.Compare(buf[:5], responseHeader) != 0 {
		return serverData{}, errResponseWrongHeader
	}

	result := serverData{
		Protocol: buf[5],
	}
	reader := bytes.NewReader(buf[6:])

	var err error
	result.Name, err = readString(reader)
	if err != nil {
		return serverData{}, fmt.Errorf("can not read name: %w", err)
	}

	result.Map, err = readString(reader)
	if err != nil {
		return serverData{}, fmt.Errorf("can not read map: %w", err)
	}

	result.Folder, err = readString(reader)
	if err != nil {
		return serverData{}, fmt.Errorf("can not read folder: %w", err)
	}

	result.Game, err = readString(reader)
	if err != nil {
		return serverData{}, fmt.Errorf("can not read game: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &result.ID); err != nil {
		return serverData{}, fmt.Errorf("can not read ID: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &result.Players); err != nil {
		return serverData{}, fmt.Errorf("can not read players: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &result.MaxPlayers); err != nil {
		return serverData{}, fmt.Errorf("can not read max players: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.Bots); err != nil {
		return serverData{}, fmt.Errorf("can not read bots: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.ServerType); err != nil {
		return serverData{}, fmt.Errorf("can not read server type: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.Environment); err != nil {
		return serverData{}, fmt.Errorf("can not read environment: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.Visibility); err != nil {
		return serverData{}, fmt.Errorf("can not read visibility: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &result.VAC); err != nil {
		return serverData{}, fmt.Errorf("can not read vac: %w", err)
	}

	return result, nil
}

func readString(reader io.ByteReader) (string, error) {
	buf := &bytes.Buffer{}
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}

		if b == 0 {
			return buf.String(), nil
		}

		buf.WriteByte(b)
	}
}
