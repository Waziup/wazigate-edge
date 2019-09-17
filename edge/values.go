package edge

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/globalsign/mgo/bson"
)

// Value is one datapoint
type Value struct {
	Value interface{} `json:"value" bson:"value"`
	Time  time.Time   `json:"time" bson:"time"`
}

// Query is used to range or limit query results
type Query struct {
	Limit int64
	From  time.Time
	To    time.Time
	Size  int64
}

// ValueIterator iterates over data points. Call .Next() to get the next value.
type ValueIterator interface {
	Next() (Value, error)
	Close() error
}

var errNotFound = CodeError{404, "device or sensor/actuator not found"}

////////////////////////////////////////////////////////////////////////////////

func newID(t time.Time) bson.ObjectId {
	id := []byte(bson.NewObjectId())
	timeID := []byte(bson.NewObjectIdWithTime(t))
	copy(id[:4], timeID[:4])
	return bson.ObjectId(id)
}

var sizeRegex = regexp.MustCompile(`^\d+[kKmMgG]?[bB]?`)
var sizeUnitRegex = regexp.MustCompile(`[kKmMgG]?[bB]?$`)

func (query *Query) Parse(req *http.Request) string {

	var param string
	var err error

	q := req.URL.Query()

	if param = q.Get("from"); param != "" {
		err = query.From.UnmarshalText([]byte(param))
		if err != nil {
			return "Query ?from=.. is mal formatted."
		}
	}

	if param = q.Get("to"); param != "" {
		err = query.To.UnmarshalText([]byte(param))
		if err != nil {
			return "Query ?to=.. is mal formatted."
		}
	}

	if param = q.Get("limit"); param != "" {
		query.Limit, err = strconv.ParseInt(param, 10, 64)
		if err != nil {
			return "Query ?limit=.. is mal formatted."
		}
	}

	if param = q.Get("size"); param != "" {
		query.Size = parseSize(param)
		if query.Size == -1 {
			return "Query ?size=.. is mal formatted."
		}
	}

	return ""
}

func parseSize(str string) (size int64) {
	for len(str) != 0 {
		match := sizeRegex.FindString(str)
		if match == "" {
			return -1
		}
		unit := sizeUnitRegex.FindString(str)
		var fact int64 = 1
		if len(unit) > 0 {
			if unit[0] == 'k' || unit[0] == 'K' {
				fact = 1e3
			} else if unit[0] == 'm' || unit[0] == 'M' {
				fact = 1e6
			} else if unit[0] == 'g' || unit[0] == 'G' {
				fact = 1e9
			}
		}
		n, _ := strconv.ParseInt(match[0:len(match)-len(unit)], 10, 64)
		size += n * fact
		str = str[len(match):]
	}
	return
}
