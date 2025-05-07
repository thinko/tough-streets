package dns

import (
	"log"
	"net"
	"strings"
	"time"

	"tough-streets/internal/server/config/dns" // Configuration struct
	"tough-streets/internal/server/models"  // Your data models (Device, IPAddress, etc.)
	"tough-streets/internal/server/storage" // Your storage interface

	// Import DNS library (e.g., github.com/miekg/dns)
	"github.com/miekg/dns" // Popular DNS library
	// Cache implementation (e.g., github.com/patrickmn/go-cache)
)

// Server represents the DNS service.
type Server struct {
	config dnsServerConfig
	db     storage.DataAccessLayer
	cache  *SimpleCache // A simple in-memory cache
}

// SimpleCache for DNS responses (replace with a more robust one if needed)
type SimpleCache struct {
	// For a real implementation, use something like github.com/patrickmn/go-cache
	items map[string]*dns.Msg
	ttl   time.Duration
}

func NewSimpleCache(defaultTTL time.Duration) *SimpleCache {
	return &SimpleCache{items: make(map[string]*dns.Msg), ttl: defaultTTL}
}
func (sc *SimpleCache) Get(key string) (*dns.Msg, bool) { msg, found := sc.items[key]; return msg, found } // Needs TTL check
func (sc *SimpleCache) Set(key string, msg *dns.Msg)    { sc.items[key] = msg }                        // Needs to store with expiry

// NewServer creates a new DNS server instance.
func NewServer(cfg Config, db storage.DataAccessLayer) *Server {
	return &Server{
		config: cfg,
		db:     db,
		cache:  NewSimpleCache(time.Duration(cfg.CacheTTLSec) * time.Second),
	}
}

// Start begins listening for DNS requests.
func (s *Server) Start() {
	dns.HandleFunc(".", s.handleRequest) // Handle all zones

	server := &dns.Server{Addr: s.config.ListenAddr, Net: "udp"}
	log.Printf("Starting DNS server on %s (UDP)", s.config.ListenAddr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("DNS server ListenAndServe error (UDP): %v", err)
	}
	// Optionally, also start a TCP listener
}

func (s *Server) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = false // We are mostly a resolver/forwarder

	if len(r.Question) == 0 {
		m.SetRcode(r, dns.RcodeFormatError)
		_ = w.WriteMsg(m)
		return
	}
	question := r.Question[0]
	cacheKey := question.Name + ":" + dns.TypeToString[question.Qtype]

	// 1. Check local cache
	if cachedMsg, found := s.cache.Get(cacheKey); found {
		log.Printf("DNS: Cache hit for %s", question.Name)
		// Need to copy relevant parts of cachedMsg to m, respecting original ID etc.
		m.Answer = cachedMsg.Answer
		m.Ns = cachedMsg.Ns
		m.Extra = cachedMsg.Extra
		_ = w.WriteMsg(m)
		return
	}

	// 2. Check "tough-streets" discovered devices for local domain or preference
	// Example: if question.Name ends with ".tough.lan" or s.config.LocalDomain
	if strings.HasSuffix(strings.ToLower(question.Name), "."+strings.ToLower(s.config.LocalDomain)) {
		hostname := strings.TrimSuffix(strings.ToLower(question.Name), "."+strings.ToLower(s.config.LocalDomain))
		hostname = strings.TrimSuffix(hostname, ".") // Remove trailing dot if any

		devices, _ := s.db.GetDevicesByHostname(hostname) // You'll need this DAL method
		for _, dev := range devices {
			for _, ipAddr := range dev.IPAddresses { // Assuming Device has a list of current IPs
				if question.Qtype == dns.TypeA && net.ParseIP(ipAddr.Address).To4() != nil {
					rr, _ := dns.NewRR(question.Name + " IN A " + ipAddr.Address)
					m.Answer = append(m.Answer, rr)
				}
				// Add AAAA record handling
			}
		}
		if len(m.Answer) > 0 {
			m.Authoritative = true // We are authoritative for our local domain
			log.Printf("DNS: Local resolution for %s to %v", question.Name, m.Answer)
			s.cache.Set(cacheKey, m)
			_ = w.WriteMsg(m)
			return
		}
	}

	// 3. Forward to upstream servers
	if len(s.config.Forwarders) > 0 {
		// For simplicity, query the first forwarder. Implement round-robin or concurrent queries.
		client := new(dns.Client)
		resp, _, err := client.Exchange(r, s.config.Forwarders[0]+":53")
		if err == nil {
			log.Printf("DNS: Forwarded %s, got response", question.Name)
			s.cache.Set(cacheKey, resp) // Cache the forwarded response
			_ = w.WriteMsg(resp)
			return
		}
		log.Printf("DNS: Error forwarding %s: %v", question.Name, err)
	}

	// 4. If all else fails, return ServFail
	m.SetRcode(r, dns.RcodeServerFailure)
	_ = w.WriteMsg(m)
}