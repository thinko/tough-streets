package dns

import (
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
	"github.com/cenkalti/backoff/v4"

	"tough-streets/internal/server/config"
	"tough-streets/internal/server/models"
	"tough-streets/internal/server/storage"
	"tough-streets/internal/logger"
	"tough-streets/internal/metrics"
)

// Server represents the DNS service
type Server struct {
	config config.DNSConfig
	db     storage.DataAccessLayer
	cache  *cache.Cache
	log    *logger.Logger
}

// NewServer creates a new DNS server instance
func NewServer(cfg config.DNSConfig, db storage.DataAccessLayer) *Server {
	return &Server{
		config: cfg,
		db:     db,
		cache:  cache.New(time.Duration(cfg.CacheTTLSec)*time.Second, time.Duration(cfg.CacheTTLSec)*time.Second*2),
		log:    logger.GetLogger(),
	}
}

// Start begins listening for DNS requests
func (s *Server) Start() error {
	dns.HandleFunc(".", s.handleRequest)

	udpServer := &dns.Server{Addr: s.config.ListenAddr, Net: "udp"}
	tcpServer := &dns.Server{Addr: s.config.ListenAddr, Net: "tcp"}

	s.log.Infof("Starting DNS server on %s (UDP/TCP)", s.config.ListenAddr)
	metrics.StorageOperations.WithLabelValues("dns_server", "start").Inc()

	// Start UDP server
	go func() {
		err := udpServer.ListenAndServe()
		if err != nil {
			s.log.WithError(err).Error("UDP server failure")
			metrics.StorageOperations.WithLabelValues("dns_server", "error").Inc()
		}
	}()

	// Start TCP server
	go func() {
		err := tcpServer.ListenAndServe()
		if err != nil {
			s.log.WithError(err).Error("TCP server failure")
			metrics.StorageOperations.WithLabelValues("dns_server", "error").Inc()
		}
	}()

	return nil
}

func (s *Server) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	startTime := time.Now()
	defer func() {
		metrics.StorageLatency.WithLabelValues("dns_query").Observe(time.Since(startTime).Seconds())
	}()

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = false

	log := s.log.WithField("operation", "dns_query")

	if len(r.Question) == 0 {
		metrics.StorageOperations.WithLabelValues("dns_query", "format_error").Inc()
		m.SetRcode(r, dns.RcodeFormatError)
		_ = w.WriteMsg(m)
		return
	}

	question := r.Question[0]
	cacheKey := question.Name + ":" + dns.TypeToString[question.Qtype]

	log = log.WithFields(logger.Fields{
		"query_name": question.Name,
		"query_type": dns.TypeToString[question.Qtype],
	})

	// Try handling the request with exponential backoff
	operation := func() error {
		// 1. Check local cache
		if cachedItem, found := s.cache.Get(cacheKey); found {
			metrics.StorageOperations.WithLabelValues("dns_query", "cache_hit").Inc()
			log.Debug("Cache hit")
			if cachedMsg, ok := cachedItem.(*dns.Msg); ok {
				m.Answer = cachedMsg.Answer
				m.Ns = cachedMsg.Ns
				m.Extra = cachedMsg.Extra
				return w.WriteMsg(m)
			}
		}

		// 2. Check local domain
		if strings.HasSuffix(strings.ToLower(question.Name), "."+strings.ToLower(s.config.LocalDomain)) {
			hostname := strings.TrimSuffix(strings.ToLower(question.Name), "."+strings.ToLower(s.config.LocalDomain))
			hostname = strings.TrimSuffix(hostname, ".")

			devices, err := s.db.GetDevicesByHostname(hostname)
			if err != nil {
				log.WithError(err).Error("Failed to query devices")
				return err
			}

			for _, dev := range devices {
				for _, ipAddr := range dev.IPAddresses {
					if question.Qtype == dns.TypeA && net.ParseIP(ipAddr.Address).To4() != nil {
						rr, err := dns.NewRR(question.Name + " IN A " + ipAddr.Address)
						if err == nil {
							m.Answer = append(m.Answer, rr)
						}
					} else if question.Qtype == dns.TypeAAAA && net.ParseIP(ipAddr.Address).To16() != nil && net.ParseIP(ipAddr.Address).To4() == nil {
						rr, err := dns.NewRR(question.Name + " IN AAAA " + ipAddr.Address)
						if err == nil {
							m.Answer = append(m.Answer, rr)
						}
					}
				}
			}

			if len(m.Answer) > 0 {
				m.Authoritative = true
				metrics.StorageOperations.WithLabelValues("dns_query", "local_resolution").Inc()
				log.WithField("answers", len(m.Answer)).Info("Local resolution successful")
				s.cache.Set(cacheKey, m, cache.DefaultExpiration)
				return w.WriteMsg(m)
			}
		}

		// 3. Forward to upstream servers
		if len(s.config.Forwarders) > 0 {
			client := new(dns.Client)
			for _, forwarder := range s.config.Forwarders {
				resp, _, err := client.Exchange(r, forwarder+":53")
				if err == nil {
					metrics.StorageOperations.WithLabelValues("dns_query", "forwarded").Inc()
					log.WithField("forwarder", forwarder).Debug("Forwarded query successful")
					s.cache.Set(cacheKey, resp, cache.DefaultExpiration)
					return w.WriteMsg(resp)
				}
				log.WithError(err).WithField("forwarder", forwarder).Warn("Forward failed")
			}
		}

		// 4. If all else fails, return ServFail
		metrics.StorageOperations.WithLabelValues("dns_query", "servfail").Inc()
		m.SetRcode(r, dns.RcodeServerFailure)
		return w.WriteMsg(m)
	}

	// Configure exponential backoff
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = 3 * time.Second // Set a reasonable timeout

	// Execute with backoff
	if err := backoff.Retry(operation, backoffConfig); err != nil {
		log.WithError(err).Error("Failed to handle DNS request after retries")
		metrics.StorageOperations.WithLabelValues("dns_query", "error").Inc()
	}
}
