package rtc

import "github.com/pion/webrtc/v4"

// The data format used for SDP requests
type RequestSDP struct {
	Offer     webrtc.SessionDescription `json:"offer"`
	Id        string                    `json:"id"`        // to distinguish between clients
	Timestamp int64                     `json:"timestamp"` // timestamp of the sender
}
