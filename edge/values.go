package edge

import (
	"time"

	"github.com/globalsign/mgo"
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
type ValueIterator struct {
	value  Value
	dbIter *mgo.Iter
}

// Next returns the next value or nil.
func (iter *ValueIterator) Next() (*Value, error) {

	end := iter.dbIter.Next(&iter.value)
	if end {
		return nil, iter.dbIter.Err()
	}
	return &iter.value, iter.dbIter.Err()
}

// Close closes the iterator.
func (iter *ValueIterator) Close() error {
	return iter.dbIter.Close()
}

var errNotFound = CodeError{404, "device or sensor/actuator not found"}

////////////////////////////////////////////////////////////////////////////////

func newID(t time.Time) bson.ObjectId {
	id := []byte(bson.NewObjectId())
	timeID := []byte(bson.NewObjectIdWithTime(t))
	copy(id[:4], timeID[:4])
	return bson.ObjectId(id)
}
