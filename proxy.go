package main

import (
	"bytes"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"sync"
	"time"
)

type Proxy struct {
	ServerName string
	Source     string `mapstructure:"src"`
	Target     string `mapstructure:"dst"`
	LastLogin  time.Time
}

type Session struct {
	LocalConn  net.Conn
	RemoteConn net.Conn
	XorKey     []byte
}

func startProxy(wg *sync.WaitGroup, proxy Proxy) {
	defer wg.Done()

	listenerAddr, err := net.ResolveTCPAddr("tcp", proxy.Source)
	if err != nil {
		log.Error().Err(err).Msg("Cannot resolve listener address")
		return
	}

	listener, err := net.ListenTCP("tcp", listenerAddr)
	if err != nil {
		log.Error().Err(err).Msgf("Cannot listen on %s", listenerAddr)
		return
	}

	defer listener.Close()

	proxy.LastLogin = time.Now()

	log.Info().Msgf("Listening on %s", listenerAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error().Err(err).Msg("Cannot accept connection")
			continue
		}

		go openServerConnection(conn, &proxy)
	}
}

func openServerConnection(sourceConn net.Conn, proxy *Proxy) {
	targetAddr, err := net.ResolveTCPAddr("tcp", proxy.Target)
	if err != nil {
		log.Error().Err(err).Msg("Cannot resolve source address")
		return
	}

	targetConn, err := net.DialTCP("tcp", nil, targetAddr)
	if err != nil {
		log.Error().Err(err).Msgf("Could not connect to target address: %s", targetAddr)
		return
	}

	log.Info().Msgf("New proxy started: %s -> %s", sourceConn.RemoteAddr(), proxy.ServerName)

	defer targetConn.Close()

	session := Session{
		LocalConn:  sourceConn,
		RemoteConn: targetConn,
	}
	go ioCopy(targetConn, sourceConn, proxy, &session)
	ioCopy(sourceConn, targetConn, proxy, &session)
}

func ioCopy(sourceConn net.Conn, targetConn net.Conn, proxy *Proxy, session *Session) {
	for {
		buf := make([]byte, 1024*1024)
		n, err := sourceConn.Read(buf[:cap(buf)])
		if err != nil {
			if err != io.EOF {
				log.Error().Err(err).Msg("Could not read from source connection")
			}
			return
		}

		if n <= 0 {
			continue
		}

		// Save RCON command to InfluxDB if it's a request
		cmd, args := readPacket(buf[:n], session)
		if sourceConn == session.LocalConn && cmd == "LOGIN" {
			delayLoginPackets(&proxy.LastLogin)
		}

		nW, err := targetConn.Write(buf[:n])
		if err != nil {
			log.Error().Err(err).Msg("Could not write to target connection")
			return
		}

		if n != nW {
			log.Error().Msgf("Could not write to target connection (expected %d bytes, got %d)", n, nW)
			return
		}

		if e := log.Debug(); e.Enabled() {
			e.Msgf("%s >>> %s (%d bytes)", sourceConn.RemoteAddr(), targetConn.RemoteAddr(), n)
		}

		// Save RCON command to InfluxDB if it's an RCON request
		if sourceConn == session.LocalConn {
			if e := log.Trace(); e.Enabled() {
				e.Msgf("Packet received: %s %s", cmd, args)
			}

			Repository.WriteRCONPoint(proxy.ServerName, cmd, args, n)
		}
	}
}

func delayLoginPackets(lastLogin *time.Time) {
	if time.Since(*lastLogin) > 100*time.Millisecond {
		*lastLogin = time.Now()
		return
	}

	d := (25 * time.Millisecond) + (100*time.Millisecond - time.Since(*lastLogin))
	log.Info().Msgf("Delayed login by %s (time since %s)", d, time.Since(*lastLogin))
	time.Sleep(d)
}

func xor(a []byte, key []byte) []byte {
	buf := make([]byte, len(a))

	for i := range a {
		buf[i] = a[i] ^ key[i%len(key)]
	}

	return buf
}

func readPacket(buf []byte, session *Session) (string, string) {
	if session.XorKey == nil {
		session.XorKey = buf
		if e := log.Trace(); e.Enabled() {
			e.Msgf("XOR key received: %b", buf)
		}

		return "", ""
	}

	message := xor(buf, (*session).XorKey)
	cmd := bytes.ToUpper(bytes.TrimSpace(bytes.Split(message, []byte(" "))[0]))
	args := bytes.TrimSpace(bytes.TrimPrefix(message, bytes.Split(message, []byte(" "))[0]))

	return string(cmd), string(args)
}
