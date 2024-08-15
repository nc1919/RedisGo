package parser

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	cache        sync.Map
	expirations  sync.Map
	expireTicker sync.Map
)

type Command struct {
	args []string
	conn net.Conn
}

func (p *Parser) inline() (Command, error) {
	for p.current() == ' ' {
		p.advance()
	}
	cmd := Command{conn: p.conn}
	for !p.atEnd() {
		arg, err := p.consumeArg()
		if err != nil {
			return cmd, err
		}
		if arg != "" {
			cmd.args = append(cmd.args, arg)
		}
	}
	return cmd, nil
}

func Initialize() {
	expireTicker.Store("ticker", time.NewTicker(time.Second))
	go func() {
		value, ok := expireTicker.Load("ticker")
		ticker, _ := value.(*time.Ticker)
		if !ok {
			log.Println("Ticker not found")
			return
		}
		for range ticker.C {
			now := time.Now().Unix()
			expirations.Range(func(key, value interface{}) bool {
				if value.(int64) <= now {
					cache.Delete(key)
					expirations.Delete(key)
				}
				return true
			})
		}
	}()
}

func (cmd *Command) handle() bool {
	switch strings.ToUpper(cmd.args[0]) {
	case "GET":
		return cmd.get()
	case "SET":
		return cmd.set()
	case "DEL":
		return cmd.del()
	case "QUIT":
		return cmd.quit()
	default:
		log.Println("Command not supported", cmd.args[0])
		cmd.conn.Write([]uint8("-ERR unknown command '" + cmd.args[0] + "'\r\n"))
	}
	return true
}

func (cmd *Command) quit() bool {
	if len(cmd.args) != 1 {
		cmd.conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.args[0] + "' command\r\n"))
		return true
	}
	log.Println("Handle QUIT")
	cmd.conn.Write([]uint8("+OK\r\n"))
	return false
}

func (cmd *Command) del() bool {
	count := 0
	for _, k := range cmd.args[1:] {
		if _, ok := cache.LoadAndDelete(k); ok {
			count++
		}
	}
	cmd.conn.Write([]uint8(fmt.Sprintf(":%d\r\n", count)))
	return true
}

// get Fetches a key from the cache if exists.
func (cmd Command) get() bool {
	if len(cmd.args) != 2 {
		cmd.conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.args[0] + "' command\r\n"))
		return true
	}
	log.Println("Handle GET")
	val, _ := cache.Load(cmd.args[1])
	if val != nil {
		res, _ := val.(string)
		if strings.HasPrefix(res, "\"") {
			res, _ = strconv.Unquote(res)
		}
		log.Println("Response length", len(res))
		cmd.conn.Write([]uint8(fmt.Sprintf("$%d\r\n", len(res))))
		cmd.conn.Write(append([]uint8(res), []uint8("\r\n")...))
	} else {
		cmd.conn.Write([]uint8("$-1\r\n"))
	}
	return true
}

// set Stores a key and value on the cache. Optionally sets expiration on the key.
func (cmd Command) set() bool {
	if len(cmd.args) < 3 || len(cmd.args) > 6 {
		cmd.conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.args[0] + "' command\r\n"))
		return true
	}
	log.Println("Handle SET")
	log.Println("Value length", len(cmd.args[2]))
	if len(cmd.args) > 3 {
		pos := 3
		option := strings.ToUpper(cmd.args[pos])
		switch option {
		case "NX":
			log.Println("Handle NX")
			if _, ok := cache.Load(cmd.args[1]); ok {
				cmd.conn.Write([]uint8("$-1\r\n"))
				return true
			}
			pos++
		case "XX":
			log.Println("Handle XX")
			if _, ok := cache.Load(cmd.args[1]); !ok {
				cmd.conn.Write([]uint8("$-1\r\n"))
				return true
			}
			pos++
		}
		if len(cmd.args) > pos {
			if err := cmd.setExpiration(pos); err != nil {
				cmd.conn.Write([]uint8("-ERR " + err.Error() + "\r\n"))
				return true
			}
		}
	}
	cache.Store(cmd.args[1], cmd.args[2])
	cmd.conn.Write([]uint8("+OK\r\n"))
	return true
}

// setExpiration Handles expiration when passed as part of the 'set' command.
func (cmd Command) setExpiration(pos int) error {
	option := strings.ToUpper(cmd.args[pos])
	value, _ := strconv.Atoi(cmd.args[pos+1])
	var duration time.Duration
	switch option {
	case "EX":
		duration = time.Second * time.Duration(value)
	case "PX":
		duration = time.Millisecond * time.Duration(value)
	default:
		return fmt.Errorf("expiration option is not valid")
	}
	go func() {
		log.Printf("Handling '%s', sleeping for %v\n", option, duration)
		time.Sleep(duration)
		cache.Delete(cmd.args[1])
	}()
	return nil
}
