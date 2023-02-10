package scorecard

import "time"

// Option is a functional option that configures the scorecard API client
type Option func(c *Client)

// WithTimeout is a functional option that configures the timeout duration for
// HTTP requests
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}
