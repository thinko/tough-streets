// Package processor provides packet processing functionality for tough-streets.
// It implements the core packet analysis pipeline, metrics collection, and structured logging.
// The package is designed to be extensible, allowing for custom processors to be added
// for specific protocol analysis needs.
package processor

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"tough-streets/internal/logger"
	"tough-streets/internal/metrics"
	"tough-streets/internal/transport"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// BaseProcessor implements basic packet processing functionality
type BaseProcessor struct {
	packetsProcessed uint64
	packetsDropped   uint64
	log             *logger.Logger
}

// NewBaseProcessor creates a new base processor instance
func NewBaseProcessor() *BaseProcessor {
	return &BaseProcessor{
		log: logger.GetLogger(),
	}
}

// Process implements PacketProcessor interface
func (p *BaseProcessor) Process(ctx context.Context, packet transport.CapturedPacket) error {
	startTime := time.Now()
	defer func() {
		metrics.ProcessingLatency.WithLabelValues(packet.Metadata.InterfaceName).Observe(time.Since(startTime).Seconds())
	}()

	defer atomic.AddUint64(&p.packetsProcessed, 1)

	// Create packet data for processing
	pkt := gopacket.NewPacket(packet.Data, layers.LayerTypeEthernet, gopacket.Default)
	if pkt.ErrorLayer() != nil {
		atomic.AddUint64(&p.packetsDropped, 1)
		metrics.PacketsDropped.WithLabelValues(packet.Metadata.InterfaceName, "decode_error").Inc()
		err := fmt.Errorf("error decoding packet: %v", pkt.ErrorLayer().Error())
		p.log.WithError(err).Error("Failed to decode packet")
		return err
	}

	// Process Ethernet layer
	ethLayer := pkt.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		metrics.PacketsDropped.WithLabelValues(packet.Metadata.InterfaceName, "no_ethernet").Inc()
		return fmt.Errorf("no ethernet layer found")
	}
	eth, _ := ethLayer.(*layers.Ethernet)

	// Track frame type metrics
	frameType := "unicast"
	if eth.DstMAC.String() == "ff:ff:ff:ff:ff:ff" {
		frameType = "broadcast"
	} else if eth.DstMAC[0]&0x01 == 1 {
		frameType = "multicast"
	}
	metrics.PacketsProcessed.WithLabelValues(packet.Metadata.InterfaceName, frameType).Inc()

	// Process protocol layers and collect metrics
	p.processProtocolLayers(ctx, pkt, packet.Metadata.InterfaceName)

	// TODO: Process application layer protocols (DNS, DHCP, LLDP, etc.)
	// TODO: Send processed data to aggregation engine

	return nil
}

// processProtocolLayers analyzes and collects metrics for each protocol layer
func (p *BaseProcessor) processProtocolLayers(ctx context.Context, pkt gopacket.Packet, interfaceName string) error {
	log := p.log

	// Process VLAN tags if present
	if vlanLayer := pkt.Layer(layers.LayerTypeDot1Q); vlanLayer != nil {
		if vlan, ok := vlanLayer.(*layers.Dot1Q); ok {
			log = log.WithField("vlan_id", vlan.Type)
		}
	}

	// Process IP layer
	if ipv4Layer := pkt.Layer(layers.LayerTypeIPv4); ipv4Layer != nil {
		ipv4, _ := ipv4Layer.(*layers.IPv4)
		log = log.WithField("src_ip", ipv4.SrcIP.String()).WithField("dst_ip", ipv4.DstIP.String())
		metrics.PacketsProcessed.WithLabelValues(interfaceName, "ipv4").Inc()
	} else if ipv6Layer := pkt.Layer(layers.LayerTypeIPv6); ipv6Layer != nil {
		ipv6, _ := ipv6Layer.(*layers.IPv6)
		log = log.WithField("src_ip", ipv6.SrcIP.String()).WithField("dst_ip", ipv6.DstIP.String())
		metrics.PacketsProcessed.WithLabelValues(interfaceName, "ipv6").Inc()
	}

	// Process transport layer
	if tcpLayer := pkt.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		log = log.WithField("src_port", tcp.SrcPort.String()).
			WithField("dst_port", tcp.DstPort.String()).
			WithField("protocol", "tcp")

		// Track retransmissions
		if tcp.RST || tcp.FIN {
			metrics.PacketsProcessed.WithLabelValues(interfaceName, "tcp_retransmit").Inc()
		}
	} else if udpLayer := pkt.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		log = log.WithField("src_port", udp.SrcPort.String()).
			WithField("dst_port", udp.DstPort.String()).
			WithField("protocol", "udp")
	}

	// Log packet details at debug level
	log.Debug("Processed packet")

	return nil
}

// Stats returns packet processing statistics
func (p *BaseProcessor) Stats() (uint64, uint64) {
	return atomic.LoadUint64(&p.packetsProcessed), atomic.LoadUint64(&p.packetsDropped)
}
