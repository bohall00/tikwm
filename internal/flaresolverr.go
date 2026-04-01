package main

import (
    "bytes"
    "encoding/json"
    "errors"
    "net/http"
    "sync"
    "time"
)

type FlareSolverrClient struct {
    BaseURL    string
    SessionID  string
    HTTPClient *http.Client
    mu         sync.Mutex
    ready      bool
}

type flareRequest struct {
    Cmd        string `json:"cmd"`
    URL        string `json:"url,omitempty"`
    Session    string `json:"session,omitempty"`
    MaxTimeout int    `json:"maxTimeout,omitempty"`
}

type flareResponse struct {
    Status  string `json:"status"`
    Message string `json:"message"`
    Solution struct {
        Response string `json:"response"`
    } `json:"solution"`
}

// Create a new client
func NewFlareSolverrClient(baseURL string) *FlareSolverrClient {
    return &FlareSolverrClient{
        BaseURL:   baseURL,
        SessionID: "tikwm-session",
        HTTPClient: &http.Client{
            Timeout: 90 * time.Second,
        },
    }
}

// EnsureSession initializes ONE persistent session and solves CF only once.
func (f *FlareSolverrClient) EnsureSession() error {
    f.mu.Lock()
    defer f.mu.Unlock()

    if f.ready {
        return nil
    }

    reqBody := flareRequest{
        Cmd:        "request.get",
        URL:        "https://tikwm.com/",
        Session:    f.SessionID,
        MaxTimeout: 60000,
    }

    _, err := f.send(reqBody)
    if err != nil {
        return err
    }

    f.ready = true
    return nil
}

// Perform a request THROUGH the existing session (no re-init)
func (f *FlareSolverrClient) Get(url string) (string, error) {
    if !f.ready {
        if err := f.EnsureSession(); err != nil {
            return "", err
        }
    }

    reqBody := flareRequest{
        Cmd:        "request.get",
        URL:        url,
        Session:    f.SessionID,
        MaxTimeout: 60000,
    }

    resp, err := f.send(reqBody)
    if err != nil {
        return "", err
    }

    return resp.Solution.Response, nil
}

// Internal sender
func (f *FlareSolverrClient) send(payload flareRequest) (*flareResponse, error) {
    body, _ := json.Marshal(payload)

    req, err := http.NewRequest("POST", f.BaseURL+"/v1", bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := f.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result flareResponse
    err = json.NewDecoder(resp.Body).Decode(&result)
    if err != nil {
        return nil, err
    }

    if result.Status != "ok" {
        return nil, errors.New(result.Message)
    }

    return &result, nil
}

// Optional: destroy only when program exits
func (f *FlareSolverrClient) DestroySession() {
    f.mu.Lock()
    defer f.mu.Unlock()

    payload := flareRequest{
        Cmd:     "sessions.destroy",
        Session: f.SessionID,
    }

    body, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", f.BaseURL+"/v1", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    f.HTTPClient.Do(req)
}