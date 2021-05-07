package gourt

import (
	"fmt"
	"net"
	"errors"
	"strings"
	"time"
	"strconv"
	"regexp"
)

type Master_list struct
{
	List	[]string /* "ip:port" slice */
	Len	int
}

type Server struct
{
	Address	string			/* "ip:port" */
	Info	map[string]string	/* srv_key:srv_val map */
	Status	map[string]string	/* srv_key:srv_val map */
	Players	map[string]int		/* playername:ping map */
	UpdateTime time.Time
}

func sanitize_playername(s string) string {
	r, err := regexp.Compile("\\^[0-9]|^\"|\"$")

	if err != nil {
		return s
	}

	return r.ReplaceAllString(s, "")
}


func getstatus(con *net.Conn, srv *Server) error {
	buf := make([]byte, 4096)
	list := make([]string, 1024)

	fmt.Fprintf(*con, "\xff\xff\xff\xffgetstatus")
	size, err := (*con).Read(buf)

	if err != nil || size < 32 {
		return errors.New("Read error: " + err.Error())
	}

	t := strings.Split(string(buf), "\n")

	if len(t) < 3 {
		return errors.New("Invalid server response (1)")
	}

	list = strings.Split(t[1][1:], "\\")

	/* Length has to be even or something's wrong */
	if (len(list)%2) != 0 {
		return errors.New("Invalid server response (2)")
	}

	srv.Status = make(map[string]string)
	/* Get variables */
	for i := 0; i < len(list); i += 2 {
		srv.Status[list[i]] = list[i+1]
	}

	srv.Players = make(map[string]int)
	/* Get playernames / pings */
	for i := 2; i < len(t) - 1; i++ {
		l := strings.Split(t[i], " ")
		if len(l) != 3 {
			continue
		}
		ping, _ := strconv.Atoi(l[1])
		name := sanitize_playername(l[2])
		srv.Players[name] = ping
	}

	return nil
}
func getinfo(con *net.Conn, srv *Server) error {
	buf := make([]byte, 4096)
	list := make([]string, 1024)

	fmt.Fprintf(*con, "\xff\xff\xff\xffgetinfo")
	size, err := (*con).Read(buf)

	if err != nil || size < 32 {
		return errors.New("Read error: " + err.Error())
	}

	t := strings.Split(string(buf), "\n")

	if len(t) < 2 {
		return errors.New("Invalid server response (1)")
	}

	list = strings.Split(t[1][1:], "\\")

	/* Length has to be even or something's wrong */
	if (len(list)%2) != 0 {
		return errors.New("Invalid server response (2)")
	}

	srv.Info = make(map[string]string)

	for i := 0; i < len(list); i += 2 {
		srv.Info[list[i]] = list[i+1]
	}

	return nil
}

func QueryServer(addr string) (Server, error) {
	con, err := net.DialTimeout("udp", addr, time.Second * 5)
	var ret Server

	ret.Address = addr
	ret.UpdateTime = time.Now()

	defer con.Close()

	if err != nil {
		return ret, errors.New("Connection error: " + err.Error())
	}

	err = con.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		fmt.Printf("SetReadDeadline failed: %s\n", err.Error())
	}

	err = getstatus(&con, &ret)
	if err != nil {
		return ret, errors.New("getstatus: " + err.Error())
	}

	err = getinfo(&con, &ret)
	if err != nil {
		return ret, errors.New("getinfo: " + err.Error())
	}

	return ret, nil
}

func GetMasterList(master string) (Master_list, error) {
	con, err := net.DialTimeout("udp", master, time.Second * 5)
	buf := make([]byte, 8192)
	var ret Master_list

	defer con.Close()

	if err != nil {
		return ret, errors.New("Connection error: " + err.Error())
	}

	err = con.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		fmt.Printf("SetReadDeadline failed: %s\n", err.Error())
	}

	fmt.Fprintf(con, "\xff\xff\xff\xffgetservers 68 empty full demo\n")
	size, err := con.Read(buf)

	if err != nil || size < 32 {
		return ret, errors.New("Read error: " + err.Error())
	}

	buf = buf[len("\xff\xff\xff\xffgetserversResponse")+1:]

	ret.List = make([]string, 1024)

	/* Discard header response */
	for  i := 0; i+22 < size; i, ret.Len = i+7, ret.Len+1 {
		ret.List[ret.Len] = fmt.Sprintf("%d.%d.%d.%d:%d", buf[i], buf[i+1], buf[i+2], buf[i+3], (uint(buf[i+4]) << 8) + uint(buf[i+5]))
	}

	return ret, nil
}
