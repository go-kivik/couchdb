package couchdb

type bulkGetError struct {
	ID     string `json:"id"`
	Rev    string `json:"rev"`
	Error  string `json:"error"`
	Reason string `json:"reason"`
}

type bulkResultDoc struct {
	Doc   interface{}  `json:"ok"`
	Error bulkGetError `json:"error"`
}

type bulkResult struct {
	Docs []bulkResultDoc `json:"docs"`
}
