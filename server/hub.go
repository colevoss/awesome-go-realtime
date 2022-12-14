package server

import (
	"log"
	"sync"
)

type Hub struct {
	mu               sync.RWMutex
	channelsCache    map[string]*Channel
	channelFactories map[string]*ChannelFactory
}

func newHub() *Hub {
	return &Hub{
		channelsCache:    make(map[string]*Channel),
		channelFactories: make(map[string]*ChannelFactory),
	}
}

func (h *Hub) registerChannelFactory(cf *ChannelFactory) {
	log.Printf("[hub] Registering channelfactory %s", cf.path)

	if _, ok := h.channelFactories[cf.path]; ok {
		return
	}

	h.channelFactories[cf.path] = cf
}

func (h *Hub) findChannel(channelName string) (*Channel, bool) {
	log.Printf("[hub] Finding channel: %s", channelName)

	h.mu.RLock()
	defer h.mu.RUnlock()

	if channel, ok := h.channelsCache[channelName]; ok {
		log.Printf("[hub] Found channel: %s (%s)", channel.Name, channel.Path)
		return channel, ok
	}

	log.Printf("[hub] Could not find channel in cache: %s", channelName)
	return nil, false
}

func (h *Hub) findOrOpenChannel(channelName string) (*Channel, bool) {
	if channel, ok := h.findChannel(channelName); ok {
		return channel, ok
	}

	return h.openChannel(channelName)
}

func (h *Hub) openChannel(channelName string) (*Channel, bool) {
	log.Printf("[hub] Opening Channel %s", channelName)

	channelFactory, params, ok := h.findChannelFactory(channelName)

	if channelFactory == nil {
		return nil, ok
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	channel := channelFactory.newChannel(channelName, params, h)
	h.channelsCache[channelName] = channel

	return channel, true
}

func (h *Hub) closeChannel(channel *Channel) {
	log.Println("[hub] Closing channel", channel.Name)

	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.channelsCache, channel.Name)
	log.Println("[hub] Channel closed", channel.Name)
	log.Printf("[hub] %d Channels remaning", len(h.channelsCache))
}

func (h *Hub) findChannelFactory(channelName string) (*ChannelFactory, *Params, bool) {
	log.Printf("[hub] Finding channel factory for channel: %s", channelName)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, cf := range h.channelFactories {
		if doesMatch, params := cf.doesMatch(channelName); doesMatch {
			log.Printf("[hub] Found channel factory for channel: %s (%s)", channelName, cf.path)
			return cf, params, true
		}
	}

	log.Printf("[hub] Could not find matching channel factory %s", channelName)

	return nil, nil, false
}
