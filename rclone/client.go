package rclone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Client stores host/client information and its credentials.
type Client struct {
	Host string
	URI  *url.URL

	client     *http.Client
	user, pass string
}

// Response stores the response obtained after a client request.
type Response struct {
	Body io.ReadCloser
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36"

var (
	currentHost string

	sessionLock sync.Mutex
	session     map[string]*Client

	clientCtx    context.Context
	clientCancel context.CancelFunc
)

// SendRequest sends a request to the rclone host and returns a response.
func (c *Client) SendRequest(command map[string]interface{}, endpoint string, ctx ...context.Context) (Response, error) {
	if ctx == nil {
		ctx = append(ctx, clientContext(false))
	}

	commandBytes, err := json.Marshal(&command)
	if err != nil {
		return Response{}, err
	}

	req, err := http.NewRequestWithContext(
		ctx[0], http.MethodPost,
		c.Host+endpoint, bytes.NewReader(commandBytes),
	)
	if err != nil {
		return Response{}, err
	}

	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return Response{}, err
	}

	if res.StatusCode == 401 {
		return Response{}, fmt.Errorf("Unauthorized")
	}

	return Response{res.Body}, nil
}

// Hostname returns the client's hostname.
func (c *Client) Hostname() string {
	return c.URI.Scheme + "://" + c.URI.Host
}

// Decode unmarshals the json response into the provided data.
func (r *Response) Decode(v interface{}) error {
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(v)
}

// SetupClient sets up the client.
func SetupClient(host, user, pass string) error {
	sessionLock.Lock()
	defer sessionLock.Unlock()

	var client *Client

	u, err := url.Parse(host)
	if err != nil {
		return err
	}

	if c, err := GetClient(host, struct{}{}); err == nil {
		client = c

		goto LoadClient
	}

	u.User = url.UserPassword(user, pass)

	client = &Client{
		URI: u,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},

		user: user,
		pass: pass,
	}

LoadClient:
	host = u.Scheme + "://"
	if user != "" {
		host += user + ":" + pass + "@"
	}
	host += u.Hostname()
	if u.Port() != "" {
		host += ":" + u.Port()
	}

	client.Host = host

	if err := testClient(client); err != nil {
		return err
	}

	currentHost = client.Host

	if session == nil {
		session = make(map[string]*Client)
	}

	session[currentHost] = client

	return err
}

// SetSession sets the current session.
func SetSession(host string) {
	currentHost = host
}

// GetSessions gets the running sessions.
func GetSessions() []string {
	sessionLock.Lock()
	defer sessionLock.Unlock()

	var uris []string

	for s := range session {
		uris = append(uris, s)
	}

	return uris
}

// GetClient gets the client which corresponds to the provided host.
func GetClient(host string, nolock ...struct{}) (*Client, error) {
	if nolock == nil {
		sessionLock.Lock()
		defer sessionLock.Unlock()
	}

	client, ok := session[host]
	if !ok {
		return nil, fmt.Errorf("No client found")
	}

	return client, nil
}

// GetCurrentClient gets the client associated with the current host.
func GetCurrentClient() (*Client, error) {
	return GetClient(currentHost)
}

// DialServer checks whether the host is reachable.
func DialServer() error {
	var address string

	client, err := GetCurrentClient()
	if err != nil {
		return err
	}

	address = client.URI.Hostname() + ":"
	if addrURI := client.URI; addrURI.Port() == "" {
		address += addrURI.Scheme
	} else {
		address += addrURI.Port()
	}

	_, err = net.DialTimeout("tcp", address, 1*time.Second)

	return err
}

// SendCommand sends a command to the rclone host and returns a response.
// This is a blocking call.
func SendCommand(command map[string]interface{}, endpoint string, ctx ...context.Context) (Response, error) {
	client, err := GetCurrentClient()
	if err != nil {
		return Response{}, err
	}

	return client.SendRequest(command, endpoint, ctx...)
}

// SendCommandAsync asynchronously sends a command to rclone and returns the job information
// for the running command.
func SendCommandAsync(
	jobType, jobDesc string,
	command map[string]interface{}, endpoint string,
	noqueue ...struct{},
) (*Job, error) {
	var jobID map[string]interface{}

	command["_async"] = true

	response, err := SendCommand(command, endpoint)
	if err != nil {
		return nil, err
	}

	err = response.Decode(&jobID)
	if err != nil {
		return nil, err
	}

	if jobError, ok := jobID["error"]; ok {
		if respErr, ok := jobError.(string); ok {
			return nil, fmt.Errorf(respErr)
		}
	}

	id, ok := jobID["jobid"].(float64)
	if !ok {
		return nil, fmt.Errorf("Cannot get job ID from rclone")
	}

	job := NewJob(jobType, jobDesc, int64(id))
	if noqueue != nil {
		return job, nil
	}

	return AddJobToQueue(job), nil
}

// GetClientContext returns the client context.
func GetClientContext() context.Context {
	return clientContext(false)
}

// CancelClientContext cancels the client context.
func CancelClientContext() {
	clientContext(true)
}

// testClient tests the client's credentials and connectivity to the rclone host.
func testClient(client *Client) error {
	_, err := client.SendRequest(map[string]interface{}{}, "/rc/noopauth")
	return err
}

// clientContext either returns the client context, or renews the context.
func clientContext(cancel bool) context.Context {
	if cancel && clientCancel != nil {
		clientCancel()
	}

	if clientCtx == nil || cancel {
		clientCtx, clientCancel = context.WithCancel(context.Background())
	}

	return clientCtx
}
