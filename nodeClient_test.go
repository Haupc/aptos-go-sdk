package aptos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollForTransaction(t *testing.T) {
	t.Parallel()
	// this doesn't need to actually have an aptos-node!
	// API error on every GET is fine, poll for a few milliseconds then return error
	client, err := NewClient(LocalnetConfig)
	require.NoError(t, err)

	start := time.Now()
	err = client.PollForTransactions([]string{"alice", "bob"}, PollTimeout(10*time.Millisecond), PollPeriod(2*time.Millisecond))
	dt := time.Since(start)

	assert.GreaterOrEqual(t, dt, 9*time.Millisecond)
	assert.Less(t, dt, 20*time.Millisecond)
	require.Error(t, err)
}

func TestEventsByHandle(t *testing.T) {
	t.Parallel()
	createMockServer := func(t *testing.T) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				// handle initial request from client
				w.WriteHeader(http.StatusOK)
				return
			}

			assert.Equal(t, "/accounts/0x0/events/0x2/transfer", r.URL.Path)

			start := r.URL.Query().Get("start")
			limit := r.URL.Query().Get("limit")

			var startInt uint64
			var limitInt uint64

			if start != "" {
				startInt, _ = strconv.ParseUint(start, 10, 64)
			}
			if limit != "" {
				limitInt, _ = strconv.ParseUint(limit, 10, 64)
			} else {
				limitInt = 100
			}

			events := make([]map[string]interface{}, 0, limitInt)
			for i := range limitInt {
				events = append(events, map[string]interface{}{
					"type": "0x1::coin::TransferEvent",
					"guid": map[string]interface{}{
						"creation_number": "1",
						"account_address": AccountZero.String(),
					},
					"sequence_number": strconv.FormatUint(startInt+i, 10),
					"data": map[string]interface{}{
						"amount": strconv.FormatUint((startInt+i)*100, 10),
					},
				})
			}

			err := json.NewEncoder(w).Encode(events)
			if err != nil {
				t.Error(err)
				return
			}
		}))
	}

	t.Run("pagination with concurrent fetching", func(t *testing.T) {
		t.Parallel()
		mockServer := createMockServer(t)
		defer mockServer.Close()

		client, err := NewClient(NetworkConfig{
			Name:    "mocknet",
			NodeUrl: mockServer.URL,
		})
		require.NoError(t, err)

		start := uint64(0)
		limit := uint64(150)
		events, err := client.EventsByHandle(
			AccountZero,
			"0x2",
			"transfer",
			&start,
			&limit,
		)

		require.NoError(t, err)
		assert.Len(t, events, 150)
	})

	t.Run("default page size when limit not provided", func(t *testing.T) {
		t.Parallel()
		mockServer := createMockServer(t)
		defer mockServer.Close()

		client, err := NewClient(NetworkConfig{
			Name:    "mocknet",
			NodeUrl: mockServer.URL,
		})
		require.NoError(t, err)

		events, err := client.EventsByHandle(
			AccountZero,
			"0x2",
			"transfer",
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Len(t, events, 100)
		assert.Equal(t, uint64(99), events[99].SequenceNumber)
	})

	t.Run("single page fetch", func(t *testing.T) {
		t.Parallel()
		mockServer := createMockServer(t)
		defer mockServer.Close()

		client, err := NewClient(NetworkConfig{
			Name:    "mocknet",
			NodeUrl: mockServer.URL,
		})
		require.NoError(t, err)

		start := uint64(50)
		limit := uint64(5)
		events, err := client.EventsByHandle(
			AccountZero,
			"0x2",
			"transfer",
			&start,
			&limit,
		)

		require.NoError(t, err)
		assert.Len(t, events, 5)
		assert.Equal(t, uint64(50), events[0].SequenceNumber)
		assert.Equal(t, uint64(54), events[4].SequenceNumber)
	})
}

func TestEventsByCreationNumber(t *testing.T) {
	t.Parallel()
	createMockServer := func(t *testing.T) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				// handle initial request from client
				w.WriteHeader(http.StatusOK)
				return
			}

			assert.Equal(t, "/accounts/0x0/events/123", r.URL.Path)

			start := r.URL.Query().Get("start")
			limit := r.URL.Query().Get("limit")

			var startInt uint64
			var limitInt uint64

			if start != "" {
				startInt, _ = strconv.ParseUint(start, 10, 64)
			}
			if limit != "" {
				limitInt, _ = strconv.ParseUint(limit, 10, 64)
			} else {
				limitInt = 100
			}

			events := make([]map[string]interface{}, 0, limitInt)
			for i := range limitInt {
				events = append(events, map[string]interface{}{
					"type": "0x1::coin::TransferEvent",
					"guid": map[string]interface{}{
						"creation_number": "123",
						"account_address": AccountZero.String(),
					},
					"sequence_number": strconv.FormatUint(startInt+i, 10),
					"data": map[string]interface{}{
						"amount": strconv.FormatUint((startInt+i)*100, 10),
					},
				})
			}

			err := json.NewEncoder(w).Encode(events)
			if err != nil {
				t.Error(err)
				return
			}
		}))
	}

	t.Run("pagination with concurrent fetching", func(t *testing.T) {
		t.Parallel()
		mockServer := createMockServer(t)
		defer mockServer.Close()

		client, err := NewClient(NetworkConfig{
			Name:    "mocknet",
			NodeUrl: mockServer.URL,
		})
		require.NoError(t, err)

		start := uint64(0)
		limit := uint64(150)
		events, err := client.EventsByCreationNumber(
			AccountZero,
			"123",
			&start,
			&limit,
		)

		require.NoError(t, err)
		assert.Len(t, events, 150)
	})

	t.Run("default page size when limit not provided", func(t *testing.T) {
		t.Parallel()
		mockServer := createMockServer(t)
		defer mockServer.Close()

		client, err := NewClient(NetworkConfig{
			Name:    "mocknet",
			NodeUrl: mockServer.URL,
		})
		require.NoError(t, err)

		events, err := client.EventsByCreationNumber(
			AccountZero,
			"123",
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Len(t, events, 100)
		assert.Equal(t, uint64(99), events[99].SequenceNumber)
	})

	t.Run("single page fetch", func(t *testing.T) {
		t.Parallel()
		mockServer := createMockServer(t)
		defer mockServer.Close()

		client, err := NewClient(NetworkConfig{
			Name:    "mocknet",
			NodeUrl: mockServer.URL,
		})
		require.NoError(t, err)

		start := uint64(50)
		limit := uint64(5)
		events, err := client.EventsByCreationNumber(
			AccountZero,
			"123",
			&start,
			&limit,
		)

		require.NoError(t, err)
		assert.Len(t, events, 5)
		assert.Equal(t, uint64(50), events[0].SequenceNumber)
		assert.Equal(t, uint64(54), events[4].SequenceNumber)
	})
}

func TestNodeClient_ViewWithResponse(t *testing.T) {
	t.Parallel()

	// Mock /view endpoint to return the provided JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Some clients may probe root; return OK
			w.WriteHeader(http.StatusOK)
			return
		}

		assert.Equal(t, "/view", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, ContentTypeAptosViewFunctionBcs, r.Header.Get("Content-Type"))

		type innerObj struct {
			Inner string `json:"inner"`
		}
		resp := []any{[]innerObj{
			{Inner: "0xa"},
			{Inner: "0xa0d9d647c5737a5aed08d2cfeb39c31cf901d44bc4aa024eaa7e5e68b804e011"},
			{Inner: "0xaef6a8c3182e076db72d64324617114cacf9a52f28325edc10b483f7f05da0e7"},
		}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	client, err := NewClient(NetworkConfig{
		Name:    "mocknet",
		NodeUrl: mockServer.URL,
	})
	require.NoError(t, err)

	// Build a simple view payload (values not checked by server)
	payload := &ViewPayload{
		Module:   ModuleId{Address: AccountOne, Name: "coin"},
		Function: "balance",
		ArgTypes: []TypeTag{AptosCoinTypeTag},
		Args:     [][]byte{AccountOne[:]},
	}

	type innerObj struct {
		Inner string `json:"inner"`
	}
	var out []innerObj
	err = client.nodeClient.ViewWithResponse(&out, payload)
	require.NoError(t, err)

	require.Len(t, out, 3)
	assert.Equal(t, "0xa", out[0].Inner)
	assert.Equal(t, "0xa0d9d647c5737a5aed08d2cfeb39c31cf901d44bc4aa024eaa7e5e68b804e011", out[1].Inner)
	assert.Equal(t, "0xaef6a8c3182e076db72d64324617114cacf9a52f28325edc10b483f7f05da0e7", out[2].Inner)
}
