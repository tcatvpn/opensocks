package proxy

import (
	"bytes"
	"encoding/binary"
	"net"

	"github.com/net-byte/opensocks/common/enum"
)

func resp(conn net.Conn, rep byte) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	conn.Write([]byte{enum.Socks5Version, rep, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func respNoAuth(conn net.Conn) {
	/**
	  +----+--------+
	  |VER | METHOD |
	  +----+--------+
	  | 1  |   1    |
	  +----+--------+
	*/
	conn.Write([]byte{enum.Socks5Version, enum.NoAuth})
}

func respSuccess(conn net.Conn, ip net.IP, port int) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	resp := []byte{enum.Socks5Version, enum.SuccessReply, 0x00, 0x01}
	buffer := bytes.NewBuffer(resp)
	binary.Write(buffer, binary.BigEndian, ip)
	binary.Write(buffer, binary.BigEndian, uint16(port))
	conn.Write(buffer.Bytes())
}
