package ipam

import (
	"fmt"
	"sync"
)

var VipPool = SafeVipPool{address: map[string]string{}}

// SafeVipPool is safe to use concurrently.
type SafeVipPool struct {
	mu      sync.Mutex
	address map[string]string
}

// Add addresses
func (c *SafeVipPool) Add(ips interface{}) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	switch v := ips.(type) {
	case string:
		c.address[v] = ""
	case []string:
		for _, ip := range v {
			c.address[ip] = ""
		}
	default:
		return fmt.Errorf("only string or string slice parameter is supported")
	}

	return nil
}

// Acquire address
func (c *SafeVipPool) Acquire(ip string, owner string) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	curOwner := c.address[ip]
	if curOwner == "" {
		c.address[ip] = owner
	} else if curOwner != owner {
		return fmt.Errorf("ip address %s already in used by %s", ip, curOwner)
	}

	return nil
}

// Release address
func (c *SafeVipPool) Release(ip string, owner string) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.address[ip] == owner {
		c.address[ip] = ""
	}

	return nil
}

// Clear all owners
func (c *SafeVipPool) Clear() {

	c.mu.Lock()
	defer c.mu.Unlock()

	for ip, _ := range c.address {
		c.address[ip] = ""
	}

}

// Apply an unused address
func (c *SafeVipPool) Apply(owner string) (string, error) {

	c.mu.Lock()
	defer c.mu.Unlock()

	for ip, v := range c.address {
		if v == "" {
			c.address[ip] = owner
			return ip, nil
		}
	}

	return "", fmt.Errorf("no available ip address left")
}

// Print all ipaddress
func (c *SafeVipPool) String() string {

	var output string
	for ip, owner := range c.address {
		output += fmt.Sprintf("IP: %s, In Use: %v", ip, owner) + "\n"
	}
	return output

}
