package couchdb

import (
	"testing"

	"github.com/flimzy/diff"

	"github.com/go-kivik/kivik"
)

type attStruct struct {
	Attachments kivik.Attachments `json:"_attachments"`
}

type attPtrStruct struct {
	Attachments *kivik.Attachments `json:"_attachments"`
}

type wrongTypeStruct struct {
	Attachments string `json:"_attachments"`
}

type wrongTagStruct struct {
	Attachments kivik.Attachments `json:"foo"`
}

func TestExtractAttachments(t *testing.T) {
	tests := []struct {
		name string
		doc  interface{}

		expected *kivik.Attachments
		ok       bool
	}{
		{
			name:     "no attachments",
			doc:      map[string]interface{}{"foo": "bar"},
			expected: nil,
			ok:       false,
		},
		{
			name: "in map",
			doc: map[string]interface{}{"_attachments": kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain", Content: Body("test content")},
			}},
			expected: &kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain"},
			},
			ok: true,
		},
		{
			name:     "wrong type in map",
			doc:      map[string]interface{}{"_attachments": "oink"},
			expected: nil,
			ok:       false,
		},
		{
			name:     "non standard map, non struct",
			doc:      map[string]string{"foo": "bar"},
			expected: nil,
			ok:       false,
		},
		{
			name: "attachments in struct",
			doc: attStruct{
				Attachments: kivik.Attachments{
					"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain", Content: Body("test content")},
				},
			},
			expected: &kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain"},
			},
			ok: true,
		},
		{
			name: "pointer to attachments in struct",
			doc: attPtrStruct{
				Attachments: &kivik.Attachments{
					"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain", Content: Body("test content")},
				},
			},
			expected: &kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain"},
			},
			ok: true,
		},
		{
			name: "wrong type of struct",
			doc: wrongTypeStruct{
				Attachments: "foo",
			},
			expected: nil,
			ok:       false,
		},
		{
			name: "wrong json tag",
			doc: wrongTagStruct{
				Attachments: kivik.Attachments{
					"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain", Content: Body("test content")},
				},
			},
			expected: nil,
			ok:       false,
		},
		{
			name: "pointer to struct with attachments",
			doc: &attStruct{
				Attachments: kivik.Attachments{
					"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain", Content: Body("test content")},
				},
			},
			expected: &kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain"},
			},
			ok: true,
		},
		{
			name: "pointer to map with attachments",
			doc: &(map[string]interface{}{"_attachments": kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain", Content: Body("test content")},
			}}),
			expected: &kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain"},
			},
			ok: true,
		},
		{
			name: "pointer in map",
			doc: map[string]interface{}{"_attachments": &kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain", Content: Body("test content")},
			}},
			expected: &kivik.Attachments{
				"foo.txt": &kivik.Attachment{Filename: "foo.txt", ContentType: "text/plain"},
			},
			ok: true,
		},
		{
			name: "nil doc",
			doc:  nil,
			ok:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, ok := extractAttachments(test.doc)
			if ok != test.ok {
				t.Errorf("Unexpected OK: %v", ok)
			}
			if result != nil {
				for _, att := range *result {
					att.Content = nil
				}
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}
