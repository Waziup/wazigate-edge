package edge

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type Message struct {
	ID              bson.ObjectId `json:"id" bson:"_id"`
	Text            string        `json:"text" bson:"text"`
	Title           string        `json:"title" bson:"title"`
	Severity        string        `json:"severity" bson:"severity"`
	HRef            string        `json:"href" bson:"href"`
	Target          string        `json:"target" bson:"target"`
	Time            time.Time     `json:"time" bson:"time"`
	DisplayDuration int           `json:"displayDuration" bson:"displayDuration"`
}

const messageDefaultLifetime = time.Hour * 24 * 3
const messageDefaultDisplayDuration = 5000 // time.Millisecond

func PostMessage(m *Message) error {

	if m.DisplayDuration == 0 {
		m.DisplayDuration = messageDefaultDisplayDuration
	}

	if m.Time == noTime {
		m.Time = time.Now()
	}

	if m.ID == "" {
		m.ID = newID(m.Time)
	}

	if err := dbMessages.Remove(bson.M{
		"_id": bson.M{
			"$lt": bson.NewObjectIdWithTime(time.Now().Add(-messageDefaultLifetime)),
		},
	}); err != nil {
		log.Printf("[ERR  ] Can not delete old messages: %v", err)
	}

	err := dbMessages.Insert(m)
	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}

	return nil
}

type MessagesQuery struct {
	Limit int64
	From  time.Time
	To    time.Time
}

// Parse reads url.Values into the DevicesQuery.
func (query *MessagesQuery) Parse(req *http.Request) string {
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

	return ""
}

func GetMessages(query *MessagesQuery) *messagesIterator {
	m := bson.M{}
	var noTime = time.Time{}
	if query.From != noTime || query.To != noTime {
		mid := bson.M{}
		m["_id"] = mid
		if query.From != noTime {
			mid["$gte"] = bson.NewObjectIdWithTime(query.From)
		}
		if query.To != noTime {
			query.To.Add(time.Second)
			mid["$lt"] = bson.NewObjectIdWithTime(query.To)
		}
	}
	q := dbMessages.Find(m).Sort("_id")
	if query.Limit != 0 {
		q.Limit(int(query.Limit))
	}

	return &messagesIterator{
		dbIter: q.Iter(),
	}
}

type messagesIterator struct {
	dbIter *mgo.Iter
	msg    Message
}

func (iter *messagesIterator) Next() (*Message, error) {
	if iter.dbIter.Next(&iter.msg) {
		return &iter.msg, iter.dbIter.Err()
	}
	return nil, io.EOF
}

func (iter *messagesIterator) Close() error {
	return iter.dbIter.Close()
}
