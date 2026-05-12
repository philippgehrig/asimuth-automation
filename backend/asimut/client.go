package asimut

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

const timeFormat = "2006-01-02T15:04:05.000-07:00"

// UserInfo holds user account information from Asimut.
type UserInfo struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Surname        string `json:"surname"`
	Username       string `json:"username"`
	BookingHorizon string `json:"booking_horizon"`
}

// Location represents a bookable location in Asimut.
type Location struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	SecondaryName string `json:"secondary_name"`
	Bookable      bool   `json:"bookable"`
	Type          string `json:"type"`
}

// BookingResult contains the result of a booking operation.
type BookingResult struct {
	EventID int    `json:"event_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Client communicates with the Asimut scheduling system.
type Client struct {
	mu         sync.Mutex
	baseURL    string
	email      string
	password   string
	httpClient *http.Client
	loggedIn   bool
	userInfo   *UserInfo
}

// NewClient creates a new Asimut client with cookie support.
func NewClient(baseURL, email, password string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		email:    email,
		password: password,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Login authenticates with the Asimut system using form POST.
func (c *Client) Login() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	form := url.Values{}
	form.Set("authenticate-url", "/public/hfm-freiburg.asimut.net")
	form.Set("authenticate-useraccount", c.email)
	form.Set("authenticate-password", c.password)
	form.Set("authenticate-verification", "ok")

	req, err := http.NewRequest("POST", c.baseURL+"/public/login.php", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", c.baseURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing login request: %w", err)
	}
	defer resp.Body.Close()

	// Verify login by checking heartbeat
	loggedIn, err := c.getHeartbeat()
	if err != nil {
		return fmt.Errorf("verifying login: %w", err)
	}
	if !loggedIn {
		return fmt.Errorf("login failed: invalid credentials")
	}

	c.loggedIn = true
	return nil
}

// LoggedIn returns whether the client is currently authenticated.
func (c *Client) LoggedIn() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loggedIn
}

// GetLocations retrieves all available locations from Asimut.
func (c *Client) GetLocations() ([]Location, error) {
	respBody, err := c.doJSON("GET", "/services/v2/locations", nil)
	if err != nil {
		return nil, fmt.Errorf("getting locations: %w", err)
	}

	response, ok := respBody["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	locationsRaw, ok := response["locations"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected locations format")
	}

	locations := make([]Location, 0, len(locationsRaw))
	for _, lr := range locationsRaw {
		lm, ok := lr.(map[string]interface{})
		if !ok {
			continue
		}
		loc := Location{
			ID:            intFromInterface(lm["id"]),
			Name:          stringFromInterface(lm["name"]),
			SecondaryName: stringFromInterface(lm["secondary_name"]),
			Bookable:      boolFromInterface(lm["bookable"]),
			Type:          stringFromInterface(lm["type"]),
		}
		locations = append(locations, loc)
	}

	return locations, nil
}

// BookRoom books a room for the given time range.
func (c *Client) BookRoom(roomID int, start, end time.Time) (*BookingResult, error) {
	// Get event default template
	eventData, err := c.getEventDefault(roomID, start)
	if err != nil {
		return nil, fmt.Errorf("getting event default: %w", err)
	}

	// Override end time
	eventData["en"] = end.Format(timeFormat)

	// Check booking
	_, err = c.doJSONBody("POST", "/services/v2/event/type=check", eventData)
	if err != nil {
		return nil, fmt.Errorf("checking booking: %w", err)
	}

	// Save booking
	saveResp, err := c.doJSONBody("POST", "/services/v2/event/type=save", eventData)
	if err != nil {
		return nil, fmt.Errorf("saving booking: %w", err)
	}

	response, ok := saveResp["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected save response format")
	}

	eventIDs, ok := response["event_ids"].([]interface{})
	if !ok || len(eventIDs) == 0 {
		return nil, fmt.Errorf("no event ID in save response")
	}

	eventID := intFromInterface(eventIDs[0])

	return &BookingResult{
		EventID: eventID,
		Success: true,
		Message: "Booking created successfully",
	}, nil
}

// ExtendBooking extends an existing booking to a new end time.
func (c *Client) ExtendBooking(eventID int, newEnd time.Time) (*BookingResult, error) {
	// Get existing event
	path := fmt.Sprintf("/services/v2/event/event_id=%d", eventID)
	eventResp, err := c.doJSON("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting event: %w", err)
	}

	response, ok := eventResp["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected event response format")
	}

	event, ok := response["event"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected event format")
	}

	// Update end time
	event["en"] = newEnd.Format(timeFormat)

	// Check extension
	checkPath := fmt.Sprintf("/services/v2/event/event_id=%d;type=check", eventID)
	_, err = c.doJSONBody("PATCH", checkPath, event)
	if err != nil {
		return nil, fmt.Errorf("checking extension: %w", err)
	}

	// Save extension
	savePath := fmt.Sprintf("/services/v2/event/event_id=%d;type=save", eventID)
	_, err = c.doJSONBody("PATCH", savePath, event)
	if err != nil {
		return nil, fmt.Errorf("saving extension: %w", err)
	}

	return &BookingResult{
		EventID: eventID,
		Success: true,
		Message: "Booking extended successfully",
	}, nil
}

// getEventDefault retrieves the default event template for a room.
func (c *Client) getEventDefault(roomID int, start time.Time) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"st": start.Format(timeFormat),
		"ca": 1,
		"rs": []map[string]interface{}{
			{"id": roomID},
		},
	}

	respBody, err := c.doJSONBody("POST", "/services/v2/eventdefault", body)
	if err != nil {
		return nil, fmt.Errorf("getting event default: %w", err)
	}

	response, ok := respBody["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected eventdefault response format")
	}

	eventDefault, ok := response["eventdefault"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected eventdefault format")
	}

	events, ok := eventDefault["events"].([]interface{})
	if !ok || len(events) == 0 {
		return nil, fmt.Errorf("no events in eventdefault response")
	}

	event, ok := events[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected event format in eventdefault")
	}

	return event, nil
}

// getHeartbeat checks if the session is still authenticated.
func (c *Client) getHeartbeat() (bool, error) {
	respBody, err := c.doJSON("GET", "/services/v2/heartbeat/me", nil)
	if err != nil {
		return false, fmt.Errorf("getting heartbeat: %w", err)
	}

	response, ok := respBody["response"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("unexpected heartbeat response format")
	}

	loggedIn, ok := response["loggedin"].(bool)
	if !ok {
		return false, nil
	}

	return loggedIn, nil
}

// doJSON makes an HTTP request and returns the parsed JSON response.
func (c *Client) doJSON(method, path string, body io.Reader) (map[string]interface{}, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", c.baseURL)
	if method == "POST" || method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	return result, nil
}

// doJSONBody makes an HTTP request with a JSON body and returns the parsed response.
func (c *Client) doJSONBody(method, path string, body interface{}) (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	return c.doJSON(method, path, bytes.NewReader(jsonBytes))
}

// Helper functions for type conversion from interface{}.

func intFromInterface(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	default:
		return 0
	}
}

func stringFromInterface(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func boolFromInterface(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
