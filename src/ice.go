package rtc

import "github.com/pion/webrtc/v4"

// The data format used by connecting clients (and the car) to send ICE candidates to the server
type RequestICE struct {
	Candidate webrtc.ICECandidateInit `json:"candidate"`
	Id        string                  `json:"id"`        // to distinguish between clients
	Timestamp int64                   `json:"timestamp"` // timestamp of the sender
}
