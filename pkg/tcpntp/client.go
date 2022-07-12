package tcpntp

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

type NTPClient struct {
	conf *Config
	conn *net.TCPConn
}

func NewNTPClient(conf *Config) (*NTPClient, error) {
	return &NTPClient{
		conf: conf,
	}, nil
}

func (nc *NTPClient) Open() error {
	var err error
	raddr, err := net.ResolveTCPAddr("tcp", nc.conf.Address)
	if err != nil {
		return fmt.Errorf("failed to resolve tcp addr [%s]: %v",
			nc.conf.Address, err)
	}
	if nc.conn, err = net.DialTCP("tcp", nil, raddr); err != nil {
		return fmt.Errorf(
			"failed to dial tcp addr [%s]: %v", nc.conf.Address, err)
	}
	return nil
}

func (nc *NTPClient) Close() error {
	return nc.conn.Close()
}

func (nc *NTPClient) Query() (*Response, error) {
	var err error
	// Allocate a message to hold the response.
	recvMsg := new(msg)

	// Allocate a message to hold the query.
	xmitMsg := new(msg)
	xmitMsg.setMode(client)
	xmitMsg.setVersion(4)
	xmitMsg.setLeap(LeapNotInSync)

	// To ensure privacy and prevent spoofing, try to use a random 64-bit
	// value for the TransmitTime. If crypto/rand couldn't generate a
	// random value, fall back to using the system clock. Keep track of
	// when the messsage was actually transmitted.
	bits := make([]byte, 8)
	_, err = rand.Read(bits)
	var xmitTime time.Time
	if err == nil {
		xmitMsg.TransmitTime = ntpTime(binary.BigEndian.Uint64(bits))
		xmitTime = time.Now()
	} else {
		xmitTime = time.Now()
		xmitMsg.TransmitTime = toNtpTime(xmitTime)
	}

	// Transmit the query.
	err = binary.Write(nc.conn, binary.BigEndian, xmitMsg)
	if err != nil {
		return nil, err
	}

	// Receive the response.
	err = binary.Read(nc.conn, binary.BigEndian, recvMsg)
	if err != nil {
		return nil, err
	}

	// Keep track of the time the response was received.
	delta := time.Since(xmitTime)
	if delta < 0 {
		// The local system may have had its clock adjusted since it
		// sent the query. In go 1.9 and later, time.Since ensures
		// that a monotonic clock is used, so delta can never be less
		// than zero. In versions before 1.9, a monotonic clock is
		// not used, so we have to check.
		return nil, errors.New("client clock ticked backwards")
	}
	recvTime := toNtpTime(xmitTime.Add(delta))

	// Check for invalid fields.
	if recvMsg.getMode() != server {
		return nil, errors.New("invalid mode in response")
	}
	if recvMsg.TransmitTime == ntpTime(0) {
		return nil, errors.New("invalid transmit time in response")
	}
	if recvMsg.OriginTime != xmitMsg.TransmitTime {
		return nil, errors.New("server response mismatch")
	}
	if recvMsg.ReceiveTime > recvMsg.TransmitTime {
		return nil, errors.New("server clock ticked backwards")
	}

	// Correct the received message's origin time using the actual
	// transmit time.
	recvMsg.OriginTime = toNtpTime(xmitTime)
	return parseTime(recvMsg, recvTime), nil
}
