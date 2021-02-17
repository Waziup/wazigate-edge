package edge

import (
	"io"
	"net/http"
	"time"

	"github.com/globalsign/mgo"
)

type Codec interface {
	UnmarshalDevice(deviceID string, headers http.Header, r io.Reader) error
	MarshalDevice(deviceID string, headers http.Header, w io.Writer) error
	CodecName() string
}

type ScriptExecutor interface {
	UnmarshalDevice(script *ScriptCodec, deviceID string, headers http.Header, r io.Reader) error
	MarshalDevice(script *ScriptCodec, deviceID string, headers http.Header, w io.Writer) error
	ExecutorName() string
}

var Codecs = map[string]Codec{}
var ScriptExecutors = map[string]ScriptExecutor{}

type ScriptCodec struct {
	ID        string `json:"id" bson:"_id"`
	Internal  bool   `json:"internal" bson:"-"`
	Name      string `json:"name" bson:"name"`
	ServeMime string `json:"serveMime" bson:"serveMime"`
	Mime      string `json:"mime" bson:"mime"`
	Script    string `json:"script" bson:"script"`
}

////////////////////////////////////////////////////////////////////////////////

var errNoExecutor = NewError(500, "the codec uses a mime that is unknown to the system")

func (script *ScriptCodec) UnmarshalDevice(deviceID string, headers http.Header, r io.Reader) error {
	e := ScriptExecutors[script.Mime]
	if e == nil {
		return errNoExecutor
	}
	return e.UnmarshalDevice(script, deviceID, headers, r)
}

func (script *ScriptCodec) MarshalDevice(deviceID string, headers http.Header, w io.Writer) error {
	e := ScriptExecutors[script.Mime]
	if e == nil {
		return errNoExecutor
	}
	return e.MarshalDevice(script, deviceID, headers, w)
}

func (codec ScriptCodec) CodecName() string {

	return codec.Name
}

////////////////////////////////////////////////////////////////////////////////

func PostCodec(codec *ScriptCodec) error {
	if codec.ID == "" {
		codec.ID = newID(time.Now()).Hex()
	}
	_, ok := ScriptExecutors[codec.Mime]
	if !ok {
		return errNoExecutor
	}
	_, err := dbCodecs.UpsertId(codec.ID, codec)
	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func DeleteCodec(id string) error {
	err := dbCodecs.RemoveId(id)
	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// CodecsIter iterates over codecs. Call .Next() to get the next codec.
type CodecsIter struct {
	internals []string
	codec     ScriptCodec
	dbIter    *mgo.Iter
}

// Next returns the next codec or nil.
func (iter *CodecsIter) Next() (*ScriptCodec, error) {
	if len(iter.internals) != 0 {
		mime := iter.internals[0]
		codec := Codecs[mime]
		iter.codec = ScriptCodec{
			ID:        mime,
			ServeMime: mime,
			Name:      codec.CodecName(),
			Mime:      "application/octet-stream",
			Script:    "<internal>",
			Internal:  true,
		}
		iter.internals = iter.internals[1:]
		return &iter.codec, nil
	}
	if iter.dbIter.Next(&iter.codec) {
		iter.codec.Internal = false
		return &iter.codec, iter.dbIter.Err()
	}
	return nil, io.EOF
}

// Close closes the iterator.
func (iter *CodecsIter) Close() error {
	return iter.dbIter.Close()
}

// GetCodecs returns an iterator over all codecs.
func GetCodecs() *CodecsIter {

	internals := make([]string, len(Codecs))
	i := 0
	for mime, _ := range Codecs {
		internals[i] = mime
		i++
	}

	q := dbCodecs.Find(nil)

	return &CodecsIter{
		internals: internals,
		dbIter:    q.Iter(),
	}
}
