// Package client provides convenience functions for invoking API operations
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mishudark/errors"
	"github.com/oslokommune/okctl/pkg/api"
)

const (
	targetVpcs           = "vpcs/"
	targetClusters       = "clusters/"
	targetClusterConfigs = "clusterconfigs/"
)

// Cluster client API calls
type Cluster interface {
	CreateCluster(opts *api.ClusterCreateOpts) error
	DeleteCluster(opts *api.ClusterDeleteOpts) error
}

// ClusterConfig client API calls
type ClusterConfig interface {
	CreateClusterConfig(opts *api.CreateClusterConfigOpts) error
}

// Vpc client API calls
type Vpc interface {
	CreateVpc(opts *api.CreateVpcOpts) error
	DeleteVpc(opts *api.DeleteVpcOpts) error
}

// Client stores state for invoking API operations
type Client struct {
	BaseURL  string
	Client   *http.Client
	Progress io.Writer
}

// New returns a client that wraps the common API operations
func New(progress io.Writer, serverURL string) *Client {
	return &Client{
		Progress: progress,
		BaseURL:  serverURL,
		Client:   &http.Client{},
	}
}

// CreateClusterConfig invokes the cluster config create operation
func (c *Client) CreateClusterConfig(opts *api.CreateClusterConfigOpts) error {
	return c.DoPost(targetClusterConfigs, opts)
}

// CreateVpc invokes the vpc create operation
func (c *Client) CreateVpc(opts *api.CreateVpcOpts) error {
	return c.DoPost(targetVpcs, opts)
}

// DeleteVpc invokes the vpc delete operation
func (c *Client) DeleteVpc(opts *api.DeleteVpcOpts) error {
	return c.DoDelete(targetVpcs, opts)
}

// CreateCluster invokes the cluster create operation
func (c *Client) CreateCluster(opts *api.ClusterCreateOpts) error {
	return c.DoPost(targetClusters, opts)
}

// DeleteCluster invokes the cluster delete operation
func (c *Client) DeleteCluster(opts *api.ClusterDeleteOpts) error {
	return c.DoDelete(targetClusters, opts)
}

// DoPost sends a POST request to the given endpoint
func (c *Client) DoPost(endpoint string, body interface{}) error {
	return c.Do(http.MethodPost, endpoint, body)
}

// DoDelete sends a DELETE request to the given endpoint
func (c *Client) DoDelete(endpoint string, body interface{}) error {
	return c.Do(http.MethodDelete, endpoint, body)
}

// Do performs the request
func (c *Client) Do(method, endpoint string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return errors.E(err, pretty("failed to marshal data for", method, endpoint))
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, endpoint), bytes.NewReader(data))
	if err != nil {
		return errors.E(err, pretty("failed to create request for", method, endpoint))
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return errors.E(err, pretty("request failed for", method, endpoint))
	}

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.E(err, pretty("failed to read response for", method, endpoint))
	}

	defer func() {
		err = resp.Body.Close()
	}()

	_, err = io.Copy(c.Progress, strings.NewReader(string(out)))
	if err != nil {
		return errors.E(err, pretty("failed to write progress for", method, endpoint))
	}

	return nil
}

func pretty(msg, method, endpoint string) string {
	return fmt.Sprintf("%s: %s, %s", msg, method, endpoint)
}