// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.
package internal

import (
	"context"
	"errors"

	"golang.org/x/sync/semaphore"
)

// an experiments with a paralelization of work
// the must is that task must maintain the order they were submitted

type MapFunc[T any, U any] func(context.Context, T) (U, error)

func PMap[T any, U any](ctx context.Context, limit uint, slice []T, mapFunc MapFunc[T, U]) ([]U, error) {
	retu := make([]U, len(slice))
	errs := make([]error, len(slice))

	sem := semaphore.NewWeighted(int64(limit))

	for idx, input := range slice {

		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}

		go func(ctx context.Context, idx int, input T, results []U, errors []error) {
			defer sem.Release(1)

			result, err := mapFunc(ctx, input)
			if err != nil {
				errs[idx] = err
			}
			results[idx] = result
		}(ctx, idx, input, retu, errs)
	}

	if err := sem.Acquire(ctx, int64(limit)); err != nil {
		return nil, err
	}

	return retu, errors.Join(errs...)
}
