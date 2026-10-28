package main

import (
	"net"
	"reflect"
	"testing"
)

func TestStartupAddresses(t *testing.T) {
	ips := []net.IP{net.ParseIP("::1"), net.ParseIP("127.0.0.1"), net.ParseIP("169.254.1.2"), net.ParseIP("192.168.1.8"), net.ParseIP("10.0.0.2"), net.ParseIP("192.168.1.8"), net.ParseIP("0.0.0.0"), net.ParseIP("224.1.2.3")}
	for _, host := range []string{"", "0.0.0.0", "::"} {
		got := formatStartupAddresses(host, 10081, ips)
		want := []string{"10.0.0.2:10081", "192.168.1.8:10081"}
		if !reflect.DeepEqual(got, want) {
			t.Fatal(host, got)
		}
	}
	if got := formatStartupAddresses("0.0.0.0", 10081, nil); !reflect.DeepEqual(got, []string{"127.0.0.1:10081"}) {
		t.Fatal(got)
	}
	for _, host := range []string{"127.0.0.1", "192.168.1.8", "localhost"} {
		if got := formatStartupAddresses(host, 10081, ips); got[0] != host+":10081" {
			t.Fatal(host, got)
		}
	}
}
