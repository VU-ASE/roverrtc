package rtc

import (
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
	// Data channels
	MetaChannel     *webrtc.DataChannel // the data channel used to send meta information about requests made (such as requesting a video stream)
	ControlChannel  *webrtc.DataChannel // the data channel used to send controller data (e.g. steering wheel angle)
	FrameChannel    *webrtc.DataChannel // the data channel used to send frames with video data and other meta information (such as sensoric data)
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

func (r *RTC) SendMetaMessage(protobuffer proto.Message) error {
	log := r.Log()

	// We don't need to report an error
	if r.MetaChannel == nil {
		log.Warn().Msg("Cannot send meta data. Meta channel is not configured")
		return nil
	}

	// Create bytes from protobuf struct
	content, err := proto.Marshal(protobuffer)
	if err != nil {
		return err
	}

	return r.MetaChannel.Send(content)
}

func (r *RTC) SendFrameBytes(data []byte) error {
	log := r.Log()

	// We don't need to report an error to the caller, but we can report it to the client over the meta channel
	if r.FrameChannel == nil {
		log.Warn().Msg("Cannot send frame data. Frame channel is not configured")
		return nil
	}

	return r.FrameChannel.Send(data)
}

func (r *RTC) SendControlBytes(data []byte) error {
	log := r.Log()

	// We don't need to report an error to the caller, but we can report it to the client over the meta channel
	if r.ControlChannel == nil {
		log.Warn().Msg("Cannot send control data. Control channel is not configured")
		return nil
	}

	return r.ControlChannel.Send(data)
}
