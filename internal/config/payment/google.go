package paymentConfig

import "encoding/json"

type Google struct {
	PackageName        string          `json:"package_name"`
	Credentials        json.RawMessage `json:"credentials"`
	PubSubSubscription string          `json:"pubsub_subscription"`
}
