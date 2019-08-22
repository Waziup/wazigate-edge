package mqtt

// _ "github.com/j-forster/mqtt/mqtt.v5.0"

/*
type ConnectCode byte

const (
	CodeAccepted ConnectCode = iota
	CodeUnacceptableProtoV
	CodeIDentifierRejected
	CodeServerUnavaliable
	CodeBatUserOrPassword
	CodeNotAuthorized
)

var codeNames = [...]string{
	"Connection accepted.",
	"The Server does not support the level of the MQTT protocol requested by the Client.",
	"The Client identifier is correct UTF-8 but not allowed by the Server.",
	"The Network Connection has been made but the MQTT service is unavailable.",
	"The data in the user name or password is malformed.",
	"The Client is not authorized to connect.",
}

func (c ConnectCode) String() string {
	if c > 0 && int(c) < len(codeNames) {
		return codeNames[c]
	}
	return "Unknown code."
}
*/

// PacketType is the  MQTT Control Packet type.
// See https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901022
type PacketType byte

// List of packet types
const (
	CONNECT     PacketType = 1
	CONNACK                = 2
	PUBLISH                = 3
	PUBACK                 = 4
	PUBREC                 = 5
	PUBREL                 = 6
	PUBCOMP                = 7
	SUBSCRIBE              = 8
	SUBACK                 = 9
	UNSUBSCRIBE            = 10
	UNSUBACK               = 11
	PINGREQ                = 12
	PINGRESP               = 13
	DISCONNECT             = 14
)

var packetTypeNames = [...]string{
	"",
	"CONNECT",
	"CONNACK",
	"PUBLISH",
	"PUBACK",
	"PUBREC",
	"PUBREL",
	"PUBCOMP",
	"SUBSCRIBE",
	"SUBACK",
	"UNSUBSCRIBE",
	"UNSUBACK",
	"PINGREQ",
	"PINGRESP",
	"DISCONNECT",
}

func (p PacketType) String() string {
	if p > 0 && int(p) < len(packetTypeNames) {
		return packetTypeNames[p]
	}
	return "<unknown packet type>"
}

////////////////////////////////////////////////////////////////////////////////

// Property fields
// See https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901027
type Property byte

// Packet Properties
const (
	PropPayloadFormat     Property = 0x01 // Payload Format Indicator
	PropMsgExpiry                  = 0x02 // Message Expiry Interval
	PropContentType                = 0x03 // Content Type
	PropRespTopic                  = 0x08 // Response Topic
	PropCorrelData                 = 0x09 // Correlation Data
	PropSubsIdent                  = 0x0B // Subscription Identifier
	PropSessExpiry                 = 0x11 // Session Expiry Interval
	PropClientIdet                 = 0x12 // Assigned Client Identifier
	PropServerKeepAlive            = 0x13 // Server Keep Alive
	PropAuthMethod                 = 0x15 // Authentication Method
	PropAuthData                   = 0x16 // Authentication Data
	PropReqProblem                 = 0x17 // Request Problem Information
	PropWillDelay                  = 0x18 // Will Delay Interval
	PropReqRespInfo                = 0x19 // Request Response Information
	PropRespInfo                   = 0x1A // Response Information
	PropServerRef                  = 0x1C // Server Reference
	PropReason                     = 0x1F // Reason String
	PropRecvMax                    = 0x21 // Receive Maximum
	PropTopicAliasMax              = 0x22 // Topic Alias Maximum
	PropTopicAlias                 = 0x23 // Topic Alias
	PropMaxQoS                     = 0x24 // Maximum QoS
	PropRetainAvailable            = 0x25 // Retain Available
	PropUser                       = 0x26 // User Property
	PropMaxPacket                  = 0x27 // Maximum Packet Size
	PropWildcardSubsAvail          = 0x28 // Wildcard Subscription Available
	PropSubsIdentAvail             = 0x29 // Subscription Identifier Available
	PropSharedSubsAvail            = 0x2A // Shared Subscription Available
)

func (prop Property) String() string {
	switch prop {
	case PropPayloadFormat:
		return "Payload Format Indicator"
	case PropMsgExpiry:
		return "Message Expiry Interval"
	case PropContentType:
		return "Content Type"
	case PropRespTopic:
		return "Response Topic"
	case PropCorrelData:
		return "Correlation Data"
	case PropSubsIdent:
		return "Subscription Identifier"
	case PropSessExpiry:
		return "Session Expiry Interval"
	case PropClientIdet:
		return "Assigned Client Identifier"
	case PropServerKeepAlive:
		return "Server Keep Alive"
	case PropAuthMethod:
		return "Authentication Method"
	case PropAuthData:
		return "Authentication Data"
	case PropReqProblem:
		return "Request Problem Information"
	case PropWillDelay:
		return "Will Delay Interval"
	case PropReqRespInfo:
		return "Request Response Information"
	case PropRespInfo:
		return "Response Information"
	case PropServerRef:
		return "Server Reference"
	case PropReason:
		return "Reason String"
	case PropRecvMax:
		return "Receive Maximum"
	case PropTopicAliasMax:
		return "Topic Alias Maximum"
	case PropTopicAlias:
		return "Topic Alias"
	case PropMaxQoS:
		return "Maximum QoS"
	case PropRetainAvailable:
		return "Retain Available"
	case PropUser:
		return "User Property"
	case PropMaxPacket:
		return "Maximum Packet Size"
	case PropWildcardSubsAvail:
		return "Wildcard Subscription Available"
	case PropSubsIdentAvail:
		return "Subscription Identifier Available"
	case PropSharedSubsAvail:
		return "Shared Subscription Available"
	default:
		return "<unknown property>"
	}
}

////////////////////////////////////////////////////////////////////////////////

// A ReasonCode is a one byte unsigned value that indicates the result of an operation.
// Reason Codes less than 0x80 indicate successful completion of an operation.
// The normal Reason Code for success is 0.
// Reason Code values of 0x80 or greater indicate failure.
// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901031
type ReasonCode byte

// Reason Codes
const (
	ReasonSuccess           ReasonCode = 0x00 // Success
	ReasonDisconnect        ReasonCode = 0x00 // Normal disconnection
	ReasonQoS0              ReasonCode = 0x00 // Granted QoS 0
	ReasonQoS1              ReasonCode = 0x01 // Granted QoS 1
	ReasonQoS2              ReasonCode = 0x02 // Granted QoS 2
	ReasonDisconnectWill    ReasonCode = 0x04 // Disconnect with Will Message
	ReasonNoSubsMatch       ReasonCode = 0x10 // No matching subscribers
	ReasonNoSubsExist       ReasonCode = 0x11 // No subscription existed
	ReasonContinueAuth      ReasonCode = 0x18 // Continue authentication
	ReasonReAuth            ReasonCode = 0x19 // Re-authenticate
	ReasonUnspecErr         ReasonCode = 0x80 // Unspecified error
	ReasonMalfPacket        ReasonCode = 0x81 // Malformed Packet
	ReasonProtoErr          ReasonCode = 0x82 // Protocol Error
	ReasonImplSpec          ReasonCode = 0x83 // Implementation specific
	ReasonUnsupProtoV       ReasonCode = 0x84 // Unsupported Protocol Version
	ReasonClientIDInvalid   ReasonCode = 0x85 // Client Identifier not valid
	ReasonBadAuth           ReasonCode = 0x86 // Bad User Name or Password
	ReasonNotAuth           ReasonCode = 0x87 // Not authorized
	ReasonUnavail           ReasonCode = 0x88 // Server unavailable
	ReasonBusy              ReasonCode = 0x89 // Server busy
	ReasonBanned            ReasonCode = 0x8A // Banned
	ReasonSuttingDown       ReasonCode = 0x8B // Server shutting down
	ReasonBadAuthMehtod     ReasonCode = 0x8C // Bad authentication method
	ReasonKeepAliveTimeout  ReasonCode = 0x8D // Keep Alive timeout
	ReasonSessionTaken      ReasonCode = 0x8E // Session taken over
	ReasonInvalFilter       ReasonCode = 0x8F // Topic Filter invalid
	ReasonInvalName         ReasonCode = 0x90 // Topic Name invalid
	ReasonIdentInUse        ReasonCode = 0x91 // Packet Identifier in use
	ReasonIdentNotFound     ReasonCode = 0x92 // Packet Identifier not found
	ReasonMaxReceive        ReasonCode = 0x93 // Receive Maximum exceeded
	ReasonInvalAlias        ReasonCode = 0x94 // Topic Alias invalid
	ReasonTooLarge          ReasonCode = 0x95 // Packet too large
	ReasonHighRate          ReasonCode = 0x96 // Message rate too high
	ReasonQuotaExceeded     ReasonCode = 0x97 // Quota exceeded
	ReasonAdmin             ReasonCode = 0x98 // Administrative action
	ReasonInvalFormat       ReasonCode = 0x99 // Payload format invalid
	ReasonUnsupRetain       ReasonCode = 0x9A // Retain not supported
	ReasonUnsupQoS          ReasonCode = 0x9B // QoS not supported
	ReasonUseAnother        ReasonCode = 0x9C // Use another server
	ReasonMoved             ReasonCode = 0x9D // Server moved
	ReasonUnsubShared       ReasonCode = 0x9E // Shared Subscriptions not supported
	ReasonConnReateExceeded ReasonCode = 0x9F // Connection rate exceeded
	ReasonMaxConnTime       ReasonCode = 0xA0 // Maximum connect time
	ReasonUnsupSubsIdent    ReasonCode = 0xA1 // Subscription Identifiers not supported
	ReasonUnsupWildcard     ReasonCode = 0xA2 // Wildcard Subscriptions not supported
)

type reasonError ReasonCode

func (err reasonError) Error() string {
	return ReasonCode(err).String()
}

// Error transforms the RasonCode to an error.
// The error is nil if the reason is `ReasonSuccess` (which is not an error).
func (code ReasonCode) Error() error {
	if code == 0x00 {
		return nil
	}
	return reasonError(code)
}

func (code ReasonCode) String() string {
	switch code {
	// They are all 0x00:
	/*
		case ReasonSuccess:
			return "Success"
		case ReasonDisconnect:
			return "Normal disconnection"
		case ReasonQoS0:
			return "Granted QoS 0"
	*/
	case 0x00:
		return ""

	case ReasonQoS1:
		return "Granted QoS 1"
	case ReasonQoS2:
		return "Granted QoS 2"
	case ReasonDisconnectWill:
		return "Disconnect with Will Message"
	case ReasonNoSubsMatch:
		return "No matching subscribers"
	case ReasonNoSubsExist:
		return "No subscription existed"
	case ReasonContinueAuth:
		return "Continue authentication"
	case ReasonReAuth:
		return "Re-authenticate"
	case ReasonUnspecErr:
		return "Unspecified error"
	case ReasonMalfPacket:
		return "Malformed Packet"
	case ReasonProtoErr:
		return "Protocol Error"
	case ReasonImplSpec:
		return "Implementation specific"
	case ReasonUnsupProtoV:
		return "Unsupported Protocol Version"
	case ReasonClientIDInvalid:
		return "Client Identifier not valid"
	case ReasonBadAuth:
		return "Bad User Name or Password"
	case ReasonNotAuth:
		return "Not authorized"
	case ReasonUnavail:
		return "Server unavailable"
	case ReasonBusy:
		return "Server busy"
	case ReasonBanned:
		return "Banned"
	case ReasonSuttingDown:
		return "Server shutting down"
	case ReasonBadAuthMehtod:
		return "Bad authentication method"
	case ReasonKeepAliveTimeout:
		return "Keep Alive timeout"
	case ReasonSessionTaken:
		return "Session taken over"
	case ReasonInvalFilter:
		return "Topic Filter invalid"
	case ReasonInvalName:
		return "Topic Name invalid"
	case ReasonIdentInUse:
		return "Packet Identifier in use"
	case ReasonIdentNotFound:
		return "Packet Identifier not found"
	case ReasonMaxReceive:
		return "Receive Maximum exceeded"
	case ReasonInvalAlias:
		return "Topic Alias invalid"
	case ReasonTooLarge:
		return "Packet too large"
	case ReasonHighRate:
		return "Message rate too high"
	case ReasonQuotaExceeded:
		return "Quota exceeded"
	case ReasonAdmin:
		return "Administrative action"
	case ReasonInvalFormat:
		return "Payload format invalid"
	case ReasonUnsupRetain:
		return "Retain not supported"
	case ReasonUnsupQoS:
		return "QoS not supported"
	case ReasonUseAnother:
		return "Use another server"
	case ReasonMoved:
		return "Server moved"
	case ReasonUnsubShared:
		return "Shared Subscriptions not supported"
	case ReasonConnReateExceeded:
		return "Connection rate exceeded"
	case ReasonMaxConnTime:
		return "Maximum connect time"
	case ReasonUnsupSubsIdent:
		return "Subscription Identifiers not supported"
	case ReasonUnsupWildcard:
		return "Wildcard Subscriptions not supported"
	default:
		return "<unknown reason>"
	}
}
