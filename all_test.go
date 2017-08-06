package mqweb

import (
	"fmt"
	"testing"
	"time"

	"github.com/iotalking/mqtt-broker/utils"
)

func TestServer(t *testing.T) {
	gateway := "localhost:1883"
	s := NewServer(utils.NewId())
	err := s.DialGeteWay(gateway)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
		return
	}

	go func() {
		var b time.Time
		err := s.Serv("/api/v1/helloworld", func(id string, params []byte) {
			s.Callback(id, []byte("helloworld"))
		})
		c := NewClient()
		err = c.DialGateWay(gateway)
		if err != nil {
			fmt.Println(err.Error())
			t.FailNow()
			return
		}
		defer c.Close()
		b = time.Now()
		result, err := c.Call("/api/v1/helloworld", []byte("helloworld"))
		usetime := time.Since(b).String()
		t.Logf("usetime:%s", usetime)
		if err != nil {

			fmt.Println(err.Error())
			t.FailNow()
			return
		}
		if string(result) != "helloworld" {
			t.FailNow()
		}
		s.Close()
	}()
	s.Listen()
}

func BenchmarkServer(b *testing.B) {
	b.StopTimer()
	gateway := "localhost:1883"
	s := NewServer(utils.NewId())
	err := s.DialGeteWay(gateway)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	go func() {
		err := s.Serv("/api/v1/helloworld", func(id string, params []byte) {
			s.Callback(id, []byte("helloworld"))
		})
		c := NewClient()
		err = c.DialGateWay(gateway)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer c.Close()
		b.StartTimer()
		for i := 1; i < b.N; i++ {
			_, err := c.Call("/api/v1/helloworld", []byte("helloworld"))
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
		b.StopTimer()
		s.Close()
	}()
	s.Listen()
}
