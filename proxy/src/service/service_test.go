package service

import (
	"github.com/coreos/go-iptables/iptables"
	"testing"
)

func TestInit(t *testing.T) {
	sm, err := New()
	if err != nil {
		t.Error(err)
	}
	if err = sm.Init(); err != nil {
		t.Error(err)
	}
}

func TestServiceCreate(t *testing.T) {
	sm, err := New()
	if err != nil {
		t.Error(err)
	}
	var eps = make([]EndPoint, 1)
	eps[0] = EndPoint{Name: "KUBE-SEP", Ip: "127.0.0.1", Port: "23333"}
	if err = sm.CreateService("KUBE-SVC", "10.96.1.1/32", "32222"); err != nil {
		t.Error(err)
	}
	if err = sm.CreateEndpoints("KUBE-SVC", eps); err != nil {
		t.Error(err)
	}
}

func TestServiceDelete(t *testing.T) {
	sm, err := New()
	if err != nil {
		t.Error(err)
	}
	var eps = make([]EndPoint, 1)
	eps[0] = EndPoint{Name: "KUBE-SEP", Ip: "127.0.0.1", Port: "23333"}
	if err = sm.DeleteEndPoints("KUBE-SVC", eps); err != nil {
		t.Error(err)
	}
	if err = sm.DeleteService("KUBE-SVC", "10.96.1.1/32", "32222"); err != nil {
		t.Error(err)
	}
}

func Test2ServiceCreate(t *testing.T) {
	sm, err := New()
	if err != nil {
		t.Error(err)
	}
	var eps = make([]EndPoint, 2)
	eps[0] = EndPoint{Name: "KUBE-SEP1", Ip: "10.44.0.2", Port: "80"}
	eps[1] = EndPoint{Name: "KUBE-SEP2", Ip: "10.44.0.3", Port: "80"}
	if err = sm.CreateService("KUBE-SVC", "10.96.0.1/32", "32222"); err != nil {
		t.Error(err)
	}
	if err = sm.CreateEndpoints("KUBE-SVC", eps); err != nil {
		t.Error(err)
	}
}

func TestReplicaServiceCreate(t *testing.T) {
	sm, err := New()
	if err != nil {
		t.Error(err)
	}
	var eps = make([]EndPoint, 3)
	eps[0] = EndPoint{Name: "KUBE-SEP1", Ip: "127.0.0.1", Port: "23333"}
	eps[1] = EndPoint{Name: "KUBE-SEP2", Ip: "127.0.0.1", Port: "23334"}
	eps[2] = EndPoint{Name: "KUBE-SEP3", Ip: "127.0.0.1", Port: "23335"}
	if err = sm.CreateService("KUBE-SVC", "10.96.1.1/32", "32222"); err != nil {
		t.Error(err)
	}
	if err = sm.CreateEndpoints("KUBE-SVC", eps); err != nil {
		t.Error(err)
	}
}

func TestReplicaServiceDelete(t *testing.T) {
	sm, err := New()
	if err != nil {
		t.Error(err)
	}
	var eps = make([]EndPoint, 3)
	eps[0] = EndPoint{Name: "KUBE-SEP1", Ip: "127.0.0.1", Port: "23333"}
	eps[1] = EndPoint{Name: "KUBE-SEP2", Ip: "127.0.0.1", Port: "23334"}
	eps[2] = EndPoint{Name: "KUBE-SEP3", Ip: "127.0.0.1", Port: "23335"}
	if err = sm.DeleteEndPoints("KUBE-SVC", eps); err != nil {
		t.Error(err)
	}
	if err = sm.DeleteService("KUBE-SVC", "10.96.1.1/32", "32222"); err != nil {
		t.Error(err)
	}
}

func TestClear(t *testing.T) {
	tab, err := iptables.New()
	if err != nil {
		t.Error(err)
	}
	_ = tab.ClearChain("nat", "KUBE-SERVICES")
}

func TestListChains(t *testing.T) {
	tab, err := iptables.New()
	if err != nil {
		t.Error(err)
	}
	str, _ := tab.ListChains("nat")
	t.Log(str)
}
