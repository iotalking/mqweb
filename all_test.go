package mqweb

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/iotalking/mqtt-broker/utils"
)

func TestHttp(t *testing.T) {
	http.HandleFunc("/api/v1/helloworld", func(w http.ResponseWriter, r *http.Request) {
		t.Log("helloworld")
	})
	go func() {
		http.ListenAndServe(":9090", nil)
	}()
	time.Sleep(500)
	_, err := http.Get("http://localhost:9090/api/v1/helloworld")
	if err != nil {
		t.FailNow()
	}
}

func BenchmarkHttp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := http.Get("http://localhost:9090/api/v1/helloworld")
		if err != nil {
			b.FailNow()
		}

	}

}

var once sync.Once

func TestServer(t *testing.T) {
	gateway := "localhost:1883"

	once.Do(func() {
		s := NewServer(utils.NewId())
		err := s.DialGeteWay(gateway)
		if err != nil {
			fmt.Println(err.Error())
			t.FailNow()
			return
		}
		err = s.Serv("/api/v1/helloworld", func(id string, params []byte) {
			s.Callback(id, []byte("helloworld"))
		})
		if err != nil {
			fmt.Println(err.Error())
			t.FailNow()
			return
		}
	})

	var b time.Time
	c := NewClient()
	err := c.DialGateWay(gateway)
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
}

func BenchmarkServer(b *testing.B) {
	b.StopTimer()
	var err error
	gateway := "localhost:1883"
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
}
