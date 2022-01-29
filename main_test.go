package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

//go:embed test/event-list.json
var events []byte

//go:embed test/event-bad-name-len.json
var eventBadNameLength []byte

//go:embed test/event-bad-date.json
var eventBadDate []byte

//go:embed test/event-no-venue.json
var eventNoVenue []byte

//go:embed test/event-ok.json
var eventOk []byte

func TestEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("TestListEvents", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		// Call the request handler
		listEvents(c)

		assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

		bb, err := io.ReadAll(rec.Body)
		assert.NoError(t, err)

		assert.JSONEq(t, string(events), string(bb))
	})
}

func saveEventInternal(requestBody []byte) (*http.Response, int) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	r := bytes.NewReader(requestBody)

	c.Request = httptest.NewRequest(http.MethodPut, "/", r)

	// Call the request handler
	saveEvent(c)

	return rec.Result(), rec.Result().StatusCode
}

func TestSaveEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Parallel()

	t.Run("TestHappyCase", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		r := bytes.NewReader(eventOk)

		c.Request = httptest.NewRequest(http.MethodPut, "/", r)

		// Call the request handler
		saveEvent(c)

		assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)

		// Test for generated event ID
		bb, err := io.ReadAll(rec.Body)
		assert.NoError(t, err)

		var event gin.H
		err = json.Unmarshal(bb, &event)

		assert.NoError(t, err)
		assert.NotEmpty(t, event["id"])
	})

	t.Run("TestBadRequests", func(t *testing.T) {
		var testTable = []struct {
			desc string
			in   []byte
		}{
			{"BadNameLength", eventBadNameLength},
			{"NoVenue", eventNoVenue},
			{"BadDate", eventBadDate},
		}

		for _, tt := range testTable {
			t.Run(tt.desc, func(t *testing.T) {
				out, statusCode := saveEventInternal(tt.in)
				defer out.Body.Close()

				assert.Equal(t, http.StatusBadRequest, statusCode)

				bb, err := io.ReadAll(out.Body)

				assert.NoError(t, err)

				var errMsg gin.H
				err = json.Unmarshal(bb, &errMsg)

				assert.NoError(t, err)

				assert.NotEmptyf(t, errMsg["message"], "the error response does not contain a valid message")
			})

		}
	})
}
