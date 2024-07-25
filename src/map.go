package rtc

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"sync"

	"github.com/pion/webrtc/v4"
)

//
// This file contains everything related to the RTCMap
// this map is used to conveniently store all RTC connections in a thread-safe way
//

type RTCMap struct {
	rtcMap map[string]*RTC // id -> RTC
	lock   *sync.RWMutex
}

func NewRTCMap() *RTCMap {
	var lock sync.RWMutex
	rtcMap := make(map[string]*RTC)

	return &RTCMap{
		rtcMap: rtcMap,
		lock:   &lock,
	}
}

// Remove an RTC connection from the map
func (m *RTCMap) Remove(id string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	conn := m.rtcMap[id]
	if conn == nil {
		return fmt.Errorf("Connection with id %s does not exist", id)
	}

	delete(m.rtcMap, id)
	log.Debug().Str("rtcId", id).Msg("Removed RTC connection from map")
	return nil
}

func (m *RTCMap) Add(id string, rtc *RTC, isCar bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	maxLength := 10

	if len(m.rtcMap) >= maxLength && !isCar {
		return fmt.Errorf("Maximum number of connections reached")
	}

	existingEntry := m.rtcMap[id]
	if existingEntry != nil && existingEntry.Pc.ConnectionState() != webrtc.PeerConnectionStateClosed && existingEntry.Pc.ConnectionState() != webrtc.PeerConnectionStateDisconnected {
		return fmt.Errorf("An active connection with id %s already exists.", id)
	}

	// Remove the entry (so that the connection is properly closed)
	if existingEntry != nil {
		err := m.Remove(id)
		if err != nil {
			return err
		}
	}

	m.rtcMap[id] = rtc
	log.Debug().Str("rtcId", id).Msg("Added RTC connection to map")
	return nil
}

// Returns a pointer to the RTC connection with the given id (concurrency-safe)
func (m *RTCMap) Get(id string) *RTC {
	m.lock.RLock()
	defer m.lock.RUnlock()

	rtc := m.rtcMap[id]

	return rtc
}

// Returns a copy of all Ids in the map (concurrency-safe)
func (m *RTCMap) GetAllIds() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	ids := make([]string, 0)
	for id := range m.rtcMap {
		ids = append(ids, id)
	}

	return ids
}

// Returns a list of all RTC connections in the map. Locks when reading the map but returns a list of pointers.
// If you want to execute a function for each RTC connection, use ForEach instead.
func (m *RTCMap) UnsafeGetAll() []*RTC {
	m.lock.RLock()
	defer m.lock.RUnlock()

	rtcList := make([]*RTC, 0)
	for _, rtc := range m.rtcMap {
		rtcList = append(rtcList, rtc)
	}

	return rtcList
}

// Executes a function for each RTC connection in the map
// Because the RTCMap is read locked during execution, this is concurrency-safe
func (m *RTCMap) ForEach(f func(id string, rtc *RTC)) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for id, rtc := range m.rtcMap {
		f(id, rtc)
	}
}
