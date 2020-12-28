package controllers

import (
	"fmt"
	"net"
	lbv1 "smartLB/api/v1"
	"smartLB/controllers/ipvs"
	"strconv"

	"k8s.io/apimachinery/pkg/util/sets"
)

type Loadbalancer interface {
	Create(lbv1.SmartLBStatus) error
	Delete(lbv1.SmartLBStatus) error
}

var _ Loadbalancer = &IpvsLB{}

type IpvsLB struct {
	ipvs          ipvs.Interface
	netlinkHandle ipvs.NetLinkHandle
	netDevice     string
	ipvsScheduler string
	weight        int
}

func (lb *IpvsLB) Create(status lbv1.SmartLBStatus) error {

	// curEndpoints represents IPVS destinations listed from current system.
	curEndpoints := sets.NewString()
	// newEndpoints represents Endpoints watched from API Server.
	newEndpoints := sets.NewString()

	vsInfo, rsInfo := lb.convert(status)

	for _, vs := range vsInfo {
		appliedVirtualServer, _ := lb.ipvs.GetVirtualServer(&vs)
		if appliedVirtualServer == nil || !appliedVirtualServer.Equal(&vs) {
			if appliedVirtualServer == nil {
				// IPVS service is not found, create a new service
				log.Info(fmt.Sprintf("Adding new service %s:%d/%s", vs.Address, vs.Port, vs.Protocol))
				if err := lb.ipvs.AddVirtualServer(&vs); err != nil {
					log.Error(err, "Failed to add IPVS service")
					return err
				}
				// Bind vip to interface
				if _, err := lb.netlinkHandle.EnsureAddressBind(vs.Address.String(), lb.netDevice); err != nil {
					log.Error(err, "Failed to bind service address")
					return err
				}
			} else {
				// IPVS service was changed, update the existing one
				// During updates, service VIP will not go down
				log.Info("IPVS service was changed")
				if err := lb.ipvs.UpdateVirtualServer(&vs); err != nil {
					log.Error(err, "Failed to update IPVS service")
					return err
				}
			}
		}

		appliedVirtualServer, _ = lb.ipvs.GetVirtualServer(&vs)
		curDests, err := lb.ipvs.GetRealServers(appliedVirtualServer)
		if err != nil {
			log.Error(err, "Failed to list IPVS destinations")
			return err
		}
		for _, des := range curDests {
			curEndpoints.Insert(des.String())
		}
		for _, epInfo := range lb.filter(vs, rsInfo) {
			newEndpoints.Insert(epInfo.String())
		}
		// Create new endpoints
		for _, ep := range newEndpoints.Difference(curEndpoints).UnsortedList() {
			ip, port, err := net.SplitHostPort(ep)
			if err != nil {
				log.Error(err, "Failed to parse endpoint ip/port")
				continue
			}
			portNum, err := strconv.Atoi(port)
			if err != nil {
				log.Error(err, "Failed to parse endpoint port")
				continue
			}

			newDest := &ipvs.RealServer{
				Address: net.ParseIP(ip),
				Port:    uint16(portNum),
				Weight:  lb.weight,
			}
			if err := lb.ipvs.AddRealServer(appliedVirtualServer, newDest); err != nil {
				log.Error(err, "Failed to add destination")
				continue
			}
		}
		// Delete old endpoints
		for _, ep := range curEndpoints.Difference(newEndpoints).UnsortedList() {
			ip, port, err := net.SplitHostPort(ep)
			if err != nil {
				log.Error(err, "Failed to parse endpoint ip/port")
				continue
			}
			portNum, err := strconv.Atoi(port)
			if err != nil {
				log.Error(err, "Failed to parse endpoint port")
				continue
			}
			delDest := &ipvs.RealServer{
				Address: net.ParseIP(ip),
				Port:    uint16(portNum),
			}
			if err := lb.ipvs.DeleteRealServer(appliedVirtualServer, delDest); err != nil {
				log.Error(err, "Failed to delete destination")
				continue
			}
		}
	}

	return nil
}

func (lb *IpvsLB) Delete(status lbv1.SmartLBStatus) error {

	vsInfo, _ := lb.convert(status)
	log.Info("Virtual Server will be deleted ", "numbers: ", len(vsInfo))

	for _, vs := range vsInfo {
		appliedVirtualServer, _ := lb.ipvs.GetVirtualServer(&vs)
		if appliedVirtualServer == nil {
			log.Info("Virtual Server not found, skipping delete :" + vs.String())
			continue
		}
		if err := lb.ipvs.DeleteVirtualServer(&vs); err != nil {
			log.Error(err, "Failed to delete Virtual Server :"+vs.String())
		}
		addr := vs.Address.String()
		if err := lb.netlinkHandle.UnbindAddress(addr, lb.netDevice); err != nil {
			log.Error(err, "Failed to unbind service addr :"+addr)
		}
	}

	return nil
}

// Convert SmartLB status to VS and RS slice
func (lb *IpvsLB) convert(status lbv1.SmartLBStatus) ([]ipvs.VirtualServer, []ipvs.RealServer) {

	vsInfo := make([]ipvs.VirtualServer, 0)
	rsInfo := make([]ipvs.RealServer, 0)

	for _, port := range status.Ports {
		vs := ipvs.VirtualServer{
			Address:   net.ParseIP(status.ExternalIP),
			Port:      uint16(port.Port),
			Scheduler: lb.ipvsScheduler,
			Protocol:  port.Protocol,
		}
		vsInfo = append(vsInfo, vs)
		for _, node := range status.Nodes {
			rs := ipvs.RealServer{
				Address: net.ParseIP(node),
				Port:    uint16(port.Port),
				Weight:  lb.weight,
			}
			rsInfo = append(rsInfo, rs)
		}
	}

	return vsInfo, rsInfo
}

func (lb *IpvsLB) filter(virtualServer ipvs.VirtualServer, realServer []ipvs.RealServer) []ipvs.RealServer {

	newRealServer := make([]ipvs.RealServer, 0)
	for _, rs := range realServer {
		if virtualServer.Port == rs.Port {
			newRealServer = append(newRealServer, rs)
		}
	}

	return newRealServer
}
