package codec

import (
	"github.com/Waziup/wazigate-edge/edge"
)

func init() {
	edge.Codecs["application/x-xlpp"] = XLPPCodec{}
	edge.Codecs["application/x-lpp"] = XLPPCodec{LagacyMode: true}
}

type XLPPCodec struct {
	LagacyMode bool
}
