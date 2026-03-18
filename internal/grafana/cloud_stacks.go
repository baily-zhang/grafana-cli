package grafana

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) CloudStackDatasources(ctx context.Context, stack string) (any, error) {
	if strings.TrimSpace(c.cfg.CloudURL) == "" {
		return nil, ErrMissingBaseURL
	}
	u, err := joinURL(c.cfg.CloudURL, "/api/instances/"+url.PathEscape(strings.TrimSpace(stack))+"/datasources", nil)
	if err != nil {
		return nil, err
	}
	return c.requestJSON(ctx, http.MethodGet, u, nil)
}

func (c *Client) CloudStackConnections(ctx context.Context, stack string) (any, error) {
	if strings.TrimSpace(c.cfg.CloudURL) == "" {
		return nil, ErrMissingBaseURL
	}
	u, err := joinURL(c.cfg.CloudURL, "/api/instances/"+url.PathEscape(strings.TrimSpace(stack))+"/connections", nil)
	if err != nil {
		return nil, err
	}
	return c.requestJSON(ctx, http.MethodGet, u, nil)
}

func (c *Client) CloudStackPlugins(ctx context.Context, stack string) (any, error) {
	if strings.TrimSpace(c.cfg.CloudURL) == "" {
		return nil, ErrMissingBaseURL
	}
	u, err := joinURL(c.cfg.CloudURL, "/api/instances/"+url.PathEscape(strings.TrimSpace(stack))+"/plugins", nil)
	if err != nil {
		return nil, err
	}
	return c.requestJSON(ctx, http.MethodGet, u, nil)
}

func (c *Client) CloudStackPlugin(ctx context.Context, stack, plugin string) (any, error) {
	if strings.TrimSpace(c.cfg.CloudURL) == "" {
		return nil, ErrMissingBaseURL
	}
	u, err := joinURL(c.cfg.CloudURL, "/api/instances/"+url.PathEscape(strings.TrimSpace(stack))+"/plugins/"+url.PathEscape(strings.TrimSpace(plugin)), nil)
	if err != nil {
		return nil, err
	}
	return c.requestJSON(ctx, http.MethodGet, u, nil)
}
