package impl

import "testing"

func TestCheckPort(t *testing.T) {
	port := []string{"80", "0", "200", "3000", "45678", "70000", "test", " 10", "bb "}
	for _, p := range port {
		if !CheckPort(p) {
			t.Error(p, "false")
		}
	}
}

func TestCheckIp(t *testing.T) {
	port := []string{"10.10.1.2555", "2555.10.10.1", "20.20", "aa.bb", "192.168.10.10", " 8.8.8.8", "10.13.88.10"}
	for _, p := range port {
		if !CheckIp(p) {
			t.Error(p, "false")
		}
	}
}

func TestCheckDuplicate(t *testing.T) {
	list := []string{"a", "a", "b", "b"}
	list1 := []string{"a", "b", "c", "d"}
	if !CheckNotDuplicate(list) {
		t.Error(list, "false")
	}
	if !CheckNotDuplicate(list1) {
		t.Error(list1, "false")
	}
}
