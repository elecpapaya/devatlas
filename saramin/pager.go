package saramin

import (
	"context"
	"errors"
	"strconv"
)

const DefaultPageSize = 110

type PageHandler func(*JobSearchResponse) error

var ErrStopPaging = errors.New("saramin: stop paging")

func (c *Client) JobSearchPages(ctx context.Context, params JobSearchParams, handler PageHandler) error {
	if handler == nil {
		return nil
	}
	if params.Count <= 0 {
		params.Count = DefaultPageSize
	}
	if params.Start < 0 {
		params.Start = 0
	}

	start := params.Start
	for {
		params.Start = start
		resp, err := c.JobSearch(ctx, params)
		if err != nil {
			return err
		}
		if err := handler(resp); err != nil {
			if errors.Is(err, ErrStopPaging) {
				return nil
			}
			return err
		}

		pageCount := len(resp.Jobs.Job)
		if pageCount == 0 {
			return nil
		}

		if total, ok := parseTotal(resp.Jobs.Total); ok {
			start += params.Count
			if start >= total {
				return nil
			}
			continue
		}

		if pageCount < params.Count {
			return nil
		}
		start += params.Count
	}
}

func parseTotal(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	total, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return total, true
}
