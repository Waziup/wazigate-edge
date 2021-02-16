package codec

import (
	"github.com/Waziup/wazigate-edge/edge"
)

func init() {
	edge.Codecs["application/x-xlpp"] = XLPPCodec{}
	edge.Codecs["application/x-lpp"] = XLPPCodec{LagacyMode: true}
}

func (c XLPPCodec) CodecName() string {
	if c.LagacyMode {
		return "LPP (Cayenne Low Power Payload)"
	}
	return "XLPP (Waziup Extended Low Power Payload)"
}

type XLPPCodec struct {
	LagacyMode bool
}
