package events

import (
	"encoding/json"
	"strings"
)

type Event struct {
	Bucket struct {
		Name string `json:"name"`
	} `json:"bucket"`
	Object struct {
		Key  string `json:"key"`
	} `json:"object"`
}

type LambdaS3CreateObjectEvent struct {
	Records []struct {
		S3 Event `json:"s3"`
	}
}

func (ev *LambdaS3CreateObjectEvent) Events() []Event {
	events := make([]Event, len(ev.Records))
	for i, x := range ev.Records {
		events[i] = x.S3
	}
	return events
}

type SQSUpdateRepoEvent struct {
	Records []struct {
		Body string `json:"body"`
	}
}

// Events returns all events indexed by their bucket name
func (ev *SQSUpdateRepoEvent) Events() (map[string][]Event, error) {
	mapping := make(map[string][]Event)
	for _, msg := range ev.Records {
		var rec LambdaS3CreateObjectEvent
		err := json.NewDecoder(strings.NewReader(msg.Body)).Decode(&rec)
		if err != nil {
			return nil, err
		}

		events := (&rec).Events()
		for _, e := range events {
			l, ok := mapping[e.Bucket.Name]
			if !ok {
				l = make([]Event, 0, 1)
			}
			mapping[e.Bucket.Name] = append(l, e)
		}
	}
	return mapping, nil
}
