// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.
package internal_test

import (
	"context"
	"testing"
	"time"

	. "github.com/gomoni/gonix/internal"
	"github.com/stretchr/testify/require"
)

func TestPMap(t *testing.T) {
	t.Parallel()
	f := func(_ context.Context, i int) (int, error) {
		t.Logf("TestPMAP.f(%d)", i)
		time.Sleep(time.Duration(i) * time.Millisecond)
		return i + 42, nil
	}
	ctx := context.TODO()

	start := time.Now()
	ret, err := PMap(ctx, 1, []int{10, 20, 50, 100, 200, 500}, f)
	stop := time.Now()
	require.NoError(t, err)
	require.Equal(t, []int{52, 62, 92, 142, 242, 542}, ret)
	duration1 := stop.Sub(start)

	start = time.Now()
	ret, err = PMap(ctx, 3, []int{10, 20, 50, 100, 200, 500}, f)
	stop = time.Now()
	require.NoError(t, err)
	require.Equal(t, []int{52, 62, 92, 142, 242, 542}, ret)
	duration2 := stop.Sub(start)

	// this may be reliable enough in all scenarios (CI vs local machine)
	// the seq case shall be 10+20+50+100+200+500=880ms long
	// the parallel one shall take slightly more than 500ms
	require.GreaterOrEqual(t, duration1, duration2)
}
