package rtc

import (
	"fmt"
	"sync"

	// Add zerolog
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/pion/webrtc/v4"
)

//
// The RTC package is shared between the server and client and contains a struct and helper functions and methods to manage a webRTC connection
// or a bundle of connections (i.e. the RTCMap)
//

type RTC struct {
	Id             string                    // the id of the connection (e.g. the client id)
	Pc             *webrtc.PeerConnection    // the actual webRTC connection
	Candidates     []webrtc.ICECandidateInit // the **local** ICE candidates (that can be transmitted to the other peers)
	CandidatesLock *sync.Mutex               // to make sure ICE candidates can be managed concurrently
	// Communication channels
	ControlChannel  *webrtc.DataChannel // the data channel used for the control protocol between server and client
	DataChannel     *webrtc.DataChannel // the data channel used to send debugging information and tuning state
	TimestampOffset int64               // the timestamp offset to calculate the time difference between the client and the server
}

// Create an easy function to get a logger with the context and connection id already set
func (r *RTC) Log() zerolog.Logger {
	logger := log.With().Str("context", "rtc").Str("connectionId", r.Id).Logger()
	return logger
}

func NewRTC(id string) *RTC {
	var candidatesMux sync.Mutex
	candidates := make([]webrtc.ICECandidateInit, 0)

	return &RTC{
		Id:              id,
		Candidates:      candidates,
		CandidatesLock:  &candidatesMux,
		TimestampOffset: 0,
	}
}

// Add a local ICE candidate to the list of candidates fetched so far
func (r *RTC) AddLocalCandidate(candidate webrtc.ICECandidateInit) {
	log := r.Log()

	r.CandidatesLock.Lock()
	defer r.CandidatesLock.Unlock()

	r.Candidates = append(r.Candidates, candidate)
	log.Debug().Msg("Added local ICE candidate")
}

// Get a copy of all local ICE candidates (concurrency-safe)
func (r *RTC) GetAllLocalCandidates() []webrtc.ICECandidateInit {
	r.CandidatesLock.Lock()
	defer r.CandidatesLock.Unlock()

	original := r.Candidates
	safeCandidates := make([]webrtc.ICECandidateInit, len(original))
	copy(safeCandidates, original)

	return safeCandidates
}

// Destroy an RTC object and the underlying webRTC connection
func (r *RTC) Destroy() {
	log := r.Log()

	if r.Pc == nil {
		log.Warn().Msg("Cannot destroy RTC connection. Connection is nil")
		return
	}

	if err := r.Pc.Close(); err != nil {
		log.Err(err).Msg("Cannot close RTC connection")
	}

	r.CandidatesLock.Lock()
	defer r.CandidatesLock.Unlock()
	r.Candidates = make([]webrtc.ICECandidateInit, 0)

	r.Pc = nil
	log.Debug().Msg("Destroyed RTC connection")
}

// Utility function to check if the connection is still active
func (r *RTC) IsConnected() bool {
	return r.Pc.ConnectionState() == webrtc.PeerConnectionStateConnected
}

//
// Wrapper functions to easily send on the data channels, without having to check if they are nil every time
//

// Sending on the data channel
func (r *RTC) SendData(pb proto.Message) error {
	content, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	return r.SendDataBytes(content)
}
func (r *RTC) SendDataBytes(b []byte) error {
	log := r.Log()

	if r.DataChannel == nil {
		log.Warn().Msg("Cannot send on data channel. Data channel is not configured")
		return fmt.Errorf("Data channel is not configured")
	}
	return r.DataChannel.Send(b)
}

// Sending on the control channel
func (r *RTC) SendControlData(pb proto.Message) error {
	content, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	return r.SendControlBytes(content)
}
func (r *RTC) SendControlBytes(b []byte) error {
	log := r.Log()

	if r.ControlChannel == nil {
		log.Warn().Msg("Cannot send control data. Control channel is not configured")
		return fmt.Errorf("Control channel is not configured")
	}

	return r.ControlChannel.Send(b)
}
